// backend/internal/infrastructure/email/imap/config.go
package imapclient

import (
	"fmt"
	"time"
)

type Config struct {
	Server   string        `yaml:"server" json:"server"`
	Port     int           `yaml:"port" json:"port"`
	Username string        `yaml:"username" json:"username"`
	Password string        `yaml:"password" json:"password"`
	Mailbox  string        `yaml:"mailbox" json:"mailbox"`
	SSL      bool          `yaml:"ssl" json:"ssl"`
	Interval time.Duration `yaml:"interval" json:"interval"` // e.g., "30s", "1m"

	// Advanced options
	ReadOnly bool          `yaml:"read_only" json:"read_only"` // Не помечать письма как прочитанные
	Timeout  time.Duration `yaml:"timeout" json:"timeout"`     // Таймаут операций

	// ✅ NEW: Extended timeout configuration from ADR-002
	ConnectTimeout   time.Duration `yaml:"connect_timeout" json:"connect_timeout"`
	LoginTimeout     time.Duration `yaml:"login_timeout" json:"login_timeout"`
	FetchTimeout     time.Duration `yaml:"fetch_timeout" json:"fetch_timeout"`
	OperationTimeout time.Duration `yaml:"operation_timeout" json:"operation_timeout"`
	PageSize         int           `yaml:"page_size" json:"page_size"`
}

// DefaultConfig возвращает конфигурацию по умолчанию
func DefaultConfig() *Config {
	return &Config{
		Server:   "outlook.office365.com",
		Port:     993,
		Mailbox:  "INBOX",
		SSL:      true,
		Interval: 30 * time.Second,
		ReadOnly: true,
		Timeout:  30 * time.Second,

		// ✅ NEW: Default timeout values
		ConnectTimeout:   30 * time.Second,
		LoginTimeout:     15 * time.Second,
		FetchTimeout:     60 * time.Second,
		OperationTimeout: 120 * time.Second,
		PageSize:         100,
	}
}

// Validate проверяет корректность конфигурации
func (c *Config) Validate() error {
	if c.Server == "" {
		return fmt.Errorf("IMAP server address is required")
	}
	if c.Port <= 0 || c.Port > 65535 {
		return fmt.Errorf("invalid IMAP port: %d", c.Port)
	}
	if c.Username == "" {
		return fmt.Errorf("IMAP username is required")
	}
	if c.Password == "" {
		return fmt.Errorf("IMAP password is required")
	}
	if c.Interval < 10*time.Second {
		return fmt.Errorf("polling interval too short: %v", c.Interval)
	}

	// ✅ NEW: Validate timeout configuration
	if c.ConnectTimeout <= 0 {
		return fmt.Errorf("connect timeout must be positive")
	}
	if c.FetchTimeout <= 0 {
		return fmt.Errorf("fetch timeout must be positive")
	}
	if c.PageSize <= 0 {
		return fmt.Errorf("page size must be positive")
	}

	return nil
}

// Addr возвращает адрес сервера в формате host:port
func (c *Config) Addr() string {
	return fmt.Sprintf("%s:%d", c.Server, c.Port)
}
