package domain

import (
	"context"
	"time"
)

// PasswordRegistrationInput represents the payload for a password registration flow.
type PasswordRegistrationInput struct {
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	Password    string `json:"password"`
}

// PasswordLoginInput represents the payload for a password login flow.
type PasswordLoginInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// GoogleLoginInput represents the payload for a Google sign-in flow.
type GoogleLoginInput struct {
	IDToken string `json:"id_token"`
}

// SessionMetadata describes request metadata captured at authentication time.
type SessionMetadata struct {
	UserAgent string `json:"user_agent"`
	IPAddress string `json:"ip_address"`
}

// AuthSessionResult is returned after a successful login or registration.
type AuthSessionResult struct {
	User         *User     `json:"user"`
	SessionToken string    `json:"-"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// AuthenticatedSession represents a validated session and its owner.
type AuthenticatedSession struct {
	User    *User        `json:"user"`
	Session *UserSession `json:"session"`
}

// AuthenticationService defines user authentication use cases.
type AuthenticationService interface {
	RegisterWithPassword(ctx context.Context, input PasswordRegistrationInput, meta SessionMetadata) (*AuthSessionResult, error)
	LoginWithPassword(ctx context.Context, input PasswordLoginInput, meta SessionMetadata) (*AuthSessionResult, error)
	LoginWithGoogle(ctx context.Context, input GoogleLoginInput, meta SessionMetadata) (*AuthSessionResult, error)
	AuthenticateSession(ctx context.Context, sessionToken string) (*AuthenticatedSession, error)
	LogoutByToken(ctx context.Context, sessionToken string) error
}
