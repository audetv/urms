// internal/infrastructure/email/phase3c_validation_test.go
package email

import (
	"context"
	"testing"
	"time"

	"github.com/audetv/urms/internal/core/ports"
	imapclient "github.com/audetv/urms/internal/infrastructure/email/imap"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPhase3C_EnhancedSearchConfiguration проверяет что конфигурация применяется
func TestPhase3C_EnhancedSearchConfiguration(t *testing.T) {
	ctx := context.Background()
	logger := &MockLoggerForAdapter{}

	// Создаем тестовую конфигурацию
	config := &EmailSearchConfig{
		ThreadSearch: ThreadSearchConfig{
			DefaultDaysBack:     180,
			ExtendedDaysBack:    365,
			MaxDaysBack:         730,
			FetchTimeout:        120 * time.Second,
			IncludeSeenMessages: true,
			SubjectPrefixes:     []string{"Re:", "Fwd:"},
		},
	}

	adapter := NewSearchConfigAdapter(config, logger)

	// Проверяем что конфигурация загружается
	threadConfig, err := adapter.GetThreadSearchConfig(ctx)
	require.NoError(t, err)
	require.NotNil(t, threadConfig)

	// ✅ ПРОВЕРЯЕМ РАСШИРЕННЫЕ ПАРАМЕТРЫ
	assert.Equal(t, 180, threadConfig.DefaultDaysBack,
		"Default days back should be 180 for extended search")
	assert.Equal(t, 365, threadConfig.ExtendedDaysBack,
		"Extended days back should be 365 for complete threading")
	assert.Equal(t, 730, threadConfig.MaxDaysBack,
		"Max days back should be 730 for maximum coverage")
	assert.True(t, threadConfig.IncludeSeenMessages,
		"Should include seen messages for complete thread detection")

	t.Logf("✅ Phase 3C Configuration: %d/%d/%d days search range",
		threadConfig.DefaultDaysBack,
		threadConfig.ExtendedDaysBack,
		threadConfig.MaxDaysBack)
}

// TestPhase3C_ThreadSearchCriteria проверяет enhanced критерии поиска
func TestPhase3C_ThreadSearchCriteria(t *testing.T) {
	// ctx := context.Background()
	logger := &MockLoggerForAdapter{}

	// Создаем IMAP адаптер с конфигурацией
	config := &EmailSearchConfig{
		ThreadSearch: ThreadSearchConfig{
			DefaultDaysBack:     180,
			ExtendedDaysBack:    365,
			MaxDaysBack:         730,
			FetchTimeout:        120 * time.Second,
			IncludeSeenMessages: true,
			SubjectPrefixes:     []string{"Re:", "Fwd:", "Ответ:"},
		},
	}

	searchConfig := NewSearchConfigAdapter(config, logger)

	// Создаем mock IMAP конфиг
	imapConfig := &imapclient.Config{
		Server:   "test",
		Port:     993,
		Username: "test",
		Password: "test",
	}

	timeoutConfig := TimeoutConfig{
		ConnectTimeout:   30 * time.Second,
		LoginTimeout:     15 * time.Second,
		FetchTimeout:     60 * time.Second,
		OperationTimeout: 120 * time.Second,
		PageSize:         100,
		MaxMessages:      500,
	}

	adapter := NewIMAPAdapterWithTimeoutsAndConfig(imapConfig, timeoutConfig, searchConfig, logger)

	// Тестовые данные для цепочки из 5 писем
	threadData := ports.ThreadSearchCriteria{
		MessageID:  "<msg5@test>",
		InReplyTo:  "<msg4@test>",
		References: []string{"<msg1@test>", "<msg2@test>", "<msg3@test>", "<msg4@test>"},
		Subject:    "Re: Important Discussion",
		Mailbox:    "INBOX",
	}

	// Проверяем что enhanced критерии создаются
	criteria, err := adapter.createEnhancedThreadSearchCriteria(threadData)
	require.NoError(t, err)
	require.NotNil(t, criteria)

	// ✅ ПРОВЕРЯЕМ ЧТО ВРЕМЕННОЙ ДИАПАЗОН РАСШИРЕН
	expectedSince := time.Now().Add(-365 * 24 * time.Hour)
	assert.True(t, criteria.Since.Before(expectedSince) || criteria.Since.Equal(expectedSince),
		"Search should cover at least 365 days for complete threading")

	t.Logf("✅ Phase 3C Search Criteria: since %s (should cover 365+ days)",
		criteria.Since.Format("2006-01-02"))
}
