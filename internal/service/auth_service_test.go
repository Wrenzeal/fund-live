package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/RomaticDOG/fund/internal/domain"
	"github.com/RomaticDOG/fund/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

const strongTestPassword = "Secret12345"

type mockGoogleVerifier struct {
	claims *GoogleIdentityClaims
	err    error
}

func (m mockGoogleVerifier) VerifyIDToken(ctx context.Context, idToken string) (*GoogleIdentityClaims, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.claims, nil
}

func TestAuthServiceRegisterAndAuthenticate(t *testing.T) {
	repo := repository.NewMemoryUserRepository()
	service := NewAuthService(repo, repo, DefaultAuthConfig())

	result, err := service.RegisterWithPassword(context.Background(), domain.PasswordRegistrationInput{
		Email:       "Boss@example.com",
		DisplayName: "Boss",
		Password:    strongTestPassword,
	}, domain.SessionMetadata{
		UserAgent: "test-agent",
		IPAddress: "127.0.0.1",
	})
	if err != nil {
		t.Fatalf("RegisterWithPassword() error = %v", err)
	}

	if result.User.Email != "boss@example.com" {
		t.Fatalf("normalized email = %q, want %q", result.User.Email, "boss@example.com")
	}
	if result.User.PasswordHash == strongTestPassword || result.User.PasswordHash == "" {
		t.Fatalf("password hash not stored correctly")
	}
	if result.SessionToken == "" {
		t.Fatalf("expected session token")
	}

	authenticated, err := service.AuthenticateSession(context.Background(), result.SessionToken)
	if err != nil {
		t.Fatalf("AuthenticateSession() error = %v", err)
	}
	if authenticated.User.ID != result.User.ID {
		t.Fatalf("authenticated user id = %q, want %q", authenticated.User.ID, result.User.ID)
	}
}

func TestAuthServiceRejectsDuplicateEmail(t *testing.T) {
	repo := repository.NewMemoryUserRepository()
	service := NewAuthService(repo, repo, DefaultAuthConfig())

	_, err := service.RegisterWithPassword(context.Background(), domain.PasswordRegistrationInput{
		Email:    "boss@example.com",
		Password: strongTestPassword,
	}, domain.SessionMetadata{})
	if err != nil {
		t.Fatalf("first registration error = %v", err)
	}

	_, err = service.RegisterWithPassword(context.Background(), domain.PasswordRegistrationInput{
		Email:    "BOSS@example.com",
		Password: strongTestPassword,
	}, domain.SessionMetadata{})
	if !errors.Is(err, ErrEmailAlreadyRegistered) {
		t.Fatalf("duplicate registration error = %v, want %v", err, ErrEmailAlreadyRegistered)
	}
}

func TestAuthServiceRejectsInvalidPassword(t *testing.T) {
	repo := repository.NewMemoryUserRepository()
	service := NewAuthService(repo, repo, DefaultAuthConfig())

	_, err := service.RegisterWithPassword(context.Background(), domain.PasswordRegistrationInput{
		Email:    "boss@example.com",
		Password: strongTestPassword,
	}, domain.SessionMetadata{})
	if err != nil {
		t.Fatalf("registration error = %v", err)
	}

	_, err = service.LoginWithPassword(context.Background(), domain.PasswordLoginInput{
		Email:    "boss@example.com",
		Password: "wrong-pass",
	}, domain.SessionMetadata{})
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("login error = %v, want %v", err, ErrInvalidCredentials)
	}
}

func TestAuthServiceGoogleAutoRegistersUser(t *testing.T) {
	repo := repository.NewMemoryUserRepository()
	service := NewAuthService(repo, repo, DefaultAuthConfig())
	service.googleVerifier = mockGoogleVerifier{
		claims: &GoogleIdentityClaims{
			Subject:       "google-sub-1",
			Email:         "boss.google@example.com",
			EmailVerified: true,
			Name:          "Boss Google",
			Picture:       "https://example.com/avatar.png",
		},
	}

	result, err := service.LoginWithGoogle(context.Background(), domain.GoogleLoginInput{
		IDToken: "fake-token",
	}, domain.SessionMetadata{})
	if err != nil {
		t.Fatalf("LoginWithGoogle() error = %v", err)
	}
	if result.User.Provider != domain.AuthProviderGoogle {
		t.Fatalf("provider = %q, want %q", result.User.Provider, domain.AuthProviderGoogle)
	}
	if result.User.GoogleSub != "google-sub-1" {
		t.Fatalf("google sub = %q", result.User.GoogleSub)
	}
}

func TestAuthServiceGoogleBindsExistingPasswordUser(t *testing.T) {
	repo := repository.NewMemoryUserRepository()
	service := NewAuthService(repo, repo, DefaultAuthConfig())

	registerResult, err := service.RegisterWithPassword(context.Background(), domain.PasswordRegistrationInput{
		Email:    "boss.bind@example.com",
		Password: strongTestPassword,
	}, domain.SessionMetadata{})
	if err != nil {
		t.Fatalf("RegisterWithPassword() error = %v", err)
	}

	service.googleVerifier = mockGoogleVerifier{
		claims: &GoogleIdentityClaims{
			Subject:       "google-sub-2",
			Email:         "boss.bind@example.com",
			EmailVerified: true,
			Name:          "Boss Bound",
		},
	}

	result, err := service.LoginWithGoogle(context.Background(), domain.GoogleLoginInput{
		IDToken: "fake-token",
	}, domain.SessionMetadata{})
	if err != nil {
		t.Fatalf("LoginWithGoogle() error = %v", err)
	}
	if result.User.ID != registerResult.User.ID {
		t.Fatalf("bound user id = %q, want %q", result.User.ID, registerResult.User.ID)
	}
	if result.User.Provider != domain.AuthProviderHybrid {
		t.Fatalf("provider = %q, want %q", result.User.Provider, domain.AuthProviderHybrid)
	}
}

