// internal/core/ports/search_strategy_factory.go
package ports

import (
	"context"
)

// SearchStrategyFactory определяет контракт для фабрики поисковых стратегий
// НЕ знает о domain типах - только о ports интерфейсах
type SearchStrategyFactory interface {
	// GetSearchStrategy возвращает поисковую стратегию для провайдера
	GetSearchStrategy(ctx context.Context, providerType string) (SearchStrategy, error)

	// RegisterSearchStrategy регистрирует новую поисковую стратегию
	RegisterSearchStrategy(ctx context.Context, providerType string, strategy SearchStrategy) error

	// GetSupportedSearchStrategies возвращает список поддерживаемых стратегий
	GetSupportedSearchStrategies(ctx context.Context) ([]string, error)

	// Health проверяет состояние фабрики
	Health(ctx context.Context) error
}
