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
	"sync"
	"time"
	"unicode"

	"github.com/RomaticDOG/fund/internal/domain"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidEmail           = errors.New("invalid email")
	ErrWeakPassword           = errors.New("password must be at least 10 characters, include letters and numbers, and contain no spaces")
	ErrAuthRateLimited        = errors.New("too many authentication attempts; please try again later")
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
	CookieName             string
	CookieSecure           bool
	SessionTTL             time.Duration
	SessionTouchInterval   time.Duration
	BcryptCost             int
	GoogleClientID         string
	DefaultQuoteSource     domain.QuoteSource
	AuthAttemptWindow      time.Duration
	MaxPasswordFailures    int
	MaxRegisterFailures    int
	MaxGoogleLoginFailures int
}

// DefaultAuthConfig returns the default authentication configuration.
func DefaultAuthConfig() AuthConfig {
	return AuthConfig{
		CookieName:             "fundlive_session",
		CookieSecure:           false,
		SessionTTL:             30 * 24 * time.Hour,
		SessionTouchInterval:   5 * time.Minute,
		BcryptCost:             bcrypt.DefaultCost,
		DefaultQuoteSource:     domain.QuoteSourceSina,
		AuthAttemptWindow:      15 * time.Minute,
		MaxPasswordFailures:    5,
		MaxRegisterFailures:    8,
		MaxGoogleLoginFailures: 10,
	}
}

// AuthService implements password-based authentication and server-side sessions.
type AuthService struct {
	userRepo       domain.UserRepository
	sessionRepo    domain.UserSessionRepository
	googleVerifier GoogleIDTokenVerifier
	config         AuthConfig
	now            func() time.Time
	rateLimiter    *authAttemptLimiter
}

// NewAuthService creates a new AuthService.
func NewAuthService(
	userRepo domain.UserRepository,
	sessionRepo domain.UserSessionRepository,
	config AuthConfig,
) *AuthService {
	defaults := DefaultAuthConfig()
	if config.CookieName == "" {
		config.CookieName = defaults.CookieName
	}
	if config.SessionTTL <= 0 {
		config.SessionTTL = defaults.SessionTTL
	}
	if config.SessionTouchInterval <= 0 {
		config.SessionTouchInterval = defaults.SessionTouchInterval
	}
	if config.BcryptCost == 0 {
		config.BcryptCost = defaults.BcryptCost
	}
	if config.AuthAttemptWindow <= 0 {
		config.AuthAttemptWindow = defaults.AuthAttemptWindow
	}
	if config.MaxPasswordFailures <= 0 {
		config.MaxPasswordFailures = defaults.MaxPasswordFailures
	}
	if config.MaxRegisterFailures <= 0 {
		config.MaxRegisterFailures = defaults.MaxRegisterFailures
	}
	if config.MaxGoogleLoginFailures <= 0 {
		config.MaxGoogleLoginFailures = defaults.MaxGoogleLoginFailures
	}
	config.DefaultQuoteSource = domain.ResolveQuoteSource(config.DefaultQuoteSource, defaults.DefaultQuoteSource)

	return &AuthService{
		userRepo:       userRepo,
		sessionRepo:    sessionRepo,
		googleVerifier: newGoogleIDTokenVerifier(config.GoogleClientID),
		config:         config,
		now:            time.Now,
		rateLimiter:    newAuthAttemptLimiter(config.AuthAttemptWindow),
	}
}

// RegisterWithPassword creates a new user and immediately creates a session.
func (s *AuthService) RegisterWithPassword(ctx context.Context, input domain.PasswordRegistrationInput, meta domain.SessionMetadata) (*domain.AuthSessionResult, error) {
	email, err := normalizeEmail(input.Email)
	if err != nil {
		if rateErr := s.checkRateLimit(authAttemptKindRegister, meta, ""); rateErr != nil {
			return nil, rateErr
		}
		s.recordAuthFailure(authAttemptKindRegister, meta, "")
		return nil, err
	}

	if err := s.checkRateLimit(authAttemptKindRegister, meta, email); err != nil {
		return nil, err
	}

	password := input.Password
	if err := validatePasswordStrength(password); err != nil {
		s.recordAuthFailure(authAttemptKindRegister, meta, email)
		return nil, err
	}

	existing, err := s.userRepo.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		s.recordAuthFailure(authAttemptKindRegister, meta, email)
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

	s.clearAuthFailures(authAttemptKindRegister, meta, email)
	return s.createSession(ctx, user, meta)
}

