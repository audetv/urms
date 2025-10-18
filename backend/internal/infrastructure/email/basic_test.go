package email

import (
	"testing"
	"time"

	imapclient "github.com/audetv/urms/internal/infrastructure/email/imap"
	"github.com/audetv/urms/internal/infrastructure/logging"
	"github.com/stretchr/testify/assert"
)

// TestBasicCompilation проверяет, что пакет компилируется
func TestBasicCompilation(t *testing.T) {
	assert.True(t, true, "Basic test should pass")
}

// TestIMAPAdapterCreation проверяет создание IMAP адаптера
func TestIMAPAdapterCreation(t *testing.T) {
	config := &imapclient.Config{
		Server:   "test.server.com",
		Port:     993,
		Username: "test",
		Password: "test",
		SSL:      true,
	}

	// ✅ ИСПРАВЛЕНО: Используем legacy конструктор для обратной совместимости
	adapter := NewIMAPAdapterLegacy(config)
	assert.NotNil(t, adapter, "IMAPAdapter should be created")
}

// TestIMAPAdapterCreationWithTimeouts проверяет создание IMAP адаптера с таймаутами
func TestIMAPAdapterCreationWithTimeouts(t *testing.T) {
	// Создаем тестовый logger
	logger := logging.NewTestLogger()

	config := &imapclient.Config{
		Server:   "test.server.com",
		Port:     993,
		Username: "test",
		Password: "test",
		SSL:      true,
	}

	timeoutConfig := TimeoutConfig{
		ConnectTimeout:   30 * time.Second,
		LoginTimeout:     15 * time.Second,
		FetchTimeout:     60 * time.Second,
		OperationTimeout: 120 * time.Second,
		PageSize:         100,
		MaxMessages:      500,
		MaxRetries:       3,
		RetryDelay:       10 * time.Second,
	}

	// ✅ ИСПРАВЛЕНО: Используем новый конструктор с таймаутами
	adapter := NewIMAPAdapter(config, timeoutConfig, logger)
	assert.NotNil(t, adapter, "IMAPAdapter with timeouts should be created")
}
