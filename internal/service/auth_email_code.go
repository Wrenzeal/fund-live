package service

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/RomaticDOG/fund/internal/domain"
)

const (
	emailCodeAvailabilityTimeout = 750 * time.Millisecond
	emailCodeSecretMinLength     = 32
)

// AuthCodeStore persists one-time codes, cooldowns, and fixed-window counters.
// DragonFly is the production implementation; tests use a deterministic fake.
type AuthCodeStore interface {
	Ping(ctx context.Context) error
	PutEmailCode(ctx context.Context, email, digest string, ttl time.Duration) error
	VerifyEmailCode(ctx context.Context, email, digest string, maxFailures int) (bool, error)
	DeleteEmailCodeIfMatches(ctx context.Context, email, digest string) error
	AcquireCooldown(ctx context.Context, scope, token string, ttl time.Duration) (bool, time.Duration, error)
	ReleaseCooldownIfMatches(ctx context.Context, scope, token string) error
	AllowFixedWindow(ctx context.Context, scope string, limit int, window time.Duration) (bool, time.Duration, error)
}

// VerificationCooldownError carries retry metadata for a recently sent code.
type VerificationCooldownError struct {
	RetryAfter time.Duration
}

func (e *VerificationCooldownError) Error() string { return ErrVerificationCodeCooldown.Error() }
func (e *VerificationCooldownError) Unwrap() error { return ErrVerificationCodeCooldown }

// AuthRateLimitError carries retry metadata for email/IP send limits.
type AuthRateLimitError struct {
	RetryAfter time.Duration
}

func (e *AuthRateLimitError) Error() string { return ErrAuthRateLimited.Error() }
func (e *AuthRateLimitError) Unwrap() error { return ErrAuthRateLimited }

// SetEmailCodeDependencies enables email-code login when configuration and
// both runtime dependencies are available.
func (s *AuthService) SetEmailCodeDependencies(store AuthCodeStore, sender EmailSender) {
	s.emailCodeStore = store
	s.emailSender = sender
}

// EmailCodeLoginAvailable reports whether the optional email-code path can be
// used without affecting password or Google authentication availability.
func (s *AuthService) EmailCodeLoginAvailable(ctx context.Context) bool {
	if s == nil || !s.config.EmailCodeEnabled || s.emailCodeStore == nil || s.emailSender == nil {
		return false
	}
	if len(strings.TrimSpace(s.config.EmailCodeSecret)) < emailCodeSecretMinLength {
		return false
	}
	pingCtx, cancel := context.WithTimeout(ctx, emailCodeAvailabilityTimeout)
	defer cancel()
	return s.emailCodeStore.Ping(pingCtx) == nil
}

// StartEmailCodeLogin creates and sends a one-time email challenge without
// revealing whether the normalized email already belongs to an account.
func (s *AuthService) StartEmailCodeLogin(ctx context.Context, input domain.EmailCodeStartInput, meta domain.SessionMetadata) (*domain.EmailCodeStartResult, error) {
	email, err := normalizeEmail(input.Email)
	if err != nil {
		return nil, err
	}
	if !s.EmailCodeLoginAvailable(ctx) {
		return nil, ErrEmailCodeLoginUnavailable
	}

	code, err := randomNumericCode(6)
	if err != nil {
		return nil, fmt.Errorf("generate email verification code: %w", err)
	}
	digest := s.emailCodeDigest(email, code)
	cooldownScope := "email:" + email
	acquired, retryAfter, err := s.emailCodeStore.AcquireCooldown(ctx, cooldownScope, digest, s.config.EmailResendCooldown)
	if err != nil {
		return nil, fmt.Errorf("acquire email verification cooldown: %w", err)
	}
	if !acquired {
		return nil, &VerificationCooldownError{RetryAfter: retryAfter}
	}
	releaseCooldown := func() {
		_ = s.emailCodeStore.ReleaseCooldownIfMatches(context.WithoutCancel(ctx), cooldownScope, digest)
	}

	allowed, retryAfter, err := s.emailCodeStore.AllowFixedWindow(ctx, "email:"+email, s.config.MaxEmailSendsPerHour, time.Hour)
	if err != nil {
		releaseCooldown()
		return nil, fmt.Errorf("check email verification limit: %w", err)
	}
	if !allowed {
		releaseCooldown()
		return nil, &AuthRateLimitError{RetryAfter: retryAfter}
	}
	clientIP := strings.TrimSpace(meta.IPAddress)
	if clientIP != "" {
		allowed, retryAfter, err = s.emailCodeStore.AllowFixedWindow(ctx, "ip:"+clientIP, s.config.MaxIPEmailSendsPerHour, time.Hour)
		if err != nil {
			releaseCooldown()
			return nil, fmt.Errorf("check IP verification limit: %w", err)
		}
		if !allowed {
			releaseCooldown()
			return nil, &AuthRateLimitError{RetryAfter: retryAfter}
		}
	}

	if err := s.emailCodeStore.PutEmailCode(ctx, email, digest, s.config.EmailCodeTTL); err != nil {
		releaseCooldown()
		return nil, fmt.Errorf("store email verification code: %w", err)
	}
	if err := s.emailSender.SendVerificationCode(ctx, email, code, s.config.EmailCodeTTL); err != nil {
		_ = s.emailCodeStore.DeleteEmailCodeIfMatches(context.WithoutCancel(ctx), email, digest)
		releaseCooldown()
		return nil, fmt.Errorf("%w: %v", ErrEmailDeliveryFailed, err)
	}

	result := &domain.EmailCodeStartResult{
		Email:              email,
		ExpiresInSeconds:   durationSecondsCeil(s.config.EmailCodeTTL),
		ResendAfterSeconds: durationSecondsCeil(s.config.EmailResendCooldown),
	}
	if s.config.ExposeEmailDevCode {
		result.DevCode = code
	}
	return result, nil
}

