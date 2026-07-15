package cache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	putEmailCodeScript = redis.NewScript(`
redis.call("SET", KEYS[1], ARGV[1], "PX", ARGV[2])
redis.call("DEL", KEYS[2])
return 1`)
	verifyEmailCodeScript = redis.NewScript(`
local current = redis.call("GET", KEYS[1])
if not current then
  return 0
end
if current == ARGV[1] then
  redis.call("DEL", KEYS[1], KEYS[2])
  return 1
end
local failures = redis.call("INCR", KEYS[2])
local ttl = redis.call("PTTL", KEYS[1])
if failures == 1 and ttl > 0 then
  redis.call("PEXPIRE", KEYS[2], ttl)
end
if failures >= tonumber(ARGV[2]) then
  redis.call("DEL", KEYS[1], KEYS[2])
end
return 0`)
	deleteEmailCodeScript = redis.NewScript(`
if redis.call("GET", KEYS[1]) == ARGV[1] then
  redis.call("DEL", KEYS[1], KEYS[2])
  return 1
end
return 0`)
	acquireCooldownScript = redis.NewScript(`
local created = redis.call("SET", KEYS[1], ARGV[1], "PX", ARGV[2], "NX")
if created then
  return {1, tonumber(ARGV[2])}
end
local ttl = redis.call("PTTL", KEYS[1])
if ttl < 0 then ttl = 0 end
return {0, ttl}`)
	releaseCooldownScript = redis.NewScript(`
if redis.call("GET", KEYS[1]) == ARGV[1] then
  return redis.call("DEL", KEYS[1])
end
return 0`)
	fixedWindowScript = redis.NewScript(`
local count = redis.call("INCR", KEYS[1])
if count == 1 then
  redis.call("PEXPIRE", KEYS[1], ARGV[2])
end
local ttl = redis.call("PTTL", KEYS[1])
if ttl < 0 then ttl = 0 end
if count <= tonumber(ARGV[1]) then
  return {1, ttl}
end
return {0, ttl}`)
)

// DragonflyAuthCodeStore implements the auth-code primitives against the
// Redis protocol supported by DragonFly.
type DragonflyAuthCodeStore struct {
	client *redis.Client
	prefix string
}

func NewDragonflyAuthCodeStore(redisURL, prefix string) (*DragonflyAuthCodeStore, error) {
	options, err := redis.ParseURL(strings.TrimSpace(redisURL))
	if err != nil {
		return nil, fmt.Errorf("parse REDIS_URL: %w", err)
	}
	prefix = strings.Trim(strings.TrimSpace(prefix), ":")
	if prefix == "" {
		prefix = "fundlive"
	}
	return &DragonflyAuthCodeStore{
		client: redis.NewClient(options),
		prefix: prefix,
	}, nil
}

func (s *DragonflyAuthCodeStore) Ping(ctx context.Context) error {
	if s == nil || s.client == nil {
		return fmt.Errorf("dragonfly client is not configured")
	}
	return s.client.Ping(ctx).Err()
}

func (s *DragonflyAuthCodeStore) Close() error {
	if s == nil || s.client == nil {
		return nil
	}
	return s.client.Close()
}

func (s *DragonflyAuthCodeStore) PutEmailCode(ctx context.Context, email, digest string, ttl time.Duration) error {
	if ttl <= 0 {
		return fmt.Errorf("email code TTL must be positive")
	}
	return putEmailCodeScript.Run(
		ctx,
		s.client,
		[]string{s.emailCodeKey(email), s.emailFailureKey(email)},
		digest,
		ttl.Milliseconds(),
	).Err()
}

func (s *DragonflyAuthCodeStore) VerifyEmailCode(ctx context.Context, email, digest string, maxFailures int) (bool, error) {
	if maxFailures <= 0 {
		return false, fmt.Errorf("max email code failures must be positive")
	}
	result, err := verifyEmailCodeScript.Run(
		ctx,
		s.client,
		[]string{s.emailCodeKey(email), s.emailFailureKey(email)},
		digest,
		maxFailures,
	).Int64()
	if err != nil {
		return false, err
	}
	return result == 1, nil
}

