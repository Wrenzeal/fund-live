package service

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

const googleJWKSURL = "https://www.googleapis.com/oauth2/v3/certs"

// GoogleIdentityClaims contains the verified identity information extracted from a Google ID token.
type GoogleIdentityClaims struct {
	Subject       string
	Email         string
	EmailVerified bool
	Name          string
	Picture       string
}

// GoogleIDTokenVerifier verifies Google ID tokens.
type GoogleIDTokenVerifier interface {
	VerifyIDToken(ctx context.Context, idToken string) (*GoogleIdentityClaims, error)
}

type googleIDTokenVerifier struct {
	clientID string
	client   *http.Client

	mu         sync.RWMutex
	keys       map[string]*rsa.PublicKey
	keysExpiry time.Time
}

type googleJWTHeader struct {
	Algorithm string `json:"alg"`
	KeyID     string `json:"kid"`
	Type      string `json:"typ"`
}

type googleJWKSDocument struct {
	Keys []googleJWK `json:"keys"`
}

type googleJWK struct {
	KeyType string `json:"kty"`
	KeyID   string `json:"kid"`
	Use     string `json:"use"`
	N       string `json:"n"`
	E       string `json:"e"`
}

type googleIDTokenPayload struct {
	Iss           string `json:"iss"`
	Azp           string `json:"azp"`
	Aud           string `json:"aud"`
	Sub           string `json:"sub"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
	Exp           int64  `json:"exp"`
	Iat           int64  `json:"iat"`
}

func newGoogleIDTokenVerifier(clientID string) GoogleIDTokenVerifier {
	clientID = strings.TrimSpace(clientID)
	if clientID == "" {
		return nil
	}

	return &googleIDTokenVerifier{
		clientID: clientID,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		keys: make(map[string]*rsa.PublicKey),
	}
}

func (v *googleIDTokenVerifier) VerifyIDToken(ctx context.Context, idToken string) (*GoogleIdentityClaims, error) {
	idToken = strings.TrimSpace(idToken)
	if idToken == "" {
		return nil, ErrInvalidGoogleToken
	}

	parts := strings.Split(idToken, ".")
	if len(parts) != 3 {
		return nil, ErrInvalidGoogleToken
	}

	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, ErrInvalidGoogleToken
	}

	var header googleJWTHeader
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		return nil, ErrInvalidGoogleToken
	}
	if header.Algorithm != "RS256" || header.KeyID == "" {
		return nil, ErrInvalidGoogleToken
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, ErrInvalidGoogleToken
	}

	var payload googleIDTokenPayload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return nil, ErrInvalidGoogleToken
	}

	if err := v.validatePayload(payload); err != nil {
		return nil, err
	}

	publicKey, err := v.getPublicKey(ctx, header.KeyID)
	if err != nil {
		return nil, err
	}

	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, ErrInvalidGoogleToken
	}

	signingInput := parts[0] + "." + parts[1]
	hash := sha256.Sum256([]byte(signingInput))
	if err := rsa.VerifyPKCS1v15(publicKey, crypto.SHA256, hash[:], signature); err != nil {
		return nil, ErrInvalidGoogleToken
	}

	return &GoogleIdentityClaims{
		Subject:       payload.Sub,
		Email:         strings.ToLower(strings.TrimSpace(payload.Email)),
		EmailVerified: payload.EmailVerified,
		Name:          strings.TrimSpace(payload.Name),
		Picture:       strings.TrimSpace(payload.Picture),
	}, nil
}

func (v *googleIDTokenVerifier) validatePayload(payload googleIDTokenPayload) error {
	if payload.Sub == "" {
		return ErrInvalidGoogleToken
	}
	if payload.Iss != "https://accounts.google.com" && payload.Iss != "accounts.google.com" {
		return ErrInvalidGoogleToken
	}
	if payload.Aud != v.clientID {
		return ErrInvalidGoogleToken
	}
	if payload.Exp <= time.Now().Unix() {
		return ErrInvalidGoogleToken
	}
	return nil
}

func (v *googleIDTokenVerifier) getPublicKey(ctx context.Context, keyID string) (*rsa.PublicKey, error) {
	v.mu.RLock()
	if time.Now().Before(v.keysExpiry) {
		if key, ok := v.keys[keyID]; ok {
			v.mu.RUnlock()
			return key, nil
		}
	}
	v.mu.RUnlock()

	if err := v.refreshKeys(ctx); err != nil {
		return nil, err
	}

	v.mu.RLock()
	defer v.mu.RUnlock()
	key, ok := v.keys[keyID]
	if !ok {
		return nil, ErrInvalidGoogleToken
	}
	return key, nil
}

func (v *googleIDTokenVerifier) refreshKeys(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, googleJWKSURL, nil)
	if err != nil {
		return fmt.Errorf("create google jwks request: %w", err)
	}

	resp, err := v.client.Do(req)
	if err != nil {
		return fmt.Errorf("fetch google jwks: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("fetch google jwks: unexpected status %d", resp.StatusCode)
	}

	var document googleJWKSDocument
	if err := json.NewDecoder(resp.Body).Decode(&document); err != nil {
		return fmt.Errorf("decode google jwks: %w", err)
	}

	keys := make(map[string]*rsa.PublicKey, len(document.Keys))
	for _, jwk := range document.Keys {
		if jwk.KeyType != "RSA" || jwk.Use != "sig" || jwk.KeyID == "" {
			continue
		}

		key, err := parseGoogleRSAPublicKey(jwk)
		if err != nil {
			continue
		}
		keys[jwk.KeyID] = key
	}

	if len(keys) == 0 {
		return errors.New("google jwks contained no usable keys")
	}

	v.mu.Lock()
	v.keys = keys
	v.keysExpiry = time.Now().Add(parseGoogleKeysMaxAge(resp.Header.Get("Cache-Control")))
	v.mu.Unlock()

	return nil
}

func parseGoogleRSAPublicKey(jwk googleJWK) (*rsa.PublicKey, error) {
	modulusBytes, err := base64.RawURLEncoding.DecodeString(jwk.N)
	if err != nil {
		return nil, err
	}
	exponentBytes, err := base64.RawURLEncoding.DecodeString(jwk.E)
	if err != nil {
		return nil, err
	}

	modulus := new(big.Int).SetBytes(modulusBytes)
	exponent := new(big.Int).SetBytes(exponentBytes)
	if modulus.Sign() == 0 || exponent.Sign() == 0 {
		return nil, ErrInvalidGoogleToken
	}

	return &rsa.PublicKey{
		N: modulus,
		E: int(exponent.Int64()),
	}, nil
}

func parseGoogleKeysMaxAge(cacheControl string) time.Duration {
	const fallback = time.Hour

	directives := strings.Split(cacheControl, ",")
	for _, directive := range directives {
		part := strings.TrimSpace(directive)
		if !strings.HasPrefix(part, "max-age=") {
			continue
		}

		seconds, err := strconv.Atoi(strings.TrimPrefix(part, "max-age="))
		if err != nil || seconds <= 0 {
			return fallback
		}
		return time.Duration(seconds) * time.Second
	}

	return fallback
}
