// internal/infrastructure/email/pipeline_strategies/generic_strategy.go
package pipeline_strategies

import (
	"context"
	"time"

	"github.com/audetv/urms/internal/core/domain"
	"github.com/audetv/urms/internal/core/ports"
	"github.com/audetv/urms/internal/infrastructure/email/search_strategies"
)

// GenericPipelineStrategy реализует ports.PipelineStrategy для generic провайдеров
type GenericPipelineStrategy struct {
	config *domain.EmailProviderConfig
	logger ports.Logger
}

// GetSearchStrategy возвращает generic search strategy
func (s *GenericPipelineStrategy) GetSearchStrategy() ports.SearchStrategy {
	// Создаем структуру напрямую
	searchStrategy := &search_strategies.GenericSearchStrategy{}

	// Конфигурируем стратегию
	if err := searchStrategy.Configure(&s.config.SearchConfig); err != nil {
		s.logger.Warn(context.Background(),
			"Failed to configure Generic search strategy",
			"error", err.Error())
	}

	return searchStrategy
}

// GetBatchSize возвращает batch size для generic провайдеров
func (s *GenericPipelineStrategy) GetBatchSize() int {
	if s.config.PipelineConfig.FetchBatchSize > 0 {
		return s.config.PipelineConfig.FetchBatchSize
	}
	return 25 // Default для generic
}

// GetWorkerCount возвращает количество воркеров для generic провайдеров
func (s *GenericPipelineStrategy) GetWorkerCount() int {
	if s.config.PipelineConfig.WorkerCount > 0 {
		return s.config.PipelineConfig.WorkerCount
	}
	return 3 // Default для generic
}

// GetQueueSize возвращает размер очереди для generic провайдеров
func (s *GenericPipelineStrategy) GetQueueSize() int {
	if s.config.PipelineConfig.QueueSize > 0 {
		return s.config.PipelineConfig.QueueSize
	}
	return 50 // Default для generic
}

// GetFetchTimeout возвращает fetch timeout для generic провайдеров
func (s *GenericPipelineStrategy) GetFetchTimeout() time.Duration {
	if s.config.PipelineConfig.FetchTimeout > 0 {
		return s.config.PipelineConfig.FetchTimeout
	}
	return 45 * time.Second // Default для generic
}

// GetProcessTimeout возвращает process timeout для generic провайдеров
func (s *GenericPipelineStrategy) GetProcessTimeout() time.Duration {
	if s.config.PipelineConfig.ProcessTimeout > 0 {
		return s.config.PipelineConfig.ProcessTimeout
	}
	return 90 * time.Second // Default для generic
}

// GetRetryPolicy возвращает retry policy для generic провайдеров
func (s *GenericPipelineStrategy) GetRetryPolicy() ports.RetryPolicy {
	maxRetries := s.config.PipelineConfig.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 2 // Default для generic
	}

	return ports.RetryPolicy{
		MaxAttempts:   maxRetries,
		BaseDelay:     3 * time.Second,
		MaxDelay:      45 * time.Second,
		BackoffFactor: 1.8,
	}
}

// GetSearchStrategyConfig возвращает конфигурацию search strategy
func (s *GenericPipelineStrategy) GetSearchStrategyConfig() domain.SearchStrategyConfig {
	return s.config.SearchConfig
}

// GetProviderType возвращает тип провайдера
func (s *GenericPipelineStrategy) GetProviderType() string {
	return "generic"
}
