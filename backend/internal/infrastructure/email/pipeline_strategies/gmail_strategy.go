// internal/infrastructure/email/pipeline_strategies/gmail_strategy.go
package pipeline_strategies

import (
	"context"
	"time"

	"github.com/audetv/urms/internal/core/domain"
	"github.com/audetv/urms/internal/core/ports"
	"github.com/audetv/urms/internal/infrastructure/email/search_strategies"
)

// GmailPipelineStrategy реализует ports.PipelineStrategy для Gmail
type GmailPipelineStrategy struct {
	config *domain.EmailProviderConfig
	logger ports.Logger
}

// GetSearchStrategy возвращает Gmail-specific search strategy
func (s *GmailPipelineStrategy) GetSearchStrategy() ports.SearchStrategy {
	// Создаем структуру напрямую
	searchStrategy := &search_strategies.GmailSearchStrategy{}

	// Конфигурируем стратегию
	if err := searchStrategy.Configure(&s.config.SearchConfig); err != nil {
		s.logger.Warn(context.Background(),
			"Failed to configure Gmail search strategy",
			"error", err.Error())
	}

	return searchStrategy
}

// GetBatchSize возвращает batch size для Gmail
func (s *GmailPipelineStrategy) GetBatchSize() int {
	if s.config.PipelineConfig.FetchBatchSize > 0 {
		return s.config.PipelineConfig.FetchBatchSize
	}
	return 50 // Default для Gmail
}

// GetWorkerCount возвращает количество воркеров для Gmail
func (s *GmailPipelineStrategy) GetWorkerCount() int {
	if s.config.PipelineConfig.WorkerCount > 0 {
		return s.config.PipelineConfig.WorkerCount
	}
	return 5 // Default для Gmail
}

// GetQueueSize возвращает размер очереди для Gmail
func (s *GmailPipelineStrategy) GetQueueSize() int {
	if s.config.PipelineConfig.QueueSize > 0 {
		return s.config.PipelineConfig.QueueSize
	}
	return 100 // Default для Gmail
}

// GetFetchTimeout возвращает fetch timeout для Gmail
func (s *GmailPipelineStrategy) GetFetchTimeout() time.Duration {
	if s.config.PipelineConfig.FetchTimeout > 0 {
		return s.config.PipelineConfig.FetchTimeout
	}
	return 60 * time.Second // Default для Gmail
}

// GetProcessTimeout возвращает process timeout для Gmail
func (s *GmailPipelineStrategy) GetProcessTimeout() time.Duration {
	if s.config.PipelineConfig.ProcessTimeout > 0 {
		return s.config.PipelineConfig.ProcessTimeout
	}
	return 120 * time.Second // Default для Gmail
}

// GetRetryPolicy возвращает retry policy для Gmail
func (s *GmailPipelineStrategy) GetRetryPolicy() ports.RetryPolicy {
	maxRetries := s.config.PipelineConfig.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 3 // Default для Gmail
	}

	return ports.RetryPolicy{
		MaxAttempts:   maxRetries,
		BaseDelay:     2 * time.Second,
		MaxDelay:      60 * time.Second,
		BackoffFactor: 2.0,
	}
}

// GetSearchStrategyConfig возвращает конфигурацию search strategy
func (s *GmailPipelineStrategy) GetSearchStrategyConfig() domain.SearchStrategyConfig {
	return s.config.SearchConfig
}

// GetProviderType возвращает тип провайдера
func (s *GmailPipelineStrategy) GetProviderType() string {
	return "gmail"
}
