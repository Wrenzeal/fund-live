// Package database provides database connection and initialization.
package database

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/RomaticDOG/fund/internal/appconfig"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Config holds database connection configuration.
type Config struct {
	Host        string
	Port        string
	User        string
	Password    string
	DBName      string
	SSLMode     string
	TimeZone    string
	LogLevel    string
	AutoMigrate bool
}

// DefaultConfig returns fallback database settings.
// Runtime should prefer fundlive.yaml, with environment variables overriding that file when present.
func DefaultConfig() Config {
	cfg := Config{
		Host:        "localhost",
		Port:        "15432",
		User:        "fundlive",
		Password:    "fundlive_secret",
		DBName:      "fundlive",
		SSLMode:     "disable",
		TimeZone:    "Asia/Shanghai",
		LogLevel:    "warn",
		AutoMigrate: false,
	}

	if fileCfg, err := appconfig.LoadConfig(); err == nil && fileCfg != nil {
		if fileCfg.Database.Host != "" {
			cfg.Host = fileCfg.Database.Host
		}
		if fileCfg.Database.Port != "" {
			cfg.Port = fileCfg.Database.Port
		}
		if fileCfg.Database.User != "" {
			cfg.User = fileCfg.Database.User
		}
		if fileCfg.Database.Password != "" {
			cfg.Password = fileCfg.Database.Password
		}
		if fileCfg.Database.Name != "" {
			cfg.DBName = fileCfg.Database.Name
		}
		if fileCfg.Database.SSLMode != "" {
			cfg.SSLMode = fileCfg.Database.SSLMode
		}
		if fileCfg.Database.TimeZone != "" {
			cfg.TimeZone = fileCfg.Database.TimeZone
		}
		if fileCfg.Database.LogLevel != "" {
			cfg.LogLevel = fileCfg.Database.LogLevel
		}
		if fileCfg.Database.AutoMigrate != nil {
			cfg.AutoMigrate = *fileCfg.Database.AutoMigrate
		}
	}

	cfg.Host = getEnv("DB_HOST", cfg.Host)
	cfg.Port = getEnv("DB_PORT", cfg.Port)
	cfg.User = getEnv("DB_USER", cfg.User)
	cfg.Password = getEnv("DB_PASSWORD", cfg.Password)
	cfg.DBName = getEnv("DB_NAME", cfg.DBName)
	cfg.SSLMode = getEnv("DB_SSLMODE", cfg.SSLMode)
	cfg.TimeZone = getEnv("DB_TIMEZONE", cfg.TimeZone)
	cfg.LogLevel = getEnv("DB_LOG_LEVEL", cfg.LogLevel)
	if val := os.Getenv("DB_AUTO_MIGRATE"); val != "" {
		if parsed, err := strconv.ParseBool(val); err == nil {
			cfg.AutoMigrate = parsed
		}
	}

	return cfg
}

// DSN returns the PostgreSQL connection string.
func (c Config) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s TimeZone=%s",
		c.Host, c.Port, c.User, c.Password, c.DBName, c.SSLMode, c.TimeZone,
	)
}

// DB is the global database instance.
var DB *gorm.DB

// InitDB initializes the database connection and runs auto-migration.
// It connects to PostgreSQL using the provided configuration and
// automatically creates/updates tables based on the model structs.
func InitDB(cfg Config, models ...interface{}) (*gorm.DB, error) {
	// Configure GORM logger
	gormLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:             200 * time.Millisecond,
			LogLevel:                  parseGORMLogLevel(cfg.LogLevel),
			IgnoreRecordNotFoundError: true,
			Colorful:                  true,
		},
	)

	// Connect to PostgreSQL
	db, err := gorm.Open(postgres.Open(cfg.DSN()), &gorm.Config{
		Logger: gormLogger,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	// Run auto-migration
	if cfg.AutoMigrate && len(models) > 0 {
		log.Println("🔄 Running database auto-migration...")
		if err := db.AutoMigrate(models...); err != nil {
			return nil, fmt.Errorf("failed to auto-migrate: %w", err)
		}
		log.Println("✅ Database migration completed successfully")
	} else if !cfg.AutoMigrate {
		log.Println("ℹ️ Database auto-migration disabled by configuration")
	}

	if err := RunDatabaseMigrations(db); err != nil {
		log.Printf("⚠️ Failed to run database migrations: %v", err)
	} else {
		log.Println("🧭 Database migrations checked")
	}

	// Set global DB instance
	DB = db

	log.Printf("📊 Connected to PostgreSQL: %s@%s:%s/%s", cfg.User, cfg.Host, cfg.Port, cfg.DBName)
	return db, nil
}

// Close closes the database connection.
func Close() error {
	if DB == nil {
		return nil
	}
	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// GetDB returns the global database instance.
// Returns nil if database has not been initialized.
func GetDB() *gorm.DB {
	return DB
}

// getEnv returns the environment variable value or a default value.
func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func parseGORMLogLevel(raw string) logger.LogLevel {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "silent":
		return logger.Silent
	case "error":
		return logger.Error
	case "info":
		return logger.Info
	case "warn", "warning", "":
		return logger.Warn
	default:
		return logger.Warn
	}
}
