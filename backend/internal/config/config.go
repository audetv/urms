// backend/internal/config/config.go
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config представляет основную конфигурацию приложения
type Config struct {
	// Database configuration
	Database DatabaseConfig `yaml:"database"`

	// Email configuration
	Email EmailConfig `yaml:"email"`

	// Server configuration
	Server ServerConfig `yaml:"server"`

	// Logging configuration
	Logging LoggingConfig `yaml:"logging"`
}

// DatabaseConfig конфигурация базы данных
type DatabaseConfig struct {
	Provider string         `yaml:"provider"` // postgres, inmemory
	Postgres PostgresConfig `yaml:"postgres"`
}

// PostgresConfig конфигурация PostgreSQL
type PostgresConfig struct {
	DSN             string        `yaml:"dsn"`
	MaxOpenConns    int           `yaml:"max_open_conns"`
	MaxIdleConns    int           `yaml:"max_idle_conns"`
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime"`
}

// EmailConfig конфигурация email модуля
type EmailConfig struct {
	IMAP IMAPConfig `yaml:"imap"`
}

// IMAPConfig конфигурация IMAP
type IMAPConfig struct {
	Server       string        `yaml:"server"`
	Port         int           `yaml:"port"`
	Username     string        `yaml:"username"`
	Password     string        `yaml:"password"`
	Mailbox      string        `yaml:"mailbox"`
	SSL          bool          `yaml:"ssl"`
	PollInterval time.Duration `yaml:"poll_interval"`
	ReadOnly     bool          `yaml:"read_only"`

	// ✅ NEW: Timeout configuration from ADR-002
	ConnectTimeout     time.Duration `yaml:"connect_timeout"`
	LoginTimeout       time.Duration `yaml:"login_timeout"`
	FetchTimeout       time.Duration `yaml:"fetch_timeout"`
	OperationTimeout   time.Duration `yaml:"operation_timeout"`
	PageSize           int           `yaml:"page_size"`
	MaxMessagesPerPoll int           `yaml:"max_messages_per_poll"`
	MaxRetries         int           `yaml:"max_retries"`
	RetryDelay         time.Duration `yaml:"retry_delay"`
}

// ServerConfig конфигурация сервера
type ServerConfig struct {
	Port         int           `yaml:"port"`
	ReadTimeout  time.Duration `yaml:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout"`
}

// LoggingConfig конфигурация логирования
type LoggingConfig struct {
	Level  string `yaml:"level"`  // debug, info, warn, error
	Format string `yaml:"format"` // json, text
}

