package email

import (
	"testing"

	imapclient "github.com/audetv/urms/internal/infrastructure/email/imap"
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

	adapter := NewIMAPAdapter(config)
	assert.NotNil(t, adapter, "IMAPAdapter should be created")
}
