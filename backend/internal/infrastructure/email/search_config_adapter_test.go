// internal/infrastructure/email/search_config_adapter_test.go
package email

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockLoggerForAdapter мок логгера для тестирования адаптера
type MockLoggerForAdapter struct{}

func (m *MockLoggerForAdapter) Debug(ctx context.Context, msg string, fields ...interface{}) {}
func (m *MockLoggerForAdapter) Info(ctx context.Context, msg string, fields ...interface{})  {}
func (m *MockLoggerForAdapter) Warn(ctx context.Context, msg string, fields ...interface{})  {}
func (m *MockLoggerForAdapter) Error(ctx context.Context, msg string, fields ...interface{}) {}
func (m *MockLoggerForAdapter) WithContext(ctx context.Context) context.Context              { return ctx }

func TestSearchConfigAdapter_GetThreadSearchConfig_Success(t *testing.T) {
	ctx := context.Background()
	logger := new(MockLoggerForAdapter)

	config := &EmailSearchConfig{
		ThreadSearch: ThreadSearchConfig{
			DefaultDaysBack:     30,
			ExtendedDaysBack:    90,
			MaxDaysBack:         180,
			FetchTimeout:        60 * time.Second,
			IncludeSeenMessages: true,
			SubjectPrefixes:     []string{"Re:", "Fwd:"},
		},
	}

	adapter := NewSearchConfigAdapter(config, logger)

	result, err := adapter.GetThreadSearchConfig(ctx)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 30, result.DefaultDaysBack)
	assert.Equal(t, 90, result.ExtendedDaysBack)
	assert.Equal(t, 180, result.MaxDaysBack)
	assert.Equal(t, 60*time.Second, result.FetchTimeout)
	assert.True(t, result.IncludeSeenMessages)
	assert.Equal(t, []string{"Re:", "Fwd:"}, result.SubjectPrefixes)
}

func TestSearchConfigAdapter_GetThreadSearchConfig_DefaultSubjectPrefixes(t *testing.T) {
	ctx := context.Background()
	logger := new(MockLoggerForAdapter)

	config := &EmailSearchConfig{
		ThreadSearch: ThreadSearchConfig{
			DefaultDaysBack:     30,
			ExtendedDaysBack:    90,
			MaxDaysBack:         180,
			FetchTimeout:        60 * time.Second,
			IncludeSeenMessages: true,
			SubjectPrefixes:     nil, // Не заданы явно
		},
	}

	adapter := NewSearchConfigAdapter(config, logger)

	result, err := adapter.GetThreadSearchConfig(ctx)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.NotEmpty(t, result.SubjectPrefixes)
	// Должны использоваться значения по умолчанию
	assert.Contains(t, result.SubjectPrefixes, "Re:")
	assert.Contains(t, result.SubjectPrefixes, "Fwd:")
	assert.Contains(t, result.SubjectPrefixes, "Ответ:")
}

func TestSearchConfigAdapter_GetThreadSearchConfig_InvalidConfig(t *testing.T) {
	ctx := context.Background()
	logger := new(MockLoggerForAdapter)

	// Невалидная конфигурация
	config := &EmailSearchConfig{
		ThreadSearch: ThreadSearchConfig{
			DefaultDaysBack:     0, // Невалидное значение
			ExtendedDaysBack:    90,
			MaxDaysBack:         180,
			FetchTimeout:        60 * time.Second,
			IncludeSeenMessages: true,
			SubjectPrefixes:     []string{"Re:"},
		},
	}

	adapter := NewSearchConfigAdapter(config, logger)

	result, err := adapter.GetThreadSearchConfig(ctx)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "default_days_back must be positive")
}

func TestSearchConfigAdapter_GetProviderSpecificConfig_KnownProvider(t *testing.T) {
	ctx := context.Background()
	logger := new(MockLoggerForAdapter)

	config := &EmailSearchConfig{
		ThreadSearch: ThreadSearchConfig{
			DefaultDaysBack:     30,
			ExtendedDaysBack:    90,
			MaxDaysBack:         180,
			FetchTimeout:        60 * time.Second,
			IncludeSeenMessages: true,
			SubjectPrefixes:     []string{"Re:"},
		},
		ProviderConfig: map[string]ProviderSearchConfig{
			"gmail": {
				MaxDaysBack:    365,
				SearchTimeout:  120 * time.Second,
				SupportedFlags: []string{"X-GM-RAW"},
				Optimizations:  []string{"gmail_thread_id"},
			},
		},
	}

	adapter := NewSearchConfigAdapter(config, logger)

	result, err := adapter.GetProviderSpecificConfig(ctx, "gmail")

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "gmail", result.ProviderName)
	assert.Equal(t, 365, result.MaxDaysBack)
	assert.Equal(t, 120*time.Second, result.SearchTimeout)
	assert.Contains(t, result.SupportedFlags, "X-GM-RAW")
	assert.Contains(t, result.Optimizations, "gmail_thread_id")
}

