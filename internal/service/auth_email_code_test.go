package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"image/png"
	"io"
	"mime"
	"mime/multipart"
	"net/mail"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/RomaticDOG/fund/internal/domain"
	"github.com/RomaticDOG/fund/internal/repository"
)

type fakeAuthCodeStore struct {
	mu        sync.Mutex
	pingErr   error
	codes     map[string]string
	failures  map[string]int
	cooldowns map[string]string
	rates     map[string]int
}

func newFakeAuthCodeStore() *fakeAuthCodeStore {
	return &fakeAuthCodeStore{
		codes:     make(map[string]string),
		failures:  make(map[string]int),
		cooldowns: make(map[string]string),
		rates:     make(map[string]int),
	}
}

func (s *fakeAuthCodeStore) Ping(context.Context) error { return s.pingErr }

func (s *fakeAuthCodeStore) PutEmailCode(_ context.Context, email, digest string, _ time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.codes[email] = digest
	delete(s.failures, email)
	return nil
}

func (s *fakeAuthCodeStore) VerifyEmailCode(_ context.Context, email, digest string, maxFailures int) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	current, ok := s.codes[email]
	if !ok {
		return false, nil
	}
	if current == digest {
		delete(s.codes, email)
		delete(s.failures, email)
		return true, nil
	}
	s.failures[email]++
	if s.failures[email] >= maxFailures {
		delete(s.codes, email)
		delete(s.failures, email)
	}
	return false, nil
}

func (s *fakeAuthCodeStore) DeleteEmailCodeIfMatches(_ context.Context, email, digest string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.codes[email] == digest {
		delete(s.codes, email)
		delete(s.failures, email)
	}
	return nil
}

func (s *fakeAuthCodeStore) AcquireCooldown(_ context.Context, scope, token string, ttl time.Duration) (bool, time.Duration, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.cooldowns[scope]; exists {
		return false, ttl, nil
	}
	s.cooldowns[scope] = token
	return true, ttl, nil
}

func (s *fakeAuthCodeStore) ReleaseCooldownIfMatches(_ context.Context, scope, token string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cooldowns[scope] == token {
		delete(s.cooldowns, scope)
	}
	return nil
}

func (s *fakeAuthCodeStore) AllowFixedWindow(_ context.Context, scope string, limit int, window time.Duration) (bool, time.Duration, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rates[scope]++
	return s.rates[scope] <= limit, window, nil
}

type fakeEmailSender struct {
	mu    sync.Mutex
	email string
	code  string
	err   error
}

func (s *fakeEmailSender) SendVerificationCode(_ context.Context, email, code string, _ time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.email = email
	s.code = code
	return s.err
}

func emailCodeTestConfig() AuthConfig {
	config := DefaultAuthConfig()
	config.EmailCodeEnabled = true
	config.EmailCodeSecret = "test-email-code-secret-32-bytes!"
	config.ExposeEmailDevCode = true
	config.EmailResendCooldown = time.Minute
	config.MaxEmailSendsPerHour = 5
	config.MaxIPEmailSendsPerHour = 20
	config.MaxEmailCodeFailures = 5
	return config
}

