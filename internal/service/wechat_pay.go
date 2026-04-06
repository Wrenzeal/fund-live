package service

import (
	"context"
	"crypto"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/RomaticDOG/fund/internal/domain"
)

var ErrWeChatPayNotConfigured = errors.New("wechat pay is not configured")

type WeChatPayConfig struct {
	Enabled                     bool
	AppID                       string
	MerchantID                  string
	MerchantCertificateSerialNo string
	MerchantPrivateKeyPath      string
	APIV3Key                    string
	NotifyURL                   string
	PlatformCertificatePath     string
	PlatformPublicKeyPath       string
	PlatformSerialNo            string
	HTTPTimeout                 time.Duration
}

func DefaultWeChatPayConfig() WeChatPayConfig {
	return WeChatPayConfig{
		Enabled:     false,
		HTTPTimeout: 15 * time.Second,
	}
}

func (c WeChatPayConfig) IsCreateConfigured() bool {
	return c.Enabled &&
		strings.TrimSpace(c.AppID) != "" &&
		strings.TrimSpace(c.MerchantID) != "" &&
		strings.TrimSpace(c.MerchantCertificateSerialNo) != "" &&
		strings.TrimSpace(c.MerchantPrivateKeyPath) != "" &&
		strings.TrimSpace(c.NotifyURL) != ""
}

func (c WeChatPayConfig) IsQueryConfigured() bool {
	return c.Enabled &&
		strings.TrimSpace(c.MerchantID) != "" &&
		strings.TrimSpace(c.MerchantCertificateSerialNo) != "" &&
		strings.TrimSpace(c.MerchantPrivateKeyPath) != ""
}

func (c WeChatPayConfig) IsNotifyConfigured() bool {
	if !c.Enabled || strings.TrimSpace(c.APIV3Key) == "" {
		return false
	}
	return strings.TrimSpace(c.PlatformPublicKeyPath) != "" || strings.TrimSpace(c.PlatformCertificatePath) != ""
}

type WeChatPayClient interface {
	CreateNativeOrder(ctx context.Context, input WeChatNativeOrderInput) (*WeChatNativeOrderResult, error)
	QueryOrderByOutTradeNo(ctx context.Context, outTradeNo string) (*WeChatOrderStatusResult, error)
	ParsePaymentNotify(headers map[string]string, body []byte) (*WeChatPaymentNotifyResult, error)
}

type WeChatNativeOrderInput struct {
	OutTradeNo  string
	Description string
	AmountFen   int64
	Currency    string
	ExpiresAt   time.Time
}

type WeChatNativeOrderResult struct {
	CodeURL string
}

type WeChatOrderStatusResult struct {
	Status        domain.VIPOrderStatus
	TradeState    string
	TransactionID string
	SuccessTime   *time.Time
}

type WeChatPaymentNotifyResult struct {
	OrderNo       string
	TransactionID string
	Status        domain.VIPOrderStatus
	TradeState    string
	SuccessTime   *time.Time
	NotifyID      string
	RawPayload    string
}

type WeChatPayClientImpl struct {
	config             WeChatPayConfig
	httpClient         *http.Client
	merchantPrivateKey *rsa.PrivateKey
	platformPublicKey  *rsa.PublicKey
}

func NewWeChatPayClient(config WeChatPayConfig) (*WeChatPayClientImpl, error) {
	if config.HTTPTimeout <= 0 {
		config.HTTPTimeout = 15 * time.Second
	}

	client := &WeChatPayClientImpl{
		config:     config,
		httpClient: &http.Client{Timeout: config.HTTPTimeout},
	}

	if path := strings.TrimSpace(config.MerchantPrivateKeyPath); path != "" {
		privateKey, err := loadRSAPrivateKeyFromPEM(path)
		if err != nil {
			return nil, fmt.Errorf("load wechat pay merchant private key: %w", err)
		}
		client.merchantPrivateKey = privateKey
	}

	var publicKey *rsa.PublicKey
	var err error
	switch {
	case strings.TrimSpace(config.PlatformPublicKeyPath) != "":
		publicKey, err = loadRSAPublicKeyFromPEM(config.PlatformPublicKeyPath)
	case strings.TrimSpace(config.PlatformCertificatePath) != "":
		publicKey, err = loadRSAPublicKeyFromCertificate(config.PlatformCertificatePath)
	}
	if err != nil {
		return nil, fmt.Errorf("load wechat pay platform public key: %w", err)
	}
	client.platformPublicKey = publicKey

	return client, nil
}