func TestSearchConfigAdapter_GetProviderSpecificConfig_UnknownProvider(t *testing.T) {
	ctx := context.Background()
	logger := new(MockLoggerForAdapter)

	config := &EmailSearchConfig{
		ThreadSearch: ThreadSearchConfig{
			DefaultDaysBack:     30,
			ExtendedDaysBack:    90,
			MaxDaysBack:         180,
			FetchTimeout:        60 * time.Second,
			IncludeSeenMessages: true,
			SubjectPrefixes:     []string{"Re:"},
		},
		ProviderConfig: map[string]ProviderSearchConfig{
			"gmail": {
				MaxDaysBack:   365,
				SearchTimeout: 120 * time.Second,
			},
		},
	}

	adapter := NewSearchConfigAdapter(config, logger)

	// Запрашиваем неизвестного провайдера
	result, err := adapter.GetProviderSpecificConfig(ctx, "unknown")

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "generic", result.ProviderName)
	// Должен использовать значения из thread конфигурации
	assert.Equal(t, 180, result.MaxDaysBack)              // Из ThreadSearch.MaxDaysBack
	assert.Equal(t, 60*time.Second, result.SearchTimeout) // Из ThreadSearch.FetchTimeout
	assert.Contains(t, result.Optimizations, "standard_search")
}

func TestSearchConfigAdapter_GetProviderSpecificConfig_ProviderWithDefaults(t *testing.T) {
	ctx := context.Background()
	logger := new(MockLoggerForAdapter)

	config := &EmailSearchConfig{
		ThreadSearch: ThreadSearchConfig{
			DefaultDaysBack:     30,
			ExtendedDaysBack:    90,
			MaxDaysBack:         180,
			FetchTimeout:        60 * time.Second,
			IncludeSeenMessages: true,
			SubjectPrefixes:     []string{"Re:"},
		},
		ProviderConfig: map[string]ProviderSearchConfig{
			"testprovider": {
				MaxDaysBack:   0, // Не задано
				SearchTimeout: 0, // Не задано
				Optimizations: []string{"test_optimization"},
			},
		},
	}

	adapter := NewSearchConfigAdapter(config, logger)

	result, err := adapter.GetProviderSpecificConfig(ctx, "testprovider")

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "testprovider", result.ProviderName)
	// Должен использовать значения по умолчанию из thread конфигурации
	assert.Equal(t, 180, result.MaxDaysBack)              // Из ThreadSearch.MaxDaysBack
	assert.Equal(t, 60*time.Second, result.SearchTimeout) // Из ThreadSearch.FetchTimeout
	assert.Contains(t, result.Optimizations, "test_optimization")
}

func TestSearchConfigAdapter_ValidateConfig_Success(t *testing.T) {
	ctx := context.Background()
	logger := new(MockLoggerForAdapter)

	config := &EmailSearchConfig{
		ThreadSearch: ThreadSearchConfig{
			DefaultDaysBack:     30,
			ExtendedDaysBack:    90,
			MaxDaysBack:         180,
			FetchTimeout:        60 * time.Second,
			IncludeSeenMessages: true,
			SubjectPrefixes:     []string{"Re:"},
		},
		ProviderConfig: map[string]ProviderSearchConfig{
			"gmail": {
				MaxDaysBack:   365,
				SearchTimeout: 120 * time.Second,
			},
		},
	}

	adapter := NewSearchConfigAdapter(config, logger)

	err := adapter.ValidateConfig(ctx)

	require.NoError(t, err)
}

func TestSearchConfigAdapter_ValidateConfig_InvalidThreadConfig(t *testing.T) {
	ctx := context.Background()
	logger := new(MockLoggerForAdapter)

	// Невалидная thread конфигурация
	config := &EmailSearchConfig{
		ThreadSearch: ThreadSearchConfig{
			DefaultDaysBack:     0, // Невалидное значение
			ExtendedDaysBack:    90,
			MaxDaysBack:         180,
			FetchTimeout:        60 * time.Second,
			IncludeSeenMessages: true,
			SubjectPrefixes:     []string{"Re:"},
		},
	}

	adapter := NewSearchConfigAdapter(config, logger)

	err := adapter.ValidateConfig(ctx)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "thread search configuration validation failed")
}
