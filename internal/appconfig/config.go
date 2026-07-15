package appconfig

import (
	"os"
	"path/filepath"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Path     string         `yaml:"-" json:"-"`
	Server   ServerConfig   `yaml:"server" json:"server"`
	Storage  StorageConfig  `yaml:"storage" json:"storage"`
	Database DatabaseConfig `yaml:"database" json:"database"`
	Cache    CacheConfig    `yaml:"cache" json:"cache"`
	Quote    QuoteConfig    `yaml:"quote" json:"quote"`
	Auth     AuthConfig     `yaml:"auth" json:"auth"`
}

type ServerConfig struct {
	Port           string   `yaml:"port" json:"port"`
	AllowedOrigins []string `yaml:"allowed_origins" json:"allowed_origins"`
}

type StorageConfig struct {
	Mode string `yaml:"mode" json:"mode"`
}

type CacheConfig struct {
	RedisURL  string `yaml:"redis_url" json:"-"`
	KeyPrefix string `yaml:"key_prefix" json:"key_prefix"`
}

type QuoteConfig struct {
	DefaultSource  string `yaml:"default_source" json:"default_source"`
	OverseasSource string `yaml:"overseas_source" json:"overseas_source"`
}

type DatabaseConfig struct {
	Host        string `yaml:"host" json:"host"`
	Port        string `yaml:"port" json:"port"`
	User        string `yaml:"user" json:"user"`
	Password    string `yaml:"password" json:"password"`
	Name        string `yaml:"name" json:"name"`
	SSLMode     string `yaml:"ssl_mode" json:"ssl_mode"`
	TimeZone    string `yaml:"timezone" json:"timezone"`
	LogLevel    string `yaml:"log_level" json:"log_level"`
	AutoMigrate *bool  `yaml:"auto_migrate" json:"auto_migrate"`
}

type AuthConfig struct {
	CookieName               string `yaml:"cookie_name" json:"cookie_name"`
	CookieSecure             bool   `yaml:"cookie_secure" json:"cookie_secure"`
	SessionTTLHours          int    `yaml:"session_ttl_hours" json:"session_ttl_hours"`
	GoogleClientID           string `yaml:"google_client_id" json:"google_client_id"`
	AuthAttemptWindowMinutes int    `yaml:"auth_attempt_window_minutes" json:"auth_attempt_window_minutes"`
	MaxPasswordFailures      int    `yaml:"max_password_failures" json:"max_password_failures"`
	MaxRegisterFailures      int    `yaml:"max_register_failures" json:"max_register_failures"`
	MaxGoogleLoginFailures   int    `yaml:"max_google_login_failures" json:"max_google_login_failures"`
	EmailCodeEnabled         bool   `yaml:"email_code_enabled" json:"email_code_enabled"`
	EmailDriver              string `yaml:"email_driver" json:"email_driver"`
	EmailCodeSecret          string `yaml:"email_code_secret" json:"-"`
	EmailCodeTTLMinutes      int    `yaml:"email_code_ttl_minutes" json:"email_code_ttl_minutes"`
	EmailResendCooldownSecs  int    `yaml:"email_resend_cooldown_seconds" json:"email_resend_cooldown_seconds"`
	MaxEmailSendsPerHour     int    `yaml:"max_email_sends_per_hour" json:"max_email_sends_per_hour"`
	MaxIPEmailSendsPerHour   int    `yaml:"max_ip_email_sends_per_hour" json:"max_ip_email_sends_per_hour"`
	MaxEmailCodeFailures     int    `yaml:"max_email_code_failures" json:"max_email_code_failures"`
	SMTPHost                 string `yaml:"smtp_host" json:"smtp_host"`
	SMTPPort                 int    `yaml:"smtp_port" json:"smtp_port"`
	SMTPUsername             string `yaml:"smtp_username" json:"smtp_username"`
	SMTPPassword             string `yaml:"smtp_password" json:"-"`
	SMTPFrom                 string `yaml:"smtp_from" json:"smtp_from"`
	SMTPFromName             string `yaml:"smtp_from_name" json:"smtp_from_name"`
	SMTPSecurity             string `yaml:"smtp_security" json:"smtp_security"`
	SMTPTimeoutSeconds       int    `yaml:"smtp_timeout_seconds" json:"smtp_timeout_seconds"`
}

var (
	loadOnce sync.Once
	cached   *Config
	loadErr  error
)

func DefaultConfigPaths() []string {
	home, _ := os.UserHomeDir()
	exePath, _ := os.Executable()
	exeDir := filepath.Dir(exePath)
	return []string{
		filepath.Join(exeDir, "fundlive.yaml"),
		"fundlive.yaml",
		"fundlive.yml",
		".fundlive.yaml",
		".fundlive.yml",
		"config/fundlive.yaml",
		"config/fundlive.yml",
		filepath.Join(home, ".fundlive", "fundlive.yaml"),
		filepath.Join(home, ".fundlive", "config.yaml"),
	}
}

func LoadConfig() (*Config, error) {
	loadOnce.Do(func() {
		if p := os.Getenv("FUNDLIVE_CONFIG"); p != "" {
			cached, loadErr = LoadConfigFromFile(p)
			return
		}

		for _, path := range DefaultConfigPaths() {
			if _, err := os.Stat(path); err == nil {
				cached, loadErr = LoadConfigFromFile(path)
				return
			}
		}
	})
	return cached, loadErr
}

func LoadConfigFromFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	cfg.Path = path
	return &cfg, nil
}

func NormalizePort(port string) string {
	if port == "" {
		return ":8080"
	}
	if strings.HasPrefix(port, ":") {
		return port
	}
	return ":" + port
}