func TestEmailCodeLoginAutoCreatesAndConsumesChallenge(t *testing.T) {
	repo := repository.NewMemoryUserRepository()
	store := newFakeAuthCodeStore()
	sender := &fakeEmailSender{}
	auth := NewAuthService(repo, repo, emailCodeTestConfig())
	auth.SetEmailCodeDependencies(store, sender)

	start, err := auth.StartEmailCodeLogin(context.Background(), domain.EmailCodeStartInput{Email: " New.User@Example.com "}, domain.SessionMetadata{IPAddress: "127.0.0.1"})
	if err != nil {
		t.Fatalf("StartEmailCodeLogin() error = %v", err)
	}
	if start.Email != "new.user@example.com" || start.DevCode == "" {
		t.Fatalf("start result = %#v", start)
	}
	if sender.email != start.Email || sender.code != start.DevCode {
		t.Fatalf("sender captured email=%q code=%q", sender.email, sender.code)
	}

	result, err := auth.LoginWithEmailCode(context.Background(), domain.EmailCodeVerifyInput{Email: start.Email, Code: start.DevCode}, domain.SessionMetadata{})
	if err != nil {
		t.Fatalf("LoginWithEmailCode() error = %v", err)
	}
	if result.SessionToken == "" || result.User.Provider != domain.AuthProviderEmailCode || !result.User.EmailVerified {
		t.Fatalf("login result = %#v", result)
	}
	if result.User.DisplayName != "New.User" && result.User.DisplayName != "new.user" {
		t.Fatalf("display name = %q", result.User.DisplayName)
	}

	_, err = auth.LoginWithEmailCode(context.Background(), domain.EmailCodeVerifyInput{Email: start.Email, Code: start.DevCode}, domain.SessionMetadata{})
	if !errors.Is(err, ErrInvalidVerificationCode) {
		t.Fatalf("reused code error = %v, want %v", err, ErrInvalidVerificationCode)
	}
}

func TestEmailCodeStartDoesNotExposeProductionCode(t *testing.T) {
	repo := repository.NewMemoryUserRepository()
	store := newFakeAuthCodeStore()
	sender := &fakeEmailSender{}
	config := emailCodeTestConfig()
	config.ExposeEmailDevCode = false
	auth := NewAuthService(repo, repo, config)
	auth.SetEmailCodeDependencies(store, sender)

	start, err := auth.StartEmailCodeLogin(context.Background(), domain.EmailCodeStartInput{Email: "production@example.com"}, domain.SessionMetadata{IPAddress: "127.0.0.5"})
	if err != nil {
		t.Fatalf("StartEmailCodeLogin() error = %v", err)
	}
	if start.DevCode != "" || sender.code == "" {
		t.Fatalf("dev code = %q, delivered code = %q", start.DevCode, sender.code)
	}
}

func TestEmailCodeLoginPreservesExistingPasswordAccount(t *testing.T) {
	repo := repository.NewMemoryUserRepository()
	store := newFakeAuthCodeStore()
	sender := &fakeEmailSender{}
	auth := NewAuthService(repo, repo, emailCodeTestConfig())
	auth.SetEmailCodeDependencies(store, sender)

	registered, err := auth.RegisterWithPassword(context.Background(), domain.PasswordRegistrationInput{
		Email: "holder@example.com", Password: "safe-password-2026", DisplayName: "持仓用户",
	}, domain.SessionMetadata{})
	if err != nil {
		t.Fatalf("RegisterWithPassword() error = %v", err)
	}
	start, err := auth.StartEmailCodeLogin(context.Background(), domain.EmailCodeStartInput{Email: "holder@example.com"}, domain.SessionMetadata{IPAddress: "127.0.0.2"})
	if err != nil {
		t.Fatalf("StartEmailCodeLogin() error = %v", err)
	}
	loggedIn, err := auth.LoginWithEmailCode(context.Background(), domain.EmailCodeVerifyInput{Email: start.Email, Code: start.DevCode}, domain.SessionMetadata{})
	if err != nil {
		t.Fatalf("LoginWithEmailCode() error = %v", err)
	}
	if loggedIn.User.ID != registered.User.ID || loggedIn.User.Provider != domain.AuthProviderPassword || !loggedIn.User.EmailVerified {
		t.Fatalf("existing user changed unexpectedly: %#v", loggedIn.User)
	}
	if _, err := auth.LoginWithPassword(context.Background(), domain.PasswordLoginInput{Email: start.Email, Password: "safe-password-2026"}, domain.SessionMetadata{}); err != nil {
		t.Fatalf("password login after email verification error = %v", err)
	}
}