func (c *WeChatPayClientImpl) CreateNativeOrder(ctx context.Context, input WeChatNativeOrderInput) (*WeChatNativeOrderResult, error) {
	if c == nil || !c.config.IsCreateConfigured() || c.merchantPrivateKey == nil {
		return nil, ErrWeChatPayNotConfigured
	}

	payload := map[string]any{
		"appid":        c.config.AppID,
		"mchid":        c.config.MerchantID,
		"description":  input.Description,
		"out_trade_no": input.OutTradeNo,
		"notify_url":   c.config.NotifyURL,
		"amount": map[string]any{
			"total":    input.AmountFen,
			"currency": firstNonEmpty(strings.TrimSpace(input.Currency), "CNY"),
		},
	}
	if !input.ExpiresAt.IsZero() {
		payload["time_expire"] = input.ExpiresAt.UTC().Format(time.RFC3339)
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal wechat native order payload: %w", err)
	}

	responseBody, _, err := c.doSignedJSONRequest(ctx, http.MethodPost, "/v3/pay/transactions/native", body)
	if err != nil {
		return nil, err
	}

	var response struct {
		CodeURL string `json:"code_url"`
	}
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("decode wechat native order response: %w", err)
	}
	if strings.TrimSpace(response.CodeURL) == "" {
		return nil, fmt.Errorf("wechat native order response missing code_url")
	}

	return &WeChatNativeOrderResult{
		CodeURL: response.CodeURL,
	}, nil
}

func (c *WeChatPayClientImpl) QueryOrderByOutTradeNo(ctx context.Context, outTradeNo string) (*WeChatOrderStatusResult, error) {
	if c == nil || !c.config.IsQueryConfigured() || c.merchantPrivateKey == nil {
		return nil, ErrWeChatPayNotConfigured
	}

	queryPath := fmt.Sprintf("/v3/pay/transactions/out-trade-no/%s?mchid=%s",
		url.PathEscape(strings.TrimSpace(outTradeNo)),
		url.QueryEscape(c.config.MerchantID),
	)

	responseBody, statusCode, err := c.doSignedJSONRequest(ctx, http.MethodGet, queryPath, nil)
	if err != nil {
		return nil, err
	}
	if statusCode == http.StatusNoContent {
		return nil, nil
	}

	var response struct {
		TradeState    string `json:"trade_state"`
		TransactionID string `json:"transaction_id"`
		SuccessTime   string `json:"success_time"`
	}
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("decode wechat order query response: %w", err)
	}

	successTime, err := parseOptionalRFC3339(response.SuccessTime)
	if err != nil {
		return nil, fmt.Errorf("parse wechat success time: %w", err)
	}

	return &WeChatOrderStatusResult{
		Status:        mapWeChatTradeState(response.TradeState),
		TradeState:    response.TradeState,
		TransactionID: response.TransactionID,
		SuccessTime:   successTime,
	}, nil
}

