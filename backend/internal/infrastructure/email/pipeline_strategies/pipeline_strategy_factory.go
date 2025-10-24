// internal/infrastructure/email/pipeline_strategies/pipeline_strategy_factory.go
package pipeline_strategies

import (
	"context"
	"fmt"
	"strings"

	"github.com/audetv/urms/internal/core/domain"
	"github.com/audetv/urms/internal/core/ports"
)

// PipelineStrategyFactoryImpl реализует ports.PipelineStrategyFactory
type PipelineStrategyFactoryImpl struct {
	strategies map[string]ports.PipelineStrategy
	config     *domain.EmailProviderConfig
	logger     ports.Logger
}

// NewPipelineStrategyFactory создает новую фабрику pipeline стратегий
func NewPipelineStrategyFactory(
	config *domain.EmailProviderConfig,
	logger ports.Logger,
) *PipelineStrategyFactoryImpl {

	factory := &PipelineStrategyFactoryImpl{
		strategies: make(map[string]ports.PipelineStrategy),
		config:     config,
		logger:     logger,
	}

	// Регистрируем стандартные стратегии
	factory.registerStandardStrategies()

	return factory
}

// registerStandardStrategies регистрирует стандартные pipeline стратегии
func (f *PipelineStrategyFactoryImpl) registerStandardStrategies() {
	// Yandex стратегия
	yandexStrategy := &YandexPipelineStrategy{
		config: f.config,
		logger: f.logger,
	}
	f.strategies["yandex"] = yandexStrategy

	// Gmail стратегия
	gmailStrategy := &GmailPipelineStrategy{
		config: f.config,
		logger: f.logger,
	}
	f.strategies["gmail"] = gmailStrategy

	// Generic стратегия
	genericStrategy := &GenericPipelineStrategy{
		config: f.config,
		logger: f.logger,
	}
	f.strategies["generic"] = genericStrategy

	f.logger.Info(context.Background(), "Pipeline strategies registered",
		"yandex_enabled", true,
		"gmail_enabled", true,
		"generic_enabled", true,
		"total_strategies", len(f.strategies))
}

// GetPipelineStrategy возвращает pipeline стратегию для провайдера
func (f *PipelineStrategyFactoryImpl) GetPipelineStrategy(providerType string) ports.PipelineStrategy {
	// Точное совпадение
	if strategy, exists := f.strategies[providerType]; exists {
		f.logger.Debug(context.Background(), "Using exact match pipeline strategy",
			"provider", providerType,
			"strategy_type", fmt.Sprintf("%T", strategy))
		return strategy
	}

	// Частичное совпадение
	for key, strategy := range f.strategies {
		if strings.Contains(strings.ToLower(providerType), strings.ToLower(key)) {
			f.logger.Debug(context.Background(), "Using partial match pipeline strategy",
				"provider", providerType,
				"matched_key", key,
				"strategy_type", fmt.Sprintf("%T", strategy))
			return strategy
		}
	}

	// Fallback на generic
	f.logger.Warn(context.Background(), "No specific pipeline strategy found, using generic fallback",
		"provider", providerType)
	return f.strategies["generic"]
}

// RegisterPipelineStrategy регистрирует новую pipeline стратегию
func (f *PipelineStrategyFactoryImpl) RegisterPipelineStrategy(providerType string, strategy ports.PipelineStrategy) error {
	if providerType == "" {
		return fmt.Errorf("provider type cannot be empty")
	}

	if strategy == nil {
		return fmt.Errorf("pipeline strategy cannot be nil")
	}

	normalizedKey := strings.ToLower(strings.TrimSpace(providerType))

	// Проверка на дубликат
	if existing, exists := f.strategies[normalizedKey]; exists {
		f.logger.Warn(context.Background(),
			"Overwriting existing pipeline strategy",
			"provider_type", normalizedKey,
			"existing_strategy", fmt.Sprintf("%T", existing),
			"new_strategy", fmt.Sprintf("%T", strategy))
	}

	f.strategies[normalizedKey] = strategy

	f.logger.Info(context.Background(),
		"Pipeline strategy registered successfully",
		"provider_type", normalizedKey,
		"strategy_type", fmt.Sprintf("%T", strategy),
		"total_strategies", len(f.strategies))

	return nil
}

// GetSupportedPipelineStrategies возвращает список поддерживаемых стратегий
func (f *PipelineStrategyFactoryImpl) GetSupportedPipelineStrategies() []string {
	strategies := make([]string, 0, len(f.strategies))
	for providerType := range f.strategies {
		strategies = append(strategies, providerType)
	}
	return strategies
}

// Health проверяет состояние фабрики
func (f *PipelineStrategyFactoryImpl) Health() error {
	if len(f.strategies) == 0 {
		return fmt.Errorf("no pipeline strategies registered")
	}

	if f.strategies["generic"] == nil {
		return fmt.Errorf("generic fallback strategy not registered")
	}

	f.logger.Debug(context.Background(),
		"Pipeline strategy factory health check passed",
		"registered_strategies", len(f.strategies),
		"has_generic_fallback", f.strategies["generic"] != nil)

	return nil
}