// LoginWithPassword validates credentials and creates a new session.
func (s *AuthService) LoginWithPassword(ctx context.Context, input domain.PasswordLoginInput, meta domain.SessionMetadata) (*domain.AuthSessionResult, error) {
	email, err := normalizeEmail(input.Email)
	if err != nil {
		if rateErr := s.checkRateLimit(authAttemptKindPassword, meta, ""); rateErr != nil {
			return nil, rateErr
		}
		s.recordAuthFailure(authAttemptKindPassword, meta, "")
		return nil, ErrInvalidCredentials
	}
	if err := s.checkRateLimit(authAttemptKindPassword, meta, email); err != nil {
		return nil, err
	}

	user, err := s.userRepo.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if user == nil || user.PasswordHash == "" {
		s.recordAuthFailure(authAttemptKindPassword, meta, email)
		return nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)); err != nil {
		s.recordAuthFailure(authAttemptKindPassword, meta, email)
		return nil, ErrInvalidCredentials
	}

	now := s.now()
	user.LastLoginAt = &now
	user.UpdatedAt = now
	if err := s.userRepo.SaveUser(ctx, user); err != nil {
		return nil, err
	}

	s.clearAuthFailures(authAttemptKindPassword, meta, email)
	return s.createSession(ctx, user, meta)
}