func TestEmailCodeLoginInvalidatesAfterFiveFailures(t *testing.T) {
	repo := repository.NewMemoryUserRepository()
	store := newFakeAuthCodeStore()
	auth := NewAuthService(repo, repo, emailCodeTestConfig())
	auth.SetEmailCodeDependencies(store, &fakeEmailSender{})

	start, err := auth.StartEmailCodeLogin(context.Background(), domain.EmailCodeStartInput{Email: "risk@example.com"}, domain.SessionMetadata{IPAddress: "127.0.0.3"})
	if err != nil {
		t.Fatalf("StartEmailCodeLogin() error = %v", err)
	}
	for i := 0; i < 5; i++ {
		if _, err := auth.LoginWithEmailCode(context.Background(), domain.EmailCodeVerifyInput{Email: start.Email, Code: "000000"}, domain.SessionMetadata{}); !errors.Is(err, ErrInvalidVerificationCode) {
			t.Fatalf("wrong attempt %d error = %v", i+1, err)
		}
	}
	if _, err := auth.LoginWithEmailCode(context.Background(), domain.EmailCodeVerifyInput{Email: start.Email, Code: start.DevCode}, domain.SessionMetadata{}); !errors.Is(err, ErrInvalidVerificationCode) {
		t.Fatalf("correct code after failure limit error = %v", err)
	}
}

func TestEmailDeliveryFailureCleansChallengeAndCooldown(t *testing.T) {
	repo := repository.NewMemoryUserRepository()
	store := newFakeAuthCodeStore()
	sender := &fakeEmailSender{err: errors.New("smtp unavailable")}
	auth := NewAuthService(repo, repo, emailCodeTestConfig())
	auth.SetEmailCodeDependencies(store, sender)
	input := domain.EmailCodeStartInput{Email: "delivery@example.com"}
	meta := domain.SessionMetadata{IPAddress: "127.0.0.4"}

	if _, err := auth.StartEmailCodeLogin(context.Background(), input, meta); !errors.Is(err, ErrEmailDeliveryFailed) {
		t.Fatalf("delivery error = %v, want %v", err, ErrEmailDeliveryFailed)
	}
	sender.err = nil
	if _, err := auth.StartEmailCodeLogin(context.Background(), input, meta); err != nil {
		t.Fatalf("retry after delivery failure error = %v", err)
	}
}

func TestEmailCodeLoginAvailabilityDegradesWithStoreFailure(t *testing.T) {
	repo := repository.NewMemoryUserRepository()
	store := newFakeAuthCodeStore()
	store.pingErr = errors.New("dragonfly unavailable")
	auth := NewAuthService(repo, repo, emailCodeTestConfig())
	auth.SetEmailCodeDependencies(store, &fakeEmailSender{})

	if auth.EmailCodeLoginAvailable(context.Background()) {
		t.Fatal("EmailCodeLoginAvailable() = true, want false")
	}
	_, err := auth.StartEmailCodeLogin(context.Background(), domain.EmailCodeStartInput{Email: "user@example.com"}, domain.SessionMetadata{})
	if !errors.Is(err, ErrEmailCodeLoginUnavailable) {
		t.Fatalf("StartEmailCodeLogin() error = %v, want unavailable", err)
	}
}

