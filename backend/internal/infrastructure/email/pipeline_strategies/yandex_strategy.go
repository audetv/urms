// internal/infrastructure/email/pipeline_strategies/yandex_strategy.go
package pipeline_strategies

import (
	"context"
	"time"

	"github.com/audetv/urms/internal/core/domain"
	"github.com/audetv/urms/internal/core/ports"
	"github.com/audetv/urms/internal/infrastructure/email/search_strategies"
)

// YandexPipelineStrategy реализует ports.PipelineStrategy для Yandex
type YandexPipelineStrategy struct {
	config *domain.EmailProviderConfig
	logger ports.Logger
}

// GetSearchStrategy возвращает Yandex-specific search strategy
func (s *YandexPipelineStrategy) GetSearchStrategy() ports.SearchStrategy {
	// Создаем структуру напрямую
	searchStrategy := &search_strategies.YandexSearchStrategy{}

	// Конфигурируем стратегию
	if err := searchStrategy.Configure(&s.config.SearchConfig); err != nil {
		s.logger.Warn(context.Background(),
			"Failed to configure Yandex search strategy",
			"error", err.Error())
	}

	return searchStrategy
}

// GetBatchSize возвращает batch size для Yandex
func (s *YandexPipelineStrategy) GetBatchSize() int {
	if s.config.PipelineConfig.FetchBatchSize > 0 {
		return s.config.PipelineConfig.FetchBatchSize
	}
	return 10 // Default для Yandex
}

// GetWorkerCount возвращает количество воркеров для Yandex
func (s *YandexPipelineStrategy) GetWorkerCount() int {
	if s.config.PipelineConfig.WorkerCount > 0 {
		return s.config.PipelineConfig.WorkerCount
	}
	return 2 // Default для Yandex
}

// GetQueueSize возвращает размер очереди для Yandex
func (s *YandexPipelineStrategy) GetQueueSize() int {
	if s.config.PipelineConfig.QueueSize > 0 {
		return s.config.PipelineConfig.QueueSize
	}
	return 20 // Default для Yandex
}

// GetFetchTimeout возвращает fetch timeout для Yandex
func (s *YandexPipelineStrategy) GetFetchTimeout() time.Duration {
	if s.config.PipelineConfig.FetchTimeout > 0 {
		return s.config.PipelineConfig.FetchTimeout
	}
	return 30 * time.Second // Default для Yandex
}

// GetProcessTimeout возвращает process timeout для Yandex
func (s *YandexPipelineStrategy) GetProcessTimeout() time.Duration {
	if s.config.PipelineConfig.ProcessTimeout > 0 {
		return s.config.PipelineConfig.ProcessTimeout
	}
	return 60 * time.Second // Default для Yandex
}

// GetRetryPolicy возвращает retry policy для Yandex
func (s *YandexPipelineStrategy) GetRetryPolicy() ports.RetryPolicy {
	maxRetries := s.config.PipelineConfig.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 2 // Default для Yandex
	}

	return ports.RetryPolicy{
		MaxAttempts:   maxRetries,
		BaseDelay:     5 * time.Second,
		MaxDelay:      30 * time.Second,
		BackoffFactor: 1.5,
	}
}

// GetSearchStrategyConfig возвращает конфигурацию search strategy
func (s *YandexPipelineStrategy) GetSearchStrategyConfig() domain.SearchStrategyConfig {
	return s.config.SearchConfig
}

// GetProviderType возвращает тип провайдера
func (s *YandexPipelineStrategy) GetProviderType() string {
	return "yandex"
}
