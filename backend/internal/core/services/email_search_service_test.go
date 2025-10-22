// internal/core/services/email_search_service_test.go
package services

import (
	"context"
	"testing"
	"time"

	"github.com/audetv/urms/internal/core/ports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockEmailSearchConfigProvider мок для тестирования
type MockEmailSearchConfigProvider struct {
	mock.Mock
}

func (m *MockEmailSearchConfigProvider) GetThreadSearchConfig(ctx context.Context) (*ports.ThreadSearchConfig, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ports.ThreadSearchConfig), args.Error(1)
}

func (m *MockEmailSearchConfigProvider) GetProviderSpecificConfig(ctx context.Context, provider string) (*ports.ProviderSearchConfig, error) {
	args := m.Called(ctx, provider)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ports.ProviderSearchConfig), args.Error(1)
}

func (m *MockEmailSearchConfigProvider) ValidateConfig(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// SimpleMockLogger простой мок логгера без сложных проверок
type SimpleMockLogger struct{}

func (m *SimpleMockLogger) Debug(ctx context.Context, msg string, fields ...interface{}) {}
func (m *SimpleMockLogger) Info(ctx context.Context, msg string, fields ...interface{})  {}
func (m *SimpleMockLogger) Warn(ctx context.Context, msg string, fields ...interface{})  {}
func (m *SimpleMockLogger) Error(ctx context.Context, msg string, fields ...interface{}) {}
func (m *SimpleMockLogger) WithContext(ctx context.Context) context.Context              { return ctx }

func TestEmailSearchService_GetThreadSearchConfig_Success(t *testing.T) {
	ctx := context.Background()

	// Создаем моки
	mockConfigProvider := new(MockEmailSearchConfigProvider)
	mockLogger := new(SimpleMockLogger) // Используем простой мок

	// Ожидаемые вызовы
	expectedConfig := &ports.ThreadSearchConfig{
		DefaultDaysBack:     30,
		ExtendedDaysBack:    90,
		MaxDaysBack:         180,
		FetchTimeout:        60 * time.Second,
		IncludeSeenMessages: true,
		SubjectPrefixes:     []string{"Re:", "Fwd:"},
	}

	mockConfigProvider.On("GetThreadSearchConfig", ctx).Return(expectedConfig, nil)

	// Создаем сервис
	service := NewEmailSearchService(mockConfigProvider, mockLogger)

	// Вызываем метод
	config, err := service.GetThreadSearchConfig(ctx)

	// Проверяем результаты
	require.NoError(t, err)
	require.NotNil(t, config)
	assert.Equal(t, 30, config.DefaultDaysBack())
	assert.Equal(t, 90, config.ExtendedDaysBack())
	assert.Equal(t, 180, config.MaxDaysBack())
	assert.Equal(t, 60*time.Second, config.FetchTimeout())
	assert.True(t, config.IncludeSeenMessages())

	// Проверяем что мок конфигурации был вызван
	mockConfigProvider.AssertExpectations(t)
}

func TestEmailSearchService_GetThreadSearchConfig_ProviderError(t *testing.T) {
	ctx := context.Background()

	mockConfigProvider := new(MockEmailSearchConfigProvider)
	mockLogger := new(SimpleMockLogger)

	// Настраиваем мок чтобы возвращал ошибку
	mockConfigProvider.On("GetThreadSearchConfig", ctx).Return((*ports.ThreadSearchConfig)(nil), assert.AnError)

	service := NewEmailSearchService(mockConfigProvider, mockLogger)

	config, err := service.GetThreadSearchConfig(ctx)

	require.Error(t, err)
	assert.Nil(t, config)
	assert.Contains(t, err.Error(), "failed to get search configuration")

	mockConfigProvider.AssertExpectations(t)
}

func TestEmailSearchService_GetThreadSearchConfig_InvalidConfig(t *testing.T) {
	ctx := context.Background()

	mockConfigProvider := new(MockEmailSearchConfigProvider)
	mockLogger := new(SimpleMockLogger)

	// Невалидная конфигурация
	invalidConfig := &ports.ThreadSearchConfig{
		DefaultDaysBack:     0, // Невалидное значение
		ExtendedDaysBack:    90,
		MaxDaysBack:         180,
		FetchTimeout:        60 * time.Second,
		IncludeSeenMessages: true,
		SubjectPrefixes:     []string{"Re:"},
	}

	mockConfigProvider.On("GetThreadSearchConfig", ctx).Return(invalidConfig, nil)

	service := NewEmailSearchService(mockConfigProvider, mockLogger)

	config, err := service.GetThreadSearchConfig(ctx)

	require.Error(t, err)
	assert.Nil(t, config)
	assert.Contains(t, err.Error(), "invalid search configuration")

	mockConfigProvider.AssertExpectations(t)
}

func TestEmailSearchService_GetSearchSince(t *testing.T) {
	ctx := context.Background()

	mockConfigProvider := new(MockEmailSearchConfigProvider)
	mockLogger := new(SimpleMockLogger)

	threadConfig := &ports.ThreadSearchConfig{
		DefaultDaysBack:     30,
		ExtendedDaysBack:    90,
		MaxDaysBack:         180,
		FetchTimeout:        60 * time.Second,
		IncludeSeenMessages: true,
		SubjectPrefixes:     []string{"Re:"},
	}

	providerConfig := &ports.ProviderSearchConfig{
		ProviderName:  "gmail",
		MaxDaysBack:   365,
		SearchTimeout: 120 * time.Second,
	}

	mockConfigProvider.On("GetThreadSearchConfig", ctx).Return(threadConfig, nil)
	mockConfigProvider.On("GetProviderSpecificConfig", ctx, "gmail").Return(providerConfig, nil)

	service := NewEmailSearchService(mockConfigProvider, mockLogger)

	// Тестируем разные типы поиска
	tests := []struct {
		name       string
		searchType string
		provider   string
	}{
		{"standard search", "standard", "gmail"},
		{"extended search", "extended", "gmail"},
		{"maximum search", "maximum", "gmail"},
		{"unknown search type", "unknown", "gmail"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			since, err := service.GetSearchSince(ctx, tt.searchType, tt.provider)

			require.NoError(t, err)
			assert.False(t, since.IsZero())

			// Проверяем что дата в прошлом
			assert.True(t, since.Before(time.Now()))
		})
	}

	mockConfigProvider.AssertExpectations(t)
}

