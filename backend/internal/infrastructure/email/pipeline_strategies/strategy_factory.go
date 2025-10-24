// internal/infrastructure/email/pipeline_strategies/strategy_factory.go
package pipeline_strategies

import (
	"context"
	"fmt"
	"strings"

	"github.com/audetv/urms/internal/core/domain"
	"github.com/audetv/urms/internal/core/ports"
)

// PipelineStrategyFactoryImpl implements ports.PipelineStrategyFactory
type PipelineStrategyFactoryImpl struct {
	strategies map[string]ports.PipelineStrategy
	config     *domain.EmailProviderConfig
	logger     ports.Logger
}

// NewPipelineStrategyFactory creates a new factory with configuration
func NewPipelineStrategyFactory(
	config *domain.EmailProviderConfig,
	logger ports.Logger,
) *PipelineStrategyFactoryImpl {
	factory := &PipelineStrategyFactoryImpl{
		strategies: make(map[string]ports.PipelineStrategy),
		config:     config,
		logger:     logger,
	}

	// Register all available strategies
	factory.registerStrategies()

	return factory
}

// registerStrategies initializes all provider-specific strategies
func (f *PipelineStrategyFactoryImpl) registerStrategies() {
	// Yandex strategy
	yandexStrategy := NewYandexPipelineStrategy(f.config, f.logger)
	f.strategies["yandex"] = yandexStrategy

	// Gmail strategy
	gmailStrategy := NewGmailPipelineStrategy(f.config, f.logger)
	f.strategies["gmail"] = gmailStrategy

	// Generic fallback strategy
	genericStrategy := NewGenericPipelineStrategy(f.config, f.logger)
	f.strategies["generic"] = genericStrategy

	f.logger.Info(context.Background(), "Pipeline strategies registered",
		"yandex_enabled", true,
		"gmail_enabled", true,
		"generic_enabled", true)
}

// GetStrategy returns the appropriate strategy for the provider
func (f *PipelineStrategyFactoryImpl) GetStrategy(providerType string) ports.PipelineStrategy {
	// Try exact match first
	if strategy, exists := f.strategies[providerType]; exists {
		f.logger.Debug(context.Background(), "Using exact match strategy",
			"provider", providerType,
			"strategy_type", fmt.Sprintf("%T", strategy))
		return strategy
	}

	// Try partial match (e.g., "imap.yandex.ru" → "yandex")
	for key, strategy := range f.strategies {
		if containsProvider(providerType, key) {
			f.logger.Debug(context.Background(), "Using partial match strategy",
				"provider", providerType,
				"matched_key", key,
				"strategy_type", fmt.Sprintf("%T", strategy))
			return strategy
		}
	}

	// Fallback to generic
	f.logger.Warn(context.Background(), "No specific strategy found, using generic fallback",
		"provider", providerType)
	return f.strategies["generic"]
}

// RegisterStrategy registers a new strategy implementation
func (f *PipelineStrategyFactoryImpl) RegisterStrategy(providerType string, strategy ports.PipelineStrategy) {
	f.strategies[providerType] = strategy
	f.logger.Info(context.Background(), "Custom strategy registered",
		"provider", providerType,
		"strategy_type", fmt.Sprintf("%T", strategy))
}

// GetSupportedProviders returns list of supported providers
func (f *PipelineStrategyFactoryImpl) GetSupportedProviders() []string {
	providers := make([]string, 0, len(f.strategies))
	for provider := range f.strategies {
		providers = append(providers, provider)
	}
	return providers
}

// containsProvider checks if provider string contains the strategy key
func containsProvider(provider, key string) bool {
	providerLower := strings.ToLower(provider)
	keyLower := strings.ToLower(key)
	return strings.Contains(providerLower, keyLower)
}
