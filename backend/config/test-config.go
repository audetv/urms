// backend/config/test-config.go
package main

import (
	"time"

	"github.com/audetv/urms/internal/email/imapclient"
)

// GetTestConfig возвращает тестовую конфигурацию
func GetTestConfig() *imapclient.Config {
	return &imapclient.Config{
		Server:   "outlook.office365.com",
		Port:     993,
		Username: "support@yourcompany.com",
		Password: "your_password",
		Mailbox:  "INBOX",
		SSL:      true,
		Interval: 30 * time.Second,
		Timeout:  30 * time.Second,
		ReadOnly: true,
	}
}

// GetSafeTestConfig возвращает конфиг с безопасными значениями по умолчанию
func GetSafeTestConfig() *imapclient.Config {
	config := GetTestConfig()
	// Переопределяем чувствительные данные пустыми значениями
	config.Username = ""
	config.Password = ""
	return config
}