func (c *WeChatPayClientImpl) ParsePaymentNotify(headers map[string]string, body []byte) (*WeChatPaymentNotifyResult, error) {
	if c == nil || !c.config.IsNotifyConfigured() || c.platformPublicKey == nil {
		return nil, ErrWeChatPayNotConfigured
	}

	timestamp := strings.TrimSpace(headers["Wechatpay-Timestamp"])
	nonce := strings.TrimSpace(headers["Wechatpay-Nonce"])
	signature := strings.TrimSpace(headers["Wechatpay-Signature"])
	serial := strings.TrimSpace(headers["Wechatpay-Serial"])
	if timestamp == "" || nonce == "" || signature == "" {
		return nil, fmt.Errorf("wechat notify headers are incomplete")
	}
	if expected := strings.TrimSpace(c.config.PlatformSerialNo); expected != "" && serial != "" && serial != expected {
		return nil, fmt.Errorf("wechat notify serial mismatch")
	}

	message := timestamp + "\n" + nonce + "\n" + string(body) + "\n"
	if err := verifyRSASignature(c.platformPublicKey, message, signature); err != nil {
		return nil, fmt.Errorf("verify wechat notify signature: %w", err)
	}

	var envelope struct {
		ID       string `json:"id"`
		Resource struct {
			Algorithm      string `json:"algorithm"`
			Ciphertext     string `json:"ciphertext"`
			AssociatedData string `json:"associated_data"`
			Nonce          string `json:"nonce"`
		} `json:"resource"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, fmt.Errorf("decode wechat notify envelope: %w", err)
	}

	plaintext, err := decryptWeChatPayResource(
		c.config.APIV3Key,
		envelope.Resource.Nonce,
		envelope.Resource.AssociatedData,
		envelope.Resource.Ciphertext,
	)
	if err != nil {
		return nil, fmt.Errorf("decrypt wechat notify resource: %w", err)
	}

	var transaction struct {
		OutTradeNo    string `json:"out_trade_no"`
		TransactionID string `json:"transaction_id"`
		TradeState    string `json:"trade_state"`
		SuccessTime   string `json:"success_time"`
	}
	if err := json.Unmarshal(plaintext, &transaction); err != nil {
		return nil, fmt.Errorf("decode wechat notify transaction: %w", err)
	}

	successTime, err := parseOptionalRFC3339(transaction.SuccessTime)
	if err != nil {
		return nil, fmt.Errorf("parse wechat notify success time: %w", err)
	}

	return &WeChatPaymentNotifyResult{
		OrderNo:       transaction.OutTradeNo,
		TransactionID: transaction.TransactionID,
		Status:        mapWeChatTradeState(transaction.TradeState),
		TradeState:    transaction.TradeState,
		SuccessTime:   successTime,
		NotifyID:      envelope.ID,
		RawPayload:    string(plaintext),
	}, nil
}

func (c *WeChatPayClientImpl) doSignedJSONRequest(ctx context.Context, method, pathWithQuery string, body []byte) ([]byte, int, error) {
	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	nonce, err := generateToken(12)
	if err != nil {
		return nil, 0, fmt.Errorf("generate wechat nonce: %w", err)
	}

	authHeader, err := buildWeChatAuthorizationHeader(
		c.merchantPrivateKey,
		c.config.MerchantID,
		c.config.MerchantCertificateSerialNo,
		method,
		pathWithQuery,
		timestamp,
		nonce,
		string(body),
	)
	if err != nil {
		return nil, 0, err
	}

	requestURL := "https://api.mch.weixin.qq.com" + pathWithQuery
	req, err := http.NewRequestWithContext(ctx, method, requestURL, strings.NewReader(string(body)))
	if err != nil {
		return nil, 0, fmt.Errorf("build wechat request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", authHeader)
	req.Header.Set("User-Agent", "FundLive/2026.4.5")
	req.Header.Set("Wechatpay-Serial", c.config.MerchantCertificateSerialNo)
	if len(body) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("wechat request failed: %w", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("read wechat response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, resp.StatusCode, fmt.Errorf("wechat api returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(responseBody)))
	}

	return responseBody, resp.StatusCode, nil
}

func buildWeChatAuthorizationHeader(
	privateKey *rsa.PrivateKey,
	merchantID string,
	certificateSerialNo string,
	method string,
	pathWithQuery string,
	timestamp string,
	nonce string,
	body string,
) (string, error) {
	if privateKey == nil {
		return "", ErrWeChatPayNotConfigured
	}

	message := strings.ToUpper(strings.TrimSpace(method)) + "\n" +
		pathWithQuery + "\n" +
		timestamp + "\n" +
		nonce + "\n" +
		body + "\n"

	hashed := sha256.Sum256([]byte(message))
	signature, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, hashed[:])
	if err != nil {
		return "", fmt.Errorf("sign wechat request: %w", err)
	}

	return fmt.Sprintf(
		`WECHATPAY2-SHA256-RSA2048 mchid="%s",nonce_str="%s",signature="%s",timestamp="%s",serial_no="%s"`,
		merchantID,
		nonce,
		base64.StdEncoding.EncodeToString(signature),
		timestamp,
		certificateSerialNo,
	), nil
}

func verifyRSASignature(publicKey *rsa.PublicKey, message string, signature string) error {
	decodedSignature, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return fmt.Errorf("decode signature: %w", err)
	}

	sum := sha256.Sum256([]byte(message))
	if err := rsa.VerifyPKCS1v15(publicKey, crypto.SHA256, sum[:], decodedSignature); err != nil {
		return err
	}
	return nil
}

func decryptWeChatPayResource(apiV3Key, nonce, associatedData, ciphertext string) ([]byte, error) {
	key := []byte(strings.TrimSpace(apiV3Key))
	if len(key) != 32 {
		return nil, fmt.Errorf("invalid api v3 key length: got %d, want 32", len(key))
	}

	decodedCiphertext, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return nil, fmt.Errorf("decode ciphertext: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	plaintext, err := gcm.Open(nil, []byte(nonce), decodedCiphertext, []byte(associatedData))
	if err != nil {
		return nil, err
	}
	return plaintext, nil
}

func loadRSAPrivateKeyFromPEM(path string) (*rsa.PrivateKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("invalid pem private key")
	}

	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}

	privateKey, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	rsaKey, ok := privateKey.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("private key is not RSA")
	}
	return rsaKey, nil
}

func loadRSAPublicKeyFromPEM(path string) (*rsa.PublicKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("invalid pem public key")
	}

	key, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	rsaKey, ok := key.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("public key is not RSA")
	}
	return rsaKey, nil
}

func loadRSAPublicKeyFromCertificate(path string) (*rsa.PublicKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("invalid certificate pem")
	}

	certificate, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, err
	}

	rsaKey, ok := certificate.PublicKey.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("certificate public key is not RSA")
	}
	return rsaKey, nil
}

func parseOptionalRFC3339(raw string) (*time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}

	parsed, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func mapWeChatTradeState(tradeState string) domain.VIPOrderStatus {
	switch strings.ToUpper(strings.TrimSpace(tradeState)) {
	case "SUCCESS":
		return domain.VIPOrderStatusPaid
	case "CLOSED", "REVOKED":
		return domain.VIPOrderStatusClosed
	case "PAYERROR":
		return domain.VIPOrderStatusFailed
	default:
		return domain.VIPOrderStatusPendingPayment
	}
}
