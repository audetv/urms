// internal/core/ports/search_strategy_provider.go
package ports

import "github.com/audetv/urms/internal/core/domain"

// SearchStrategyProvider предоставляет конфигурацию для поисковых стратегий
type SearchStrategyProvider interface {
	// GetSearchStrategyConfig возвращает конфигурацию поисковой стратегии
	GetSearchStrategyConfig() *domain.SearchStrategyConfig

	// GetProviderType возвращает тип провайдера
	GetProviderType() string
}
