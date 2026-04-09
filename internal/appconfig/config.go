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
	Quote    QuoteConfig    `yaml:"quote" json:"quote"`
	Auth     AuthConfig     `yaml:"auth" json:"auth"`
	Payment  PaymentConfig  `yaml:"payment" json:"payment"`
}

type ServerConfig struct {
	Port           string   `yaml:"port" json:"port"`
	AllowedOrigins []string `yaml:"allowed_origins" json:"allowed_origins"`
}

type StorageConfig struct {
	Mode string `yaml:"mode" json:"mode"`
}

type QuoteConfig struct {
	DefaultSource string `yaml:"default_source" json:"default_source"`
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
	CookieName      string `yaml:"cookie_name" json:"cookie_name"`
	CookieSecure    bool   `yaml:"cookie_secure" json:"cookie_secure"`
	SessionTTLHours int    `yaml:"session_ttl_hours" json:"session_ttl_hours"`
	GoogleClientID  string `yaml:"google_client_id" json:"google_client_id"`
}

type PaymentConfig struct {
	WeChatPay WeChatPayConfig `yaml:"wechat_pay" json:"wechat_pay"`
}

type WeChatPayConfig struct {
	Enabled                     bool   `yaml:"enabled" json:"enabled"`
	AppID                       string `yaml:"app_id" json:"app_id"`
	MerchantID                  string `yaml:"merchant_id" json:"merchant_id"`
	MerchantCertificateSerialNo string `yaml:"merchant_certificate_serial_no" json:"merchant_certificate_serial_no"`
	MerchantPrivateKeyPath      string `yaml:"merchant_private_key_path" json:"merchant_private_key_path"`
	APIV3Key                    string `yaml:"api_v3_key" json:"api_v3_key"`
	NotifyURL                   string `yaml:"notify_url" json:"notify_url"`
	PlatformCertificatePath     string `yaml:"platform_certificate_path" json:"platform_certificate_path"`
	PlatformPublicKeyPath       string `yaml:"platform_public_key_path" json:"platform_public_key_path"`
	PlatformSerialNo            string `yaml:"platform_serial_no" json:"platform_serial_no"`
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
