// internal/core/services/email_search_service.go
package services

import (
	"context"
	"fmt"
	"time"

	"github.com/audetv/urms/internal/core/domain"
	"github.com/audetv/urms/internal/core/ports"
)

// EmailSearchService сервис для управления конфигурацией email поиска
type EmailSearchService struct {
	configProvider ports.EmailSearchConfigProvider
	logger         ports.Logger
}

// NewEmailSearchService создает новый экземпляр сервиса
func NewEmailSearchService(
	configProvider ports.EmailSearchConfigProvider,
	logger ports.Logger,
) *EmailSearchService {
	return &EmailSearchService{
		configProvider: configProvider,
		logger:         logger,
	}
}

// GetThreadSearchConfig получает и валидирует конфигурацию thread-aware поиска
func (s *EmailSearchService) GetThreadSearchConfig(ctx context.Context) (*domain.EmailSearchConfig, error) {
	// Получаем конфигурацию через порт
	config, err := s.configProvider.GetThreadSearchConfig(ctx)
	if err != nil {
		s.logger.Error(ctx, "Failed to get thread search configuration from provider",
			"error", err.Error())
		return nil, fmt.Errorf("failed to get search configuration: %w", err)
	}

	// Конвертируем в доменную сущность с валидацией
	domainConfig, err := domain.NewEmailSearchConfig(
		config.DefaultDaysBack,
		config.ExtendedDaysBack,
		config.MaxDaysBack,
		config.FetchTimeout,
		config.IncludeSeenMessages,
		config.SubjectPrefixes,
	)
	if err != nil {
		s.logger.Error(ctx, "Invalid search configuration received from provider",
			"default_days", config.DefaultDaysBack,
			"extended_days", config.ExtendedDaysBack,
			"max_days", config.MaxDaysBack,
			"error", err.Error())
		return nil, fmt.Errorf("invalid search configuration: %w", err)
	}

	s.logger.Info(ctx, "Thread search configuration loaded successfully",
		"default_days", domainConfig.DefaultDaysBack(),
		"extended_days", domainConfig.ExtendedDaysBack(),
		"max_days", domainConfig.MaxDaysBack(),
		"fetch_timeout", domainConfig.FetchTimeout(),
		"include_seen_messages", domainConfig.IncludeSeenMessages())

	return domainConfig, nil
}

// GetProviderSearchConfig получает провайдер-специфичную конфигурацию
func (s *EmailSearchService) GetProviderSearchConfig(ctx context.Context, provider string) (*ports.ProviderSearchConfig, error) {
	s.logger.Debug(ctx, "Getting provider-specific search configuration",
		"provider", provider)

	config, err := s.configProvider.GetProviderSpecificConfig(ctx, provider)
	if err != nil {
		s.logger.Error(ctx, "Failed to get provider-specific configuration",
			"provider", provider,
			"error", err.Error())
		return nil, fmt.Errorf("failed to get provider configuration for %s: %w", provider, err)
	}

	s.logger.Debug(ctx, "Provider-specific configuration loaded",
		"provider", config.ProviderName,
		"max_days", config.MaxDaysBack,
		"search_timeout", config.SearchTimeout,
		"optimizations_count", len(config.Optimizations))

	return config, nil
}

// ValidateSearchConfig проверяет валидность всей конфигурации поиска
func (s *EmailSearchService) ValidateSearchConfig(ctx context.Context) error {
	s.logger.Debug(ctx, "Validating search configuration")

	// Проверяем thread search конфигурацию
	_, err := s.GetThreadSearchConfig(ctx)
	if err != nil {
		return fmt.Errorf("thread search configuration validation failed: %w", err)
	}

	// Проверяем провайдер-специфичные конфигурации
	providers := []string{"gmail", "yandex", "outlook", "generic"}
	for _, provider := range providers {
		_, err := s.GetProviderSearchConfig(ctx, provider)
		if err != nil {
			s.logger.Warn(ctx, "Provider configuration validation warning",
				"provider", provider,
				"error", err.Error())
			// Продолжаем валидацию для других провайдеров
		}
	}

	s.logger.Info(ctx, "Search configuration validation completed successfully")
	return nil
}

// GetSearchSince вычисляет дату для поиска на основе типа и провайдера
func (s *EmailSearchService) GetSearchSince(ctx context.Context, searchType string, provider string) (time.Time, error) {
	// Получаем thread search конфигурацию
	threadConfig, err := s.GetThreadSearchConfig(ctx)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to get thread search config: %w", err)
	}

	// Получаем провайдер-специфичную конфигурацию
	providerConfig, err := s.GetProviderSearchConfig(ctx, provider)
	if err != nil {
		s.logger.Warn(ctx, "Using thread config as fallback for provider",
			"provider", provider,
			"error", err.Error())
		// Используем thread конфигурацию как fallback
		return threadConfig.GetSearchSince(searchType), nil
	}

	// Вычисляем дату на основе типа поиска
	var daysBack int
	switch searchType {
	case "standard":
		daysBack = threadConfig.DefaultDaysBack()
	case "extended":
		daysBack = threadConfig.ExtendedDaysBack()
	case "maximum":
		daysBack = threadConfig.MaxDaysBack()
		// Ограничиваем максимальным значением провайдера
		if daysBack > providerConfig.MaxDaysBack {
			daysBack = providerConfig.MaxDaysBack
			s.logger.Debug(ctx, "Adjusted days back to provider maximum",
				"provider", provider,
				"provider_max_days", providerConfig.MaxDaysBack)
		}
	default:
		daysBack = threadConfig.DefaultDaysBack()
	}

	searchSince := time.Now().Add(-time.Duration(daysBack) * 24 * time.Hour)

	s.logger.Debug(ctx, "Calculated search since date",
		"search_type", searchType,
		"provider", provider,
		"days_back", daysBack,
		"search_since", searchSince.Format("2006-01-02"))

	return searchSince, nil
}

// GenerateSearchSubjectVariants генерирует варианты subject для поиска
func (s *EmailSearchService) GenerateSearchSubjectVariants(ctx context.Context, baseSubject string) ([]string, error) {
	config, err := s.GetThreadSearchConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get search config for subject variants: %w", err)
	}

	variants := config.GenerateSubjectVariants(baseSubject)

	s.logger.Debug(ctx, "Generated subject variants for search",
		"base_subject", baseSubject,
		"variants_count", len(variants),
		"variants", variants)

	return variants, nil
}