func TestEmailSearchService_GenerateSearchSubjectVariants(t *testing.T) {
	ctx := context.Background()

	mockConfigProvider := new(MockEmailSearchConfigProvider)
	mockLogger := new(SimpleMockLogger)

	threadConfig := &ports.ThreadSearchConfig{
		DefaultDaysBack:     30,
		ExtendedDaysBack:    90,
		MaxDaysBack:         180,
		FetchTimeout:        60 * time.Second,
		IncludeSeenMessages: true,
		SubjectPrefixes:     []string{"Re:", "Fwd:"},
	}

	mockConfigProvider.On("GetThreadSearchConfig", ctx).Return(threadConfig, nil)

	service := NewEmailSearchService(mockConfigProvider, mockLogger)

	baseSubject := "Test Subject"
	variants, err := service.GenerateSearchSubjectVariants(ctx, baseSubject)

	require.NoError(t, err)
	require.NotEmpty(t, variants)

	expected := []string{
		"Test Subject",
		"Re: Test Subject",
		"Fwd: Test Subject",
	}

	assert.Equal(t, expected, variants)

	mockConfigProvider.AssertExpectations(t)
}

func TestEmailSearchService_SimpleCreation(t *testing.T) {
	mockConfigProvider := new(MockEmailSearchConfigProvider)
	mockLogger := new(SimpleMockLogger)

	service := NewEmailSearchService(mockConfigProvider, mockLogger)

	assert.NotNil(t, service)
}

// Тестируем провайдер-специфичную конфигурацию
func TestEmailSearchService_GetProviderSearchConfig(t *testing.T) {
	ctx := context.Background()

	mockConfigProvider := new(MockEmailSearchConfigProvider)
	mockLogger := new(SimpleMockLogger)

	providerConfig := &ports.ProviderSearchConfig{
		ProviderName:  "gmail",
		MaxDaysBack:   365,
		SearchTimeout: 120 * time.Second,
		Optimizations: []string{"gmail_thread_id"},
	}

	mockConfigProvider.On("GetProviderSpecificConfig", ctx, "gmail").Return(providerConfig, nil)

	service := NewEmailSearchService(mockConfigProvider, mockLogger)

	config, err := service.GetProviderSearchConfig(ctx, "gmail")

	require.NoError(t, err)
	require.NotNil(t, config)
	assert.Equal(t, "gmail", config.ProviderName)
	assert.Equal(t, 365, config.MaxDaysBack)
	assert.Equal(t, 120*time.Second, config.SearchTimeout)
	assert.Contains(t, config.Optimizations, "gmail_thread_id")

	mockConfigProvider.AssertExpectations(t)
}

// Тестируем валидацию конфигурации
func TestEmailSearchService_ValidateSearchConfig(t *testing.T) {
	ctx := context.Background()

	mockConfigProvider := new(MockEmailSearchConfigProvider)
	mockLogger := new(SimpleMockLogger)

	validConfig := &ports.ThreadSearchConfig{
		DefaultDaysBack:     30,
		ExtendedDaysBack:    90,
		MaxDaysBack:         180,
		FetchTimeout:        60 * time.Second,
		IncludeSeenMessages: true,
		SubjectPrefixes:     []string{"Re:"},
	}

	mockConfigProvider.On("GetThreadSearchConfig", ctx).Return(validConfig, nil)
	// Для провайдеров может быть ошибка, но это не должно ломать валидацию
	mockConfigProvider.On("GetProviderSpecificConfig", ctx, mock.Anything).Return((*ports.ProviderSearchConfig)(nil), assert.AnError)

	service := NewEmailSearchService(mockConfigProvider, mockLogger)

	err := service.ValidateSearchConfig(ctx)

	// Валидация должна пройти успешно, даже если некоторые провайдеры не настроены
	require.NoError(t, err)

	mockConfigProvider.AssertExpectations(t)
}
