package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"github.com/RomaticDOG/fund/internal/domain"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidEmail           = errors.New("invalid email")
	ErrWeakPassword           = errors.New("password must be at least 8 characters")
	ErrEmailAlreadyRegistered = errors.New("email already registered")
	ErrInvalidCredentials     = errors.New("invalid credentials")
	ErrInvalidSession         = errors.New("invalid session")
	ErrSessionExpired         = errors.New("session expired")
	ErrGoogleLoginDisabled    = errors.New("google login is not configured")
	ErrInvalidGoogleToken     = errors.New("invalid google id token")
	ErrGoogleEmailNotVerified = errors.New("google account email is not verified")
)

// AuthConfig controls session and password authentication behavior.
type AuthConfig struct {
	CookieName           string
	CookieSecure         bool
	SessionTTL           time.Duration
	SessionTouchInterval time.Duration
	BcryptCost           int
	GoogleClientID       string
	DefaultQuoteSource   domain.QuoteSource
}

// DefaultAuthConfig returns the default authentication configuration.
func DefaultAuthConfig() AuthConfig {
	return AuthConfig{
		CookieName:           "fundlive_session",
		CookieSecure:         false,
		SessionTTL:           30 * 24 * time.Hour,
		SessionTouchInterval: 5 * time.Minute,
		BcryptCost:           bcrypt.DefaultCost,
		DefaultQuoteSource:   domain.QuoteSourceSina,
	}
}

// AuthService implements password-based authentication and server-side sessions.
type AuthService struct {
	userRepo       domain.UserRepository
	sessionRepo    domain.UserSessionRepository
	googleVerifier GoogleIDTokenVerifier
	config         AuthConfig
	now            func() time.Time
}

// NewAuthService creates a new AuthService.
func NewAuthService(
	userRepo domain.UserRepository,
	sessionRepo domain.UserSessionRepository,
	config AuthConfig,
) *AuthService {
	if config.CookieName == "" {
		config.CookieName = DefaultAuthConfig().CookieName
	}
	if config.SessionTTL <= 0 {
		config.SessionTTL = DefaultAuthConfig().SessionTTL
	}
	if config.SessionTouchInterval <= 0 {
		config.SessionTouchInterval = DefaultAuthConfig().SessionTouchInterval
	}
	if config.BcryptCost == 0 {
		config.BcryptCost = DefaultAuthConfig().BcryptCost
	}
	config.DefaultQuoteSource = domain.ResolveQuoteSource(config.DefaultQuoteSource, DefaultAuthConfig().DefaultQuoteSource)

	return &AuthService{
		userRepo:       userRepo,
		sessionRepo:    sessionRepo,
		googleVerifier: newGoogleIDTokenVerifier(config.GoogleClientID),
		config:         config,
		now:            time.Now,
	}
}

// RegisterWithPassword creates a new user and immediately creates a session.
func (s *AuthService) RegisterWithPassword(ctx context.Context, input domain.PasswordRegistrationInput, meta domain.SessionMetadata) (*domain.AuthSessionResult, error) {
	email, err := normalizeEmail(input.Email)
	if err != nil {
		return nil, err
	}

	password := strings.TrimSpace(input.Password)
	if len(password) < 8 {
		return nil, ErrWeakPassword
	}

	existing, err := s.userRepo.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrEmailAlreadyRegistered
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), s.config.BcryptCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	now := s.now()
	user := &domain.User{
		ID:                   generateID("usr"),
		Email:                email,
		DisplayName:          sanitizeDisplayName(input.DisplayName, email),
		PreferredQuoteSource: s.config.DefaultQuoteSource,
		PasswordHash:         string(passwordHash),
		Provider:             domain.AuthProviderPassword,
		EmailVerified:        false,
		LastLoginAt:          &now,
		CreatedAt:            now,
		UpdatedAt:            now,
	}

	if err := s.userRepo.SaveUser(ctx, user); err != nil {
		return nil, err
	}

	return s.createSession(ctx, user, meta)
}

