package service

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"html"
	"io"
	"log"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net"
	"net/mail"
	"net/smtp"
	"net/textproto"
	"strconv"
	"strings"
	"time"
)

const defaultSMTPTimeout = 15 * time.Second

// EmailSender delivers one-time authentication codes.
type EmailSender interface {
	SendVerificationCode(ctx context.Context, email, code string, ttl time.Duration) error
}

// SMTPEmailConfig describes a standard SMTP transport. Resend uses STARTTLS
// on port 587 with username "resend" and an API key as the password.
type SMTPEmailConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
	FromName string
	Security string
	Timeout  time.Duration
}

type SMTPEmailSender struct {
	config SMTPEmailConfig
}

// DevEmailSender intentionally performs no delivery. The service exposes the
// code in the development-only response instead of writing secrets to logs.
type DevEmailSender struct{}

func NewEmailSender(driver string, smtpConfig SMTPEmailConfig, appEnv string) (EmailSender, error) {
	driver = strings.ToLower(strings.TrimSpace(driver))
	appEnv = strings.ToLower(strings.TrimSpace(appEnv))
	if driver == "" {
		if appEnv == "production" {
			driver = "smtp"
		} else {
			driver = "dev"
		}
	}
	switch driver {
	case "dev":
		if appEnv == "production" {
			return nil, errors.New("AUTH_EMAIL_DRIVER=dev is not allowed in production")
		}
		return DevEmailSender{}, nil
	case "smtp":
		return NewSMTPEmailSender(smtpConfig)
	default:
		return nil, fmt.Errorf("unsupported AUTH_EMAIL_DRIVER %q", driver)
	}
}

func NewSMTPEmailSender(config SMTPEmailConfig) (*SMTPEmailSender, error) {
	config.Host = strings.TrimSpace(config.Host)
	config.Username = strings.TrimSpace(config.Username)
	config.From = strings.TrimSpace(config.From)
	config.FromName = strings.TrimSpace(config.FromName)
	if config.Port == 0 {
		config.Port = 587
	}
	config.Security = normalizeSMTPSecurity(config.Security, config.Port)
	if config.Timeout <= 0 {
		config.Timeout = defaultSMTPTimeout
	}
	if config.Host == "" {
		return nil, errors.New("SMTP_HOST is required")
	}
	if config.From == "" {
		return nil, errors.New("SMTP_FROM is required")
	}
	if config.Username != "" && config.Password == "" {
		return nil, errors.New("SMTP_PASSWORD is required when SMTP_USERNAME is configured")
	}
	if _, err := mail.ParseAddress(config.From); err != nil {
		return nil, fmt.Errorf("SMTP_FROM is invalid: %w", err)
	}
	if config.Security != "tls" && config.Security != "starttls" && config.Security != "none" {
		return nil, errors.New("SMTP_SECURITY must be one of auto, tls, starttls, none")
	}
	return &SMTPEmailSender{config: config}, nil
}

func (DevEmailSender) SendVerificationCode(context.Context, string, string, time.Duration) error {
	return nil
}

func (s *SMTPEmailSender) SendVerificationCode(ctx context.Context, email, code string, ttl time.Duration) error {
	if s == nil {
		return errors.New("smtp email sender is not configured")
	}
	message, err := buildVerificationEmail(s.config.From, s.config.FromName, email, code, ttl)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, s.config.Timeout)
	defer cancel()

	client, err := s.connect(ctx)
	if err != nil {
		return err
	}
	defer client.Close()

	if s.config.Username != "" {
		auth := smtp.PlainAuth("", s.config.Username, s.config.Password, s.config.Host)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("smtp auth: %w", err)
		}
	}
	if err := client.Mail(s.config.From); err != nil {
		return fmt.Errorf("smtp mail from: %w", err)
	}
	if err := client.Rcpt(email); err != nil {
		return fmt.Errorf("smtp rcpt to: %w", err)
	}
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}
	if _, err := io.Copy(writer, bytes.NewReader(message)); err != nil {
		_ = writer.Close()
		return fmt.Errorf("smtp write body: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("smtp close body: %w", err)
	}
	if err := client.Quit(); err != nil {
		// DATA close means the server accepted the message. A later QUIT error
		// must not invalidate a code that may already be in the recipient inbox.
		log.Printf("smtp quit after message acceptance: %v", err)
	}
	return nil
}