func TestAuthServiceGoogleRejectsInvalidEmailClaim(t *testing.T) {
	repo := repository.NewMemoryUserRepository()
	service := NewAuthService(repo, repo, testAuthConfig())
	service.googleVerifier = mockGoogleVerifier{
		claims: &GoogleIdentityClaims{
			Subject:       "google-sub-invalid-email",
			Email:         "not-an-email",
			EmailVerified: true,
			Name:          "Invalid Email",
		},
	}

	_, err := service.LoginWithGoogle(context.Background(), domain.GoogleLoginInput{
		IDToken: "fake-token",
	}, domain.SessionMetadata{})
	if !errors.Is(err, ErrInvalidGoogleToken) {
		t.Fatalf("LoginWithGoogle() error = %v, want %v", err, ErrInvalidGoogleToken)
	}
}

func TestAuthServiceRejectsWeakPasswordPolicy(t *testing.T) {
	repo := repository.NewMemoryUserRepository()
	service := NewAuthService(repo, repo, testAuthConfig())

	tests := []string{
		"password123",
		"longpassword",
		"1234567890",
		"short1",
		"Secret 12345",
	}
	for _, password := range tests {
		t.Run(password, func(t *testing.T) {
			_, err := service.RegisterWithPassword(context.Background(), domain.PasswordRegistrationInput{
				Email:    "boss-weak@example.com",
				Password: password,
			}, domain.SessionMetadata{})
			if !errors.Is(err, ErrWeakPassword) {
				t.Fatalf("registration error = %v, want %v", err, ErrWeakPassword)
			}
		})
	}
}

func TestAuthServiceRateLimitsPasswordFailures(t *testing.T) {
	repo := repository.NewMemoryUserRepository()
	config := testAuthConfig()
	config.AuthAttemptWindow = time.Hour
	config.MaxPasswordFailures = 2
	service := NewAuthService(repo, repo, config)
	meta := domain.SessionMetadata{IPAddress: "127.0.0.1"}

	_, err := service.RegisterWithPassword(context.Background(), domain.PasswordRegistrationInput{
		Email:    "rate-limit@example.com",
		Password: strongTestPassword,
	}, meta)
	if err != nil {
		t.Fatalf("registration error = %v", err)
	}

	for i := 0; i < config.MaxPasswordFailures; i++ {
		_, err = service.LoginWithPassword(context.Background(), domain.PasswordLoginInput{
			Email:    "rate-limit@example.com",
			Password: "wrong-pass",
		}, meta)
		if !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("login %d error = %v, want %v", i+1, err, ErrInvalidCredentials)
		}
	}

	_, err = service.LoginWithPassword(context.Background(), domain.PasswordLoginInput{
		Email:    "rate-limit@example.com",
		Password: "wrong-pass",
	}, meta)
	if !errors.Is(err, ErrAuthRateLimited) {
		t.Fatalf("rate limited login error = %v, want %v", err, ErrAuthRateLimited)
	}
}

func TestAuthServiceClearsPasswordFailuresAfterSuccessfulLogin(t *testing.T) {
	repo := repository.NewMemoryUserRepository()
	config := testAuthConfig()
	config.AuthAttemptWindow = time.Hour
	config.MaxPasswordFailures = 2
	service := NewAuthService(repo, repo, config)
	meta := domain.SessionMetadata{IPAddress: "127.0.0.1"}

	_, err := service.RegisterWithPassword(context.Background(), domain.PasswordRegistrationInput{
		Email:    "rate-clear@example.com",
		Password: strongTestPassword,
	}, meta)
	if err != nil {
		t.Fatalf("registration error = %v", err)
	}

	_, err = service.LoginWithPassword(context.Background(), domain.PasswordLoginInput{
		Email:    "rate-clear@example.com",
		Password: "wrong-pass",
	}, meta)
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("failed login error = %v, want %v", err, ErrInvalidCredentials)
	}

	_, err = service.LoginWithPassword(context.Background(), domain.PasswordLoginInput{
		Email:    "rate-clear@example.com",
		Password: strongTestPassword,
	}, meta)
	if err != nil {
		t.Fatalf("successful login error = %v", err)
	}

	_, err = service.LoginWithPassword(context.Background(), domain.PasswordLoginInput{
		Email:    "rate-clear@example.com",
		Password: "wrong-pass",
	}, meta)
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("post-clear failed login error = %v, want %v", err, ErrInvalidCredentials)
	}
}

func testAuthConfig() AuthConfig {
	config := DefaultAuthConfig()
	config.BcryptCost = bcrypt.MinCost
	return config
}