func (s *DragonflyAuthCodeStore) DeleteEmailCodeIfMatches(ctx context.Context, email, digest string) error {
	return deleteEmailCodeScript.Run(
		ctx,
		s.client,
		[]string{s.emailCodeKey(email), s.emailFailureKey(email)},
		digest,
	).Err()
}

func (s *DragonflyAuthCodeStore) AcquireCooldown(ctx context.Context, scope, token string, ttl time.Duration) (bool, time.Duration, error) {
	if ttl <= 0 {
		return true, 0, nil
	}
	result, err := acquireCooldownScript.Run(
		ctx,
		s.client,
		[]string{s.cooldownKey(scope)},
		token,
		ttl.Milliseconds(),
	).Slice()
	if err != nil {
		return false, 0, err
	}
	if len(result) != 2 {
		return false, 0, fmt.Errorf("unexpected cooldown response length %d", len(result))
	}
	created, err := redisInt64(result[0])
	if err != nil {
		return false, 0, fmt.Errorf("decode cooldown status: %w", err)
	}
	retryMilliseconds, err := redisInt64(result[1])
	if err != nil {
		return false, 0, fmt.Errorf("decode cooldown TTL: %w", err)
	}
	return created == 1, time.Duration(retryMilliseconds) * time.Millisecond, nil
}

func (s *DragonflyAuthCodeStore) ReleaseCooldownIfMatches(ctx context.Context, scope, token string) error {
	return releaseCooldownScript.Run(ctx, s.client, []string{s.cooldownKey(scope)}, token).Err()
}

func (s *DragonflyAuthCodeStore) AllowFixedWindow(ctx context.Context, scope string, limit int, window time.Duration) (bool, time.Duration, error) {
	if limit <= 0 || window <= 0 {
		return false, 0, fmt.Errorf("fixed-window limit and duration must be positive")
	}
	now := time.Now().UTC()
	bucket := now.UnixNano() / window.Nanoseconds()
	windowRemaining := window - time.Duration(now.UnixNano()%window.Nanoseconds())
	result, err := fixedWindowScript.Run(
		ctx,
		s.client,
		[]string{s.rateKey(scope, bucket)},
		limit,
		windowRemaining.Milliseconds(),
	).Slice()
	if err != nil {
		return false, 0, err
	}
	if len(result) != 2 {
		return false, 0, fmt.Errorf("unexpected rate-limit response length %d", len(result))
	}
	allowed, err := redisInt64(result[0])
	if err != nil {
		return false, 0, fmt.Errorf("decode rate-limit status: %w", err)
	}
	retryMilliseconds, err := redisInt64(result[1])
	if err != nil {
		return false, 0, fmt.Errorf("decode rate-limit TTL: %w", err)
	}
	return allowed == 1, time.Duration(retryMilliseconds) * time.Millisecond, nil
}

func (s *DragonflyAuthCodeStore) emailCodeKey(email string) string {
	return s.key("auth", "email-code", scopeHash(email))
}

func (s *DragonflyAuthCodeStore) emailFailureKey(email string) string {
	return s.key("auth", "email-code-failures", scopeHash(email))
}

func (s *DragonflyAuthCodeStore) cooldownKey(scope string) string {
	return s.key("auth", "cooldown", scopeHash(scope))
}

func (s *DragonflyAuthCodeStore) rateKey(scope string, bucket int64) string {
	return s.key("auth", "rate", scopeHash(scope), fmt.Sprintf("%d", bucket))
}

func (s *DragonflyAuthCodeStore) key(parts ...string) string {
	return s.prefix + ":" + strings.Join(parts, ":")
}

func scopeHash(value string) string {
	sum := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(value))))
	return hex.EncodeToString(sum[:])
}

func redisInt64(value interface{}) (int64, error) {
	switch typed := value.(type) {
	case int64:
		return typed, nil
	case string:
		var parsed int64
		_, err := fmt.Sscan(typed, &parsed)
		return parsed, err
	default:
		return 0, fmt.Errorf("unexpected Redis integer type %T", value)
	}
}
