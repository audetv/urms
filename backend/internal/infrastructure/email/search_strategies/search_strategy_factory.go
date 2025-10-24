// internal/infrastructure/email/search_strategies/search_strategy_factory.go
package search_strategies

import (
	"context"
	"fmt"
	"strings"

	"github.com/audetv/urms/internal/core/ports"
)

// SearchStrategyFactoryImpl реализует ports.SearchStrategyFactory
type SearchStrategyFactoryImpl struct {
	strategies map[string]ports.SearchStrategy
	logger     ports.Logger
}

// NewSearchStrategyFactory создает новую фабрику поисковых стратегий
func NewSearchStrategyFactory(logger ports.Logger) *SearchStrategyFactoryImpl {
	factory := &SearchStrategyFactoryImpl{
		strategies: make(map[string]ports.SearchStrategy),
		logger:     logger,
	}

	// Базовая инициализация - стратегии регистрируются динамически
	factory.logger.Info(context.Background(),
		"Search strategy factory initialized",
		"initial_strategy_count", 0,
		"registration_method", "dynamic")

	return factory
}

// GetSearchStrategy возвращает поисковую стратегию для указанного провайдера
func (f *SearchStrategyFactoryImpl) GetSearchStrategy(
	ctx context.Context,
	providerType string,
) (ports.SearchStrategy, error) {

	if providerType == "" {
		return nil, fmt.Errorf("provider type cannot be empty")
	}

	// Поиск стратегии
	strategy := f.findMatchingStrategy(providerType)
	if strategy == nil {
		f.logger.Warn(ctx,
			"No specific search strategy found, using generic fallback",
			"provider_type", providerType,
			"available_strategies", f.getAvailableStrategyKeys())

		strategy = f.strategies["generic"]
		if strategy == nil {
			return nil, fmt.Errorf("no search strategy available for provider: %s", providerType)
		}
	}

	f.logger.Debug(ctx,
		"Search strategy selected",
		"provider_type", providerType,
		"strategy_type", fmt.Sprintf("%T", strategy))

	return strategy, nil
}

// RegisterSearchStrategy регистрирует новую поисковую стратегию
func (f *SearchStrategyFactoryImpl) RegisterSearchStrategy(
	ctx context.Context,
	providerType string,
	strategy ports.SearchStrategy,
) error {

	if providerType == "" {
		return fmt.Errorf("provider type cannot be empty")
	}

	if strategy == nil {
		return fmt.Errorf("search strategy cannot be nil")
	}

	// Нормализация ключа провайдера
	normalizedKey := strings.ToLower(strings.TrimSpace(providerType))

	// Проверка на дубликат
	if existing, exists := f.strategies[normalizedKey]; exists {
		f.logger.Warn(ctx,
			"Overwriting existing search strategy",
			"provider_type", normalizedKey,
			"existing_strategy", fmt.Sprintf("%T", existing),
			"new_strategy", fmt.Sprintf("%T", strategy))
	}

	f.strategies[normalizedKey] = strategy

	f.logger.Info(ctx,
		"Search strategy registered successfully",
		"provider_type", normalizedKey,
		"strategy_type", fmt.Sprintf("%T", strategy),
		"strategy_complexity", strategy.GetComplexity().String(),
		"total_strategies", len(f.strategies))

	return nil
}

// GetSupportedSearchStrategies возвращает список поддерживаемых стратегий
func (f *SearchStrategyFactoryImpl) GetSupportedSearchStrategies(ctx context.Context) ([]string, error) {
	strategies := make([]string, 0, len(f.strategies))
	for providerType := range f.strategies {
		strategies = append(strategies, providerType)
	}

	f.logger.Debug(ctx,
		"Supported search strategies retrieved",
		"total_strategies", len(strategies),
		"strategies", strategies)

	return strategies, nil
}

// Health проверяет состояние фабрики
func (f *SearchStrategyFactoryImpl) Health(ctx context.Context) error {
	if len(f.strategies) == 0 {
		return fmt.Errorf("no search strategies registered")
	}

	// Проверяем, что есть generic fallback стратегия
	if f.strategies["generic"] == nil {
		return fmt.Errorf("generic fallback strategy not registered")
	}

	f.logger.Debug(ctx,
		"Search strategy factory health check passed",
		"registered_strategies", len(f.strategies),
		"has_generic_fallback", f.strategies["generic"] != nil)

	return nil
}

// findMatchingStrategy ищет подходящую стратегию для провайдера
func (f *SearchStrategyFactoryImpl) findMatchingStrategy(providerType string) ports.SearchStrategy {
	normalizedProvider := strings.ToLower(providerType)

	// 1. Точное совпадение
	if strategy, exists := f.strategies[normalizedProvider]; exists {
		return strategy
	}

	// 2. Частичное совпадение (например, "imap.yandex.ru" → "yandex")
	for key, strategy := range f.strategies {
		if strings.Contains(normalizedProvider, key) {
			return strategy
		}
	}

	// 3. Стратегия не найдена
	return nil
}

// getAvailableStrategyKeys возвращает ключи зарегистрированных стратегий
func (f *SearchStrategyFactoryImpl) getAvailableStrategyKeys() []string {
	keys := make([]string, 0, len(f.strategies))
	for key := range f.strategies {
		keys = append(keys, key)
	}
	return keys
}