func (s *SMTPEmailSender) connect(ctx context.Context) (*smtp.Client, error) {
	address := net.JoinHostPort(s.config.Host, strconv.Itoa(s.config.Port))
	tlsConfig := &tls.Config{ServerName: s.config.Host, MinVersion: tls.VersionTLS12}
	if s.config.Security == "tls" {
		connection, err := tls.DialWithDialer(&net.Dialer{Timeout: s.config.Timeout}, "tcp", address, tlsConfig)
		if err != nil {
			return nil, fmt.Errorf("smtp tls dial: %w", err)
		}
		client, err := smtp.NewClient(connection, s.config.Host)
		if err != nil {
			_ = connection.Close()
			return nil, fmt.Errorf("smtp client: %w", err)
		}
		return client, nil
	}

	connection, err := (&net.Dialer{Timeout: s.config.Timeout}).DialContext(ctx, "tcp", address)
	if err != nil {
		return nil, fmt.Errorf("smtp dial: %w", err)
	}
	client, err := smtp.NewClient(connection, s.config.Host)
	if err != nil {
		_ = connection.Close()
		return nil, fmt.Errorf("smtp client: %w", err)
	}
	if s.config.Security == "starttls" {
		if ok, _ := client.Extension("STARTTLS"); !ok {
			_ = client.Close()
			return nil, errors.New("smtp server does not support STARTTLS")
		}
		if err := client.StartTLS(tlsConfig); err != nil {
			_ = client.Close()
			return nil, fmt.Errorf("smtp starttls: %w", err)
		}
	}
	return client, nil
}

func normalizeSMTPSecurity(security string, port int) string {
	security = strings.ToLower(strings.TrimSpace(security))
	if security == "" || security == "auto" {
		if port == 465 {
			return "tls"
		}
		return "starttls"
	}
	return security
}

func buildVerificationEmail(from, fromName, to, code string, ttl time.Duration) ([]byte, error) {
	if _, err := mail.ParseAddress(to); err != nil {
		return nil, fmt.Errorf("recipient email is invalid: %w", err)
	}
	fromAddress := mail.Address{Name: strings.TrimSpace(fromName), Address: from}
	toAddress := mail.Address{Address: to}
	subject := mime.QEncoding.Encode("UTF-8", fmt.Sprintf("FundLive 登录验证码：%s", code))
	minutes := int(ttl.Round(time.Minute).Minutes())
	if minutes <= 0 {
		minutes = 10
	}
	plainBody, htmlBody := verificationEmailBodies(code, minutes)

	var body bytes.Buffer
	alternative := multipart.NewWriter(&body)
	if err := writeEmailPart(alternative, "text/plain; charset=UTF-8", plainBody); err != nil {
		return nil, err
	}
	if err := writeEmailPart(alternative, "text/html; charset=UTF-8", htmlBody); err != nil {
		return nil, err
	}
	if err := alternative.Close(); err != nil {
		return nil, fmt.Errorf("close email multipart body: %w", err)
	}

	var message bytes.Buffer
	fmt.Fprintf(&message, "From: %s\r\n", fromAddress.String())
	fmt.Fprintf(&message, "To: %s\r\n", toAddress.String())
	fmt.Fprintf(&message, "Subject: %s\r\n", subject)
	fmt.Fprint(&message, "MIME-Version: 1.0\r\n")
	fmt.Fprintf(&message, "Content-Type: multipart/alternative; boundary=%q\r\n\r\n", alternative.Boundary())
	message.Write(body.Bytes())
	return message.Bytes(), nil
}

func writeEmailPart(writer *multipart.Writer, contentType, content string) error {
	header := make(textproto.MIMEHeader)
	header.Set("Content-Type", contentType)
	header.Set("Content-Transfer-Encoding", "quoted-printable")
	part, err := writer.CreatePart(header)
	if err != nil {
		return fmt.Errorf("create email part %s: %w", contentType, err)
	}
	encoded := quotedprintable.NewWriter(part)
	if _, err := io.WriteString(encoded, content); err != nil {
		_ = encoded.Close()
		return fmt.Errorf("write email part %s: %w", contentType, err)
	}
	if err := encoded.Close(); err != nil {
		return fmt.Errorf("close email part %s: %w", contentType, err)
	}
	return nil
}

