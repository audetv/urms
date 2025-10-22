// internal/infrastructure/email/search_config_adapter.go
package email

import (
	"context"
	"fmt"
	"time"

	"github.com/audetv/urms/internal/core/ports"
)

// SearchConfigAdapter инфраструктурная реализация EmailSearchConfigProvider
type SearchConfigAdapter struct {
	config *EmailSearchConfig
	logger ports.Logger
}

// EmailSearchConfig структура для хранения конфигурации поиска
type EmailSearchConfig struct {
	ThreadSearch   ThreadSearchConfig              `yaml:"thread_search"`
	ProviderConfig map[string]ProviderSearchConfig `yaml:"provider_config"`
}

// ThreadSearchConfig конфигурация thread-aware поиска
type ThreadSearchConfig struct {
	DefaultDaysBack     int           `yaml:"default_days_back"`
	ExtendedDaysBack    int           `yaml:"extended_days_back"`
	MaxDaysBack         int           `yaml:"max_days_back"`
	FetchTimeout        time.Duration `yaml:"fetch_timeout"`
	IncludeSeenMessages bool          `yaml:"include_seen_messages"`
	SubjectPrefixes     []string      `yaml:"subject_prefixes"`
}

// ProviderSearchConfig провайдер-специфичная конфигурация
type ProviderSearchConfig struct {
	MaxDaysBack    int           `yaml:"max_days_back"`
	SearchTimeout  time.Duration `yaml:"search_timeout"`
	SupportedFlags []string      `yaml:"supported_flags"`
	Optimizations  []string      `yaml:"optimizations"`
}

// NewSearchConfigAdapter создает новый адаптер конфигурации поиска
func NewSearchConfigAdapter(config *EmailSearchConfig, logger ports.Logger) *SearchConfigAdapter {
	return &SearchConfigAdapter{
		config: config,
		logger: logger,
	}
}

// GetThreadSearchConfig возвращает конфигурацию для thread-aware поиска
func (a *SearchConfigAdapter) GetThreadSearchConfig(ctx context.Context) (*ports.ThreadSearchConfig, error) {
	// ✅ УБИРАЕМ детальное логирование получения конфигурации - вызывается слишком часто

	if a.config.ThreadSearch.DefaultDaysBack <= 0 {
		return nil, fmt.Errorf("invalid thread search configuration: default_days_back must be positive")
	}

	config := &ports.ThreadSearchConfig{
		DefaultDaysBack:     a.config.ThreadSearch.DefaultDaysBack,
		ExtendedDaysBack:    a.config.ThreadSearch.ExtendedDaysBack,
		MaxDaysBack:         a.config.ThreadSearch.MaxDaysBack,
		FetchTimeout:        a.config.ThreadSearch.FetchTimeout,
		IncludeSeenMessages: a.config.ThreadSearch.IncludeSeenMessages,
		SubjectPrefixes:     a.config.ThreadSearch.SubjectPrefixes,
	}

	// ✅ ОПТИМИЗИРУЕМ: только факт использования дефолтов, без списка
	if len(config.SubjectPrefixes) == 0 {
		config.SubjectPrefixes = []string{"Re:", "RE:", "Fwd:", "FW:", "Ответ:"}
		a.logger.Debug(ctx, "Using default subject prefixes",
			"prefixes_count", len(config.SubjectPrefixes)) // ✅ ТОЛЬКО КОЛИЧЕСТВО
	}

	// ✅ ПЕРЕВОДИМ В DEBUG - конфигурация не меняется часто
	a.logger.Debug(ctx, "Thread search configuration loaded",
		"default_days", config.DefaultDaysBack,
		"max_days", config.MaxDaysBack)
	// ✅ УБИРАЕМ: extended_days, fetch_timeout, include_seen_messages - избыточно

	return config, nil
}

// GetProviderSpecificConfig возвращает провайдер-специфичную конфигурацию
func (a *SearchConfigAdapter) GetProviderSpecificConfig(ctx context.Context, provider string) (*ports.ProviderSearchConfig, error) {
	a.logger.Debug(ctx, "Getting provider-specific configuration",
		"provider", provider)

	providerConfig, exists := a.config.ProviderConfig[provider]
	if !exists {
		a.logger.Warn(ctx, "Provider configuration not found, using generic fallback",
			"provider", provider)

		// Возвращаем generic конфигурацию как fallback
		return a.getGenericProviderConfig(ctx)
	}

	config := &ports.ProviderSearchConfig{
		ProviderName:   provider,
		MaxDaysBack:    providerConfig.MaxDaysBack,
		SearchTimeout:  providerConfig.SearchTimeout,
		SupportedFlags: providerConfig.SupportedFlags,
		Optimizations:  providerConfig.Optimizations,
	}

	// Устанавливаем значения по умолчанию если не заданы
	if config.MaxDaysBack <= 0 {
		config.MaxDaysBack = a.config.ThreadSearch.MaxDaysBack
		a.logger.Debug(ctx, "Using thread config max_days for provider",
			"provider", provider, "max_days", config.MaxDaysBack)
	}

	if config.SearchTimeout <= 0 {
		config.SearchTimeout = a.config.ThreadSearch.FetchTimeout
		a.logger.Debug(ctx, "Using thread config timeout for provider",
			"provider", provider, "timeout", config.SearchTimeout)
	}

	a.logger.Debug(ctx, "Provider-specific configuration loaded",
		"provider", config.ProviderName,
		"max_days", config.MaxDaysBack)

	return config, nil
}

// getGenericProviderConfig возвращает generic конфигурацию для неизвестных провайдеров
func (a *SearchConfigAdapter) getGenericProviderConfig(ctx context.Context) (*ports.ProviderSearchConfig, error) {
	config := &ports.ProviderSearchConfig{
		ProviderName:  "generic",
		MaxDaysBack:   a.config.ThreadSearch.MaxDaysBack,
		SearchTimeout: a.config.ThreadSearch.FetchTimeout,
		Optimizations: []string{"standard_search"},
	}

	a.logger.Debug(ctx, "Using generic provider configuration",
		"max_days", config.MaxDaysBack,
		"search_timeout", config.SearchTimeout)

	return config, nil
}

// ValidateConfig проверяет валидность всей конфигурации
func (a *SearchConfigAdapter) ValidateConfig(ctx context.Context) error {
	a.logger.Debug(ctx, "Validating search configuration")

	// Проверяем thread search конфигурацию
	threadConfig, err := a.GetThreadSearchConfig(ctx)
	if err != nil {
		return fmt.Errorf("thread search configuration validation failed: %w", err)
	}

	// Проверяем базовые правила
	if threadConfig.DefaultDaysBack <= 0 {
		return fmt.Errorf("default_days_back must be positive")
	}
	if threadConfig.ExtendedDaysBack < threadConfig.DefaultDaysBack {
		return fmt.Errorf("extended_days_back must be >= default_days_back")
	}
	if threadConfig.MaxDaysBack < threadConfig.ExtendedDaysBack {
		return fmt.Errorf("max_days_back must be >= extended_days_back")
	}
	if threadConfig.FetchTimeout <= 0 {
		return fmt.Errorf("fetch_timeout must be positive")
	}

	// Проверяем провайдер-специфичные конфигурации
	for provider := range a.config.ProviderConfig {
		_, err := a.GetProviderSpecificConfig(ctx, provider)
		if err != nil {
			a.logger.Warn(ctx, "Provider configuration validation warning",
				"provider", provider, "error", err.Error())
			// Продолжаем валидацию для других провайдеров
		}
	}

	a.logger.Info(ctx, "Search configuration validation completed successfully")
	return nil
}