// LoginWithEmailCode consumes a valid challenge and creates or resumes the
// account identified by its normalized email address.
func (s *AuthService) LoginWithEmailCode(ctx context.Context, input domain.EmailCodeVerifyInput, meta domain.SessionMetadata) (*domain.AuthSessionResult, error) {
	email, err := normalizeEmail(input.Email)
	if err != nil {
		return nil, ErrInvalidVerificationCode
	}
	code := strings.TrimSpace(input.Code)
	if len(code) != 6 || strings.IndexFunc(code, func(r rune) bool { return r < '0' || r > '9' }) >= 0 {
		return nil, ErrInvalidVerificationCode
	}
	if !s.EmailCodeLoginAvailable(ctx) {
		return nil, ErrEmailCodeLoginUnavailable
	}

	verified, err := s.emailCodeStore.VerifyEmailCode(ctx, email, s.emailCodeDigest(email, code), s.config.MaxEmailCodeFailures)
	if err != nil {
		return nil, fmt.Errorf("verify email code: %w", err)
	}
	if !verified {
		return nil, ErrInvalidVerificationCode
	}

	user, err := s.userRepo.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	now := s.now()
	if user == nil {
		user = &domain.User{
			ID:                   generateID("usr"),
			Email:                email,
			DisplayName:          sanitizeDisplayName("", email),
			PreferredQuoteSource: s.config.DefaultQuoteSource,
			Provider:             domain.AuthProviderEmailCode,
			EmailVerified:        true,
			LastLoginAt:          &now,
			CreatedAt:            now,
			UpdatedAt:            now,
		}
	} else {
		user.EmailVerified = true
		user.LastLoginAt = &now
		user.UpdatedAt = now
	}
	if err := s.userRepo.SaveUser(ctx, user); err != nil {
		return nil, err
	}
	return s.createSession(ctx, user, meta)
}

func (s *AuthService) emailCodeDigest(email, code string) string {
	mac := hmac.New(sha256.New, []byte(s.config.EmailCodeSecret))
	_, _ = mac.Write([]byte(email))
	_, _ = mac.Write([]byte{'\n'})
	_, _ = mac.Write([]byte(code))
	return hex.EncodeToString(mac.Sum(nil))
}

func randomNumericCode(length int) (string, error) {
	if length <= 0 {
		return "", fmt.Errorf("verification code length must be positive")
	}
	code := make([]byte, length)
	for i := range code {
		value, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", err
		}
		code[i] = byte('0' + value.Int64())
	}
	return string(code), nil
}

func durationSecondsCeil(value time.Duration) int64 {
	if value <= 0 {
		return 0
	}
	return int64((value + time.Second - 1) / time.Second)
}
