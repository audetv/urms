// backend/internal/email/imapclient/config.go
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
	return nil
}

// Addr возвращает адрес сервера в формате host:port
func (c *Config) Addr() string {
	return fmt.Sprintf("%s:%d", c.Server, c.Port)
}