func verificationEmailBodies(code string, minutes int) (string, string) {
	safeCode := html.EscapeString(code)
	plain := fmt.Sprintf(`FundLive 登录验证码

验证码：%s

验证码将在 %d 分钟后失效，请勿转发给任何人。
如果不是你本人发起登录，可以安全地忽略这封邮件。

FundLive · 你的基金估值系统
https://fund.wrenzeal.top
`, code, minutes)

	htmlBody := fmt.Sprintf(`<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <meta name="color-scheme" content="light">
  <title>FundLive 登录验证码</title>
</head>
<body style="margin:0;padding:0;background:#eaf5f8;color:#102a38;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI','PingFang SC','Microsoft YaHei',Arial,sans-serif;">
  <div style="display:none;max-height:0;overflow:hidden;opacity:0;color:transparent;">使用此验证码登录 FundLive，继续查看你的自选和持仓。</div>
  <table role="presentation" width="100%%" cellspacing="0" cellpadding="0" border="0" style="width:100%%;background:#eaf5f8;">
    <tr><td align="center" style="padding:40px 16px;">
      <table role="presentation" width="100%%" cellspacing="0" cellpadding="0" border="0" style="width:100%%;max-width:560px;">
        <tr><td style="padding:0 8px 18px;font-size:22px;font-weight:800;letter-spacing:-0.02em;color:#0f6476;">FundLive</td></tr>
        <tr><td style="overflow:hidden;border:1px solid #c8e2e8;border-radius:28px;background:#ffffff;box-shadow:0 18px 50px rgba(15,100,118,0.12);">
          <table role="presentation" width="100%%" cellspacing="0" cellpadding="0" border="0">
            <tr><td style="height:7px;background:#14b8a6;font-size:0;line-height:0;">&nbsp;</td></tr>
            <tr><td style="padding:42px 44px 18px;">
              <div style="font-size:11px;font-weight:700;letter-spacing:0.16em;color:#0e7490;text-transform:uppercase;">Secure access</div>
              <h1 style="margin:15px 0 0;font-size:30px;line-height:1.3;font-weight:750;letter-spacing:-0.03em;color:#102a38;">登录你的基金观察账户</h1>
              <p style="margin:17px 0 0;font-size:15px;line-height:1.8;color:#52707d;">输入下面的验证码，继续查看你的自选基金、持仓记录和个性化设置。</p>
            </td></tr>
            <tr><td style="padding:8px 44px 30px;">
              <div style="border:1px solid #9ed6dd;border-radius:20px;background:#e6f7f7;padding:25px 20px;text-align:center;">
                <div style="font-size:11px;font-weight:700;letter-spacing:0.14em;color:#0e7490;text-transform:uppercase;">Verification code</div>
                <div style="margin-top:10px;font-family:'SFMono-Regular',Consolas,'Liberation Mono',monospace;font-size:38px;line-height:1.2;font-weight:750;letter-spacing:0.22em;color:#0f6476;">%s</div>
              </div>
            </td></tr>
            <tr><td style="padding:0 44px 42px;">
              <div style="border-top:1px solid #dcebef;padding-top:23px;font-size:13px;line-height:1.75;color:#66808b;">
                验证码将在 <strong style="color:#0f6476;">%d 分钟</strong>后失效，请勿转发给任何人。<br>
                如果不是你本人发起登录，可以安全地忽略这封邮件。
              </div>
            </td></tr>
          </table>
        </td></tr>
        <tr><td align="center" style="padding:22px 16px 0;font-size:12px;line-height:1.7;color:#78909a;">FundLive · 你的基金估值系统<br>fund.wrenzeal.top</td></tr>
      </table>
    </td></tr>
  </table>
</body>
</html>`, safeCode, minutes)
	return plain, htmlBody
}