func TestVerificationEmailContainsFundLiveBrandAndCode(t *testing.T) {
	message, err := buildVerificationEmail("fundlive@mail.wrenzeal.top", "FundLive", "user@example.com", "123456", 10*time.Minute)
	if err != nil {
		t.Fatalf("buildVerificationEmail() error = %v", err)
	}

	parsed, err := mail.ReadMessage(bytes.NewReader(message))
	if err != nil {
		t.Fatalf("mail.ReadMessage() error = %v", err)
	}
	mediaType, relatedParams, err := mime.ParseMediaType(parsed.Header.Get("Content-Type"))
	if err != nil {
		t.Fatalf("parse related content type: %v", err)
	}
	if mediaType != "multipart/related" || relatedParams["type"] != "multipart/alternative" {
		t.Fatalf("top-level content type = %q params=%v", mediaType, relatedParams)
	}

	related := multipart.NewReader(parsed.Body, relatedParams["boundary"])
	alternativePart, err := related.NextPart()
	if err != nil {
		t.Fatalf("read alternative part: %v", err)
	}
	alternativeType, alternativeParams, err := mime.ParseMediaType(alternativePart.Header.Get("Content-Type"))
	if err != nil {
		t.Fatalf("parse alternative content type: %v", err)
	}
	if alternativeType != "multipart/alternative" {
		t.Fatalf("first related part content type = %q", alternativeType)
	}

	alternative := multipart.NewReader(alternativePart, alternativeParams["boundary"])
	bodies := make(map[string]string)
	for {
		part, err := alternative.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatalf("read alternative body: %v", err)
		}
		partType, _, err := mime.ParseMediaType(part.Header.Get("Content-Type"))
		if err != nil {
			t.Fatalf("parse alternative body content type: %v", err)
		}
		body, err := io.ReadAll(part)
		if err != nil {
			t.Fatalf("read %s body: %v", partType, err)
		}
		bodies[partType] = string(body)
	}
	for _, partType := range []string{"text/plain", "text/html"} {
		body := bodies[partType]
		for _, expected := range []string{"涨了多少", "FundLive", "123456", "10 分钟", "不会向你索要验证码", "fund.wrenzeal.top"} {
			if !strings.Contains(body, expected) {
				t.Fatalf("%s body missing %q", partType, expected)
			}
		}
	}
	if !strings.Contains(bodies["text/html"], `src="cid:`+fundLiveEmailMarkCID+`"`) {
		t.Fatalf("HTML body does not reference embedded mark CID")
	}

	imagePart, err := related.NextPart()
	if err != nil {
		t.Fatalf("read inline image part: %v", err)
	}
	imageType, _, err := mime.ParseMediaType(imagePart.Header.Get("Content-Type"))
	if err != nil {
		t.Fatalf("parse inline image content type: %v", err)
	}
	if imageType != "image/png" || imagePart.Header.Get("Content-ID") != "<"+fundLiveEmailMarkCID+">" {
		t.Fatalf("inline image headers: type=%q cid=%q", imageType, imagePart.Header.Get("Content-ID"))
	}
	decodedPNG, err := io.ReadAll(base64.NewDecoder(base64.StdEncoding, imagePart))
	if err != nil {
		t.Fatalf("decode inline image: %v", err)
	}
	if !bytes.Equal(decodedPNG, fundLiveEmailMarkPNG) || !bytes.HasPrefix(decodedPNG, []byte("\x89PNG\r\n\x1a\n")) {
		t.Fatal("inline image is not the embedded FundLive PNG")
	}
	imageConfig, err := png.DecodeConfig(bytes.NewReader(decodedPNG))
	if err != nil {
		t.Fatalf("decode inline PNG config: %v", err)
	}
	if imageConfig.Width != 96 || imageConfig.Height != 96 {
		t.Fatalf("inline PNG size = %dx%d, want 96x96", imageConfig.Width, imageConfig.Height)
	}
	if _, err := related.NextPart(); !errors.Is(err, io.EOF) {
		t.Fatalf("unexpected extra related part: %v", err)
	}
}

func TestProductionEmailSenderRejectsDevAndIncompleteSMTP(t *testing.T) {
	if _, err := NewEmailSender("dev", SMTPEmailConfig{}, "production"); err == nil {
		t.Fatal("NewEmailSender(dev, production) error = nil")
	}
	_, err := NewEmailSender("smtp", SMTPEmailConfig{
		Host: "smtp.resend.com", Port: 587, Username: "resend", From: "fundlive@mail.wrenzeal.top",
	}, "production")
	if err == nil || !strings.Contains(err.Error(), "SMTP_PASSWORD") {
		t.Fatalf("incomplete SMTP error = %v", err)
	}
}