// LoginWithPassword validates credentials and creates a new session.
func (s *AuthService) LoginWithPassword(ctx context.Context, input domain.PasswordLoginInput, meta domain.SessionMetadata) (*domain.AuthSessionResult, error) {
	email, err := normalizeEmail(input.Email)
	if err != nil {
		return nil, err
	}

	user, err := s.userRepo.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if user == nil || user.PasswordHash == "" {
		return nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	now := s.now()
	user.LastLoginAt = &now
	user.UpdatedAt = now
	if err := s.userRepo.SaveUser(ctx, user); err != nil {
		return nil, err
	}

	return s.createSession(ctx, user, meta)
}

// LoginWithGoogle verifies the Google ID token and signs the user in, creating an account on first login.
func (s *AuthService) LoginWithGoogle(ctx context.Context, input domain.GoogleLoginInput, meta domain.SessionMetadata) (*domain.AuthSessionResult, error) {
	if s.googleVerifier == nil {
		return nil, ErrGoogleLoginDisabled
	}

	claims, err := s.googleVerifier.VerifyIDToken(ctx, input.IDToken)
	if err != nil {
		return nil, err
	}
	if !claims.EmailVerified {
		return nil, ErrGoogleEmailNotVerified
	}
	if claims.Email == "" {
		return nil, ErrInvalidGoogleToken
	}

	user, err := s.userRepo.GetUserByGoogleSub(ctx, claims.Subject)
	if err != nil {
		return nil, err
	}

	if user == nil {
		user, err = s.userRepo.GetUserByEmail(ctx, claims.Email)
		if err != nil {
			return nil, err
		}
	}

	now := s.now()
	if user == nil {
		user = &domain.User{
			ID:                   generateID("usr"),
			Email:                claims.Email,
			DisplayName:          sanitizeDisplayName(claims.Name, claims.Email),
			AvatarURL:            strings.TrimSpace(claims.Picture),
			PreferredQuoteSource: s.config.DefaultQuoteSource,
			GoogleSub:            claims.Subject,
			Provider:             domain.AuthProviderGoogle,
			EmailVerified:        true,
			LastLoginAt:          &now,
			CreatedAt:            now,
			UpdatedAt:            now,
		}
	} else {
		user.Email = claims.Email
		user.DisplayName = sanitizeDisplayName(firstNonEmpty(claims.Name, user.DisplayName), claims.Email)
		if claims.Picture != "" {
			user.AvatarURL = claims.Picture
		}
		user.GoogleSub = claims.Subject
		user.EmailVerified = true
		user.LastLoginAt = &now
		user.UpdatedAt = now

		switch {
		case user.PasswordHash != "" && user.Provider != domain.AuthProviderHybrid:
			user.Provider = domain.AuthProviderHybrid
		case user.PasswordHash == "":
			user.Provider = domain.AuthProviderGoogle
		}
	}

	if err := s.userRepo.SaveUser(ctx, user); err != nil {
		return nil, err
	}

	return s.createSession(ctx, user, meta)
}

// AuthenticateSession validates a session token and returns the associated user.
func (s *AuthService) AuthenticateSession(ctx context.Context, sessionToken string) (*domain.AuthenticatedSession, error) {
	sessionToken = strings.TrimSpace(sessionToken)
	if sessionToken == "" {
		return nil, ErrInvalidSession
	}

	tokenHash := hashToken(sessionToken)
	session, err := s.sessionRepo.GetSessionByTokenHash(ctx, tokenHash)
	if err != nil {
		return nil, err
	}
	if session == nil {
		return nil, ErrInvalidSession
	}

	now := s.now()
	if now.After(session.ExpiresAt) {
		_ = s.sessionRepo.DeleteSessionByTokenHash(ctx, tokenHash)
		return nil, ErrSessionExpired
	}

	user, err := s.userRepo.GetUserByID(ctx, session.UserID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		_ = s.sessionRepo.DeleteSessionByTokenHash(ctx, tokenHash)
		return nil, ErrInvalidSession
	}

	if now.Sub(session.LastSeenAt) >= s.config.SessionTouchInterval {
		if err := s.sessionRepo.UpdateSessionLastSeen(ctx, session.ID, now); err == nil {
			session.LastSeenAt = now
		}
	}

	return &domain.AuthenticatedSession{
		User:    user,
		Session: session,
	}, nil
}

// LogoutByToken revokes a session token.
func (s *AuthService) LogoutByToken(ctx context.Context, sessionToken string) error {
	sessionToken = strings.TrimSpace(sessionToken)
	if sessionToken == "" {
		return nil
	}
	return s.sessionRepo.DeleteSessionByTokenHash(ctx, hashToken(sessionToken))
}

func (s *AuthService) createSession(ctx context.Context, user *domain.User, meta domain.SessionMetadata) (*domain.AuthSessionResult, error) {
	now := s.now()
	sessionToken, err := generateToken(32)
	if err != nil {
		return nil, fmt.Errorf("generate session token: %w", err)
	}

	session := &domain.UserSession{
		ID:         generateID("ses"),
		UserID:     user.ID,
		TokenHash:  hashToken(sessionToken),
		UserAgent:  strings.TrimSpace(meta.UserAgent),
		IPAddress:  strings.TrimSpace(meta.IPAddress),
		ExpiresAt:  now.Add(s.config.SessionTTL),
		CreatedAt:  now,
		LastSeenAt: now,
	}

	if err := s.sessionRepo.SaveSession(ctx, session); err != nil {
		return nil, err
	}

	return &domain.AuthSessionResult{
		User:         user,
		SessionToken: sessionToken,
		ExpiresAt:    session.ExpiresAt,
	}, nil
}

func normalizeEmail(raw string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(raw))
	if normalized == "" {
		return "", ErrInvalidEmail
	}

	parsed, err := mail.ParseAddress(normalized)
	if err != nil || parsed.Address != normalized {
		return "", ErrInvalidEmail
	}
	return normalized, nil
}

func sanitizeDisplayName(raw, email string) string {
	name := strings.TrimSpace(raw)
	if name != "" {
		return name
	}

	if idx := strings.Index(email, "@"); idx > 0 {
		return email[:idx]
	}
	return email
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func generateID(prefix string) string {
	token, err := generateToken(12)
	if err != nil {
		return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
	}
	return prefix + "_" + token
}

func generateToken(size int) (string, error) {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
