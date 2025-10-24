// internal/core/ports/pipeline_strategy_factory.go
package ports

// PipelineStrategyFactory определяет контракт для фабрики pipeline стратегий
type PipelineStrategyFactory interface {
	// GetPipelineStrategy возвращает pipeline стратегию для провайдера
	GetPipelineStrategy(providerType string) PipelineStrategy

	// RegisterPipelineStrategy регистрирует новую pipeline стратегию
	RegisterPipelineStrategy(providerType string, strategy PipelineStrategy) error

	// GetSupportedPipelineStrategies возвращает список поддерживаемых стратегий
	GetSupportedPipelineStrategies() []string

	// Health проверяет состояние фабрики
	Health() error
}
