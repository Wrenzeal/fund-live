package service

import (
	"context"
	"errors"
	"testing"

	"github.com/RomaticDOG/fund/internal/domain"
	"github.com/RomaticDOG/fund/internal/repository"
)

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
		Password:    "secret123",
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
	if result.User.PasswordHash == "secret123" || result.User.PasswordHash == "" {
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
		Password: "secret123",
	}, domain.SessionMetadata{})
	if err != nil {
		t.Fatalf("first registration error = %v", err)
	}

	_, err = service.RegisterWithPassword(context.Background(), domain.PasswordRegistrationInput{
		Email:    "BOSS@example.com",
		Password: "secret123",
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
		Password: "secret123",
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
		Password: "secret123",
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