// LoadConfig загружает конфигурацию из environment variables
func LoadConfig() (*Config, error) {
	config := &Config{
		Database: DatabaseConfig{
			Provider: getEnv("URMS_DATABASE_PROVIDER", "inmemory"),
			Postgres: PostgresConfig{
				DSN:             getEnv("URMS_DATABASE_DSN", "postgres://urms:urms@localhost:5432/urms?sslmode=disable"),
				MaxOpenConns:    getEnvAsInt("URMS_DATABASE_MAX_OPEN_CONNS", 25),
				MaxIdleConns:    getEnvAsInt("URMS_DATABASE_MAX_IDLE_CONNS", 5),
				ConnMaxLifetime: getEnvAsDuration("URMS_DATABASE_CONN_MAX_LIFETIME", time.Hour),
			},
		},
		Email: EmailConfig{
			IMAP: IMAPConfig{
				Server:       getEnv("URMS_IMAP_SERVER", "outlook.office365.com"),
				Port:         getEnvAsInt("URMS_IMAP_PORT", 993),
				Username:     getEnv("URMS_IMAP_USERNAME", ""),
				Password:     getEnv("URMS_IMAP_PASSWORD", ""),
				Mailbox:      getEnv("URMS_IMAP_MAILBOX", "INBOX"),
				SSL:          getEnvAsBool("URMS_IMAP_SSL", true),
				PollInterval: getEnvAsDuration("URMS_IMAP_POLL_INTERVAL", 30*time.Second),
				ReadOnly:     getEnvAsBool("URMS_IMAP_READ_ONLY", true),

				// ✅ NEW: Timeout defaults from ADR-002
				ConnectTimeout:     getEnvAsDuration("URMS_IMAP_CONNECT_TIMEOUT", 30*time.Second),
				LoginTimeout:       getEnvAsDuration("URMS_IMAP_LOGIN_TIMEOUT", 15*time.Second),
				FetchTimeout:       getEnvAsDuration("URMS_IMAP_FETCH_TIMEOUT", 60*time.Second),
				OperationTimeout:   getEnvAsDuration("URMS_IMAP_OPERATION_TIMEOUT", 120*time.Second),
				PageSize:           getEnvAsInt("URMS_IMAP_PAGE_SIZE", 100),
				MaxMessagesPerPoll: getEnvAsInt("URMS_IMAP_MAX_MESSAGES_PER_POLL", 500),
				MaxRetries:         getEnvAsInt("URMS_IMAP_MAX_RETRIES", 3),
				RetryDelay:         getEnvAsDuration("URMS_IMAP_RETRY_DELAY", 10*time.Second),
			},
		},
		Server: ServerConfig{
			Port:         getEnvAsInt("URMS_SERVER_PORT", 8080),
			ReadTimeout:  getEnvAsDuration("URMS_SERVER_READ_TIMEOUT", 15*time.Second),
			WriteTimeout: getEnvAsDuration("URMS_SERVER_WRITE_TIMEOUT", 15*time.Second),
		},
		Logging: LoggingConfig{
			Level:  getEnv("URMS_LOGGING_LEVEL", "info"),
			Format: getEnv("URMS_LOGGING_FORMAT", "text"),
		},
	}

	// Валидация конфигурации
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return config, nil
}

// Validate проверяет валидность конфигурации
func (c *Config) Validate() error {
	if c.Database.Provider != "postgres" && c.Database.Provider != "inmemory" {
		return fmt.Errorf("invalid database provider: %s", c.Database.Provider)
	}

	if c.Email.IMAP.Username == "" || c.Email.IMAP.Password == "" {
		return fmt.Errorf("IMAP credentials are required")
	}

	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", c.Server.Port)
	}

	// ✅ NEW: Validate timeout configuration
	if c.Email.IMAP.ConnectTimeout <= 0 {
		return fmt.Errorf("IMAP connect timeout must be positive")
	}
	if c.Email.IMAP.FetchTimeout <= 0 {
		return fmt.Errorf("IMAP fetch timeout must be positive")
	}
	if c.Email.IMAP.PageSize <= 0 {
		return fmt.Errorf("IMAP page size must be positive")
	}
	if c.Email.IMAP.MaxMessagesPerPoll <= 0 {
		return fmt.Errorf("IMAP max messages per poll must be positive")
	}
	if c.Email.IMAP.MaxRetries < 0 {
		return fmt.Errorf("IMAP max retries cannot be negative")
	}

	return nil
}

// GetIMAPTimeoutConfig возвращает конфигурацию таймаутов для IMAP операций
func (c *Config) GetIMAPTimeoutConfig() IMAPTimeoutConfig {
	return IMAPTimeoutConfig{
		ConnectTimeout:   c.Email.IMAP.ConnectTimeout,
		LoginTimeout:     c.Email.IMAP.LoginTimeout,
		FetchTimeout:     c.Email.IMAP.FetchTimeout,
		OperationTimeout: c.Email.IMAP.OperationTimeout,
		PageSize:         c.Email.IMAP.PageSize,
		MaxMessages:      c.Email.IMAP.MaxMessagesPerPoll,
		MaxRetries:       c.Email.IMAP.MaxRetries,
		RetryDelay:       c.Email.IMAP.RetryDelay,
	}
}

// IMAPTimeoutConfig представляет конфигурацию таймаутов для IMAP операций
type IMAPTimeoutConfig struct {
	ConnectTimeout   time.Duration
	LoginTimeout     time.Duration
	FetchTimeout     time.Duration
	OperationTimeout time.Duration
	PageSize         int
	MaxMessages      int
	MaxRetries       int
	RetryDelay       time.Duration
}

// Helper functions для работы с environment variables
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}