// LoginWithGoogle verifies the Google ID token and signs the user in, creating an account on first login.
func (s *AuthService) LoginWithGoogle(ctx context.Context, input domain.GoogleLoginInput, meta domain.SessionMetadata) (*domain.AuthSessionResult, error) {
	if s.googleVerifier == nil {
		return nil, ErrGoogleLoginDisabled
	}
	if err := s.checkRateLimit(authAttemptKindGoogle, meta, ""); err != nil {
		return nil, err
	}

	claims, err := s.googleVerifier.VerifyIDToken(ctx, input.IDToken)
	if err != nil {
		s.recordAuthFailure(authAttemptKindGoogle, meta, "")
		return nil, err
	}
	if !claims.EmailVerified {
		s.recordAuthFailure(authAttemptKindGoogle, meta, "")
		return nil, ErrGoogleEmailNotVerified
	}
	email, err := normalizeEmail(claims.Email)
	if err != nil {
		s.recordAuthFailure(authAttemptKindGoogle, meta, "")
		return nil, ErrInvalidGoogleToken
	}

	user, err := s.userRepo.GetUserByGoogleSub(ctx, claims.Subject)
	if err != nil {
		return nil, err
	}

	if user == nil {
		user, err = s.userRepo.GetUserByEmail(ctx, email)
		if err != nil {
			return nil, err
		}
	}

	now := s.now()
	if user == nil {
		user = &domain.User{
			ID:                   generateID("usr"),
			Email:                email,
			DisplayName:          sanitizeDisplayName(claims.Name, email),
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
		user.Email = email
		user.DisplayName = sanitizeDisplayName(firstNonEmpty(claims.Name, user.DisplayName), email)
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

	s.clearAuthFailures(authAttemptKindGoogle, meta, "")
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

func (s *AuthService) checkRateLimit(kind authAttemptKind, meta domain.SessionMetadata, email string) error {
	if s.rateLimiter == nil {
		return nil
	}
	if s.rateLimiter.isLimited(s.authAttemptKey(kind, meta, email), s.maxAttemptsFor(kind), s.now()) {
		return ErrAuthRateLimited
	}
	return nil
}

func (s *AuthService) recordAuthFailure(kind authAttemptKind, meta domain.SessionMetadata, email string) {
	if s.rateLimiter == nil {
		return
	}
	s.rateLimiter.recordFailure(s.authAttemptKey(kind, meta, email), s.now())
}

func (s *AuthService) clearAuthFailures(kind authAttemptKind, meta domain.SessionMetadata, email string) {
	if s.rateLimiter == nil {
		return
	}
	s.rateLimiter.clear(s.authAttemptKey(kind, meta, email))
}

func (s *AuthService) authAttemptKey(kind authAttemptKind, meta domain.SessionMetadata, email string) string {
	return string(kind) + "|" + normalizeRateLimitPart(meta.IPAddress) + "|" + normalizeRateLimitPart(email)
}

func (s *AuthService) maxAttemptsFor(kind authAttemptKind) int {
	switch kind {
	case authAttemptKindRegister:
		return s.config.MaxRegisterFailures
	case authAttemptKindGoogle:
		return s.config.MaxGoogleLoginFailures
	default:
		return s.config.MaxPasswordFailures
	}
}

func normalizeRateLimitPart(raw string) string {
	part := strings.ToLower(strings.TrimSpace(raw))
	if part == "" {
		return "unknown"
	}
	return part
}

func validatePasswordStrength(password string) error {
	if len([]rune(password)) < 10 {
		return ErrWeakPassword
	}
	lowered := strings.ToLower(password)
	weakPasswords := map[string]struct{}{
		"password":    {},
		"password123": {},
		"qwerty123":   {},
		"1234567890":  {},
		"admin123456": {},
		"fundlive123": {},
		"zhang123456": {},
	}
	if _, ok := weakPasswords[lowered]; ok {
		return ErrWeakPassword
	}

	hasLetter := false
	hasNumber := false
	for _, ch := range password {
		switch {
		case unicode.IsSpace(ch):
			return ErrWeakPassword
		case unicode.IsLetter(ch):
			hasLetter = true
		case unicode.IsNumber(ch):
			hasNumber = true
		}
	}
	if !hasLetter || !hasNumber {
		return ErrWeakPassword
	}
	return nil
}

type authAttemptKind string

const (
	authAttemptKindPassword authAttemptKind = "password"
	authAttemptKindRegister authAttemptKind = "register"
	authAttemptKindGoogle   authAttemptKind = "google"
)

type authAttemptLimiter struct {
	window       time.Duration
	mu           sync.Mutex
	attempts     map[string]authAttemptRecord
	lastPrunedAt time.Time
}

type authAttemptRecord struct {
	count       int
	firstSeenAt time.Time
}

func newAuthAttemptLimiter(window time.Duration) *authAttemptLimiter {
	return &authAttemptLimiter{
		window:   window,
		attempts: make(map[string]authAttemptRecord),
	}
}

func (l *authAttemptLimiter) isLimited(key string, maxAttempts int, now time.Time) bool {
	if l == nil || maxAttempts <= 0 {
		return false
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.pruneExpiredLocked(now)

	record, ok := l.attempts[key]
	if !ok {
		return false
	}
	if now.Sub(record.firstSeenAt) >= l.window {
		delete(l.attempts, key)
		return false
	}
	return record.count >= maxAttempts
}

func (l *authAttemptLimiter) recordFailure(key string, now time.Time) {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.pruneExpiredLocked(now)

	record, ok := l.attempts[key]
	if !ok || now.Sub(record.firstSeenAt) >= l.window {
		l.attempts[key] = authAttemptRecord{count: 1, firstSeenAt: now}
		return
	}
	record.count++
	l.attempts[key] = record
}

func (l *authAttemptLimiter) clear(key string) {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.attempts, key)
}

func (l *authAttemptLimiter) pruneExpiredLocked(now time.Time) {
	if l.window <= 0 {
		return
	}
	pruneInterval := l.window / 2
	if pruneInterval < time.Minute {
		pruneInterval = time.Minute
	}
	if !l.lastPrunedAt.IsZero() && now.Sub(l.lastPrunedAt) < pruneInterval {
		return
	}
	for key, record := range l.attempts {
		if now.Sub(record.firstSeenAt) >= l.window {
			delete(l.attempts, key)
		}
	}
	l.lastPrunedAt = now
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
