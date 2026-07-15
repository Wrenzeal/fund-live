package cache

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"
)

func TestDragonflyAuthCodeStorePrimitives(t *testing.T) {
	redisURL := os.Getenv("TEST_REDIS_URL")
	if redisURL == "" {
		t.Skip("TEST_REDIS_URL is not configured")
	}
	prefix := fmt.Sprintf("fundlive-test-%d", time.Now().UnixNano())
	store, err := NewDragonflyAuthCodeStore(redisURL, prefix)
	if err != nil {
		t.Fatalf("NewDragonflyAuthCodeStore() error = %v", err)
	}
	defer store.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := store.Ping(ctx); err != nil {
		t.Fatalf("Ping() error = %v", err)
	}

	if err := store.PutEmailCode(ctx, "user@example.com", "digest", time.Minute); err != nil {
		t.Fatalf("PutEmailCode() error = %v", err)
	}
	for i := 0; i < 4; i++ {
		verified, err := store.VerifyEmailCode(ctx, "user@example.com", "wrong", 5)
		if err != nil || verified {
			t.Fatalf("wrong verification %d = %v, %v", i+1, verified, err)
		}
	}
	verified, err := store.VerifyEmailCode(ctx, "user@example.com", "digest", 5)
	if err != nil || !verified {
		t.Fatalf("correct verification = %v, %v", verified, err)
	}
	verified, err = store.VerifyEmailCode(ctx, "user@example.com", "digest", 5)
	if err != nil || verified {
		t.Fatalf("reused verification = %v, %v", verified, err)
	}

	if err := store.PutEmailCode(ctx, "limited@example.com", "digest-2", time.Minute); err != nil {
		t.Fatalf("PutEmailCode(limited) error = %v", err)
	}
	for i := 0; i < 5; i++ {
		_, err := store.VerifyEmailCode(ctx, "limited@example.com", "wrong", 5)
		if err != nil {
			t.Fatalf("failure verification %d error = %v", i+1, err)
		}
	}
	verified, err = store.VerifyEmailCode(ctx, "limited@example.com", "digest-2", 5)
	if err != nil || verified {
		t.Fatalf("verification after failure cap = %v, %v", verified, err)
	}

	acquired, _, err := store.AcquireCooldown(ctx, "email:user@example.com", "token", time.Minute)
	if err != nil || !acquired {
		t.Fatalf("first cooldown = %v, %v", acquired, err)
	}
	acquired, retryAfter, err := store.AcquireCooldown(ctx, "email:user@example.com", "new-token", time.Minute)
	if err != nil || acquired || retryAfter <= 0 {
		t.Fatalf("second cooldown = %v, %v, %v", acquired, retryAfter, err)
	}
	if err := store.ReleaseCooldownIfMatches(ctx, "email:user@example.com", "token"); err != nil {
		t.Fatalf("ReleaseCooldownIfMatches() error = %v", err)
	}
	acquired, _, err = store.AcquireCooldown(ctx, "email:user@example.com", "third-token", time.Minute)
	if err != nil || !acquired {
		t.Fatalf("cooldown after release = %v, %v", acquired, err)
	}

	for i := 0; i < 3; i++ {
		allowed, retryAfter, err := store.AllowFixedWindow(ctx, "ip:127.0.0.1", 2, time.Minute)
		if err != nil {
			t.Fatalf("AllowFixedWindow(%d) error = %v", i+1, err)
		}
		if allowed != (i < 2) || retryAfter <= 0 {
			t.Fatalf("AllowFixedWindow(%d) = %v, %v", i+1, allowed, retryAfter)
		}
	}
}
