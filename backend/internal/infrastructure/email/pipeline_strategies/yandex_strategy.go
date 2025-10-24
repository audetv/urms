// internal/infrastructure/email/pipeline_strategies/yandex_strategy.go
package pipeline_strategies

import (
	"time"

	"github.com/audetv/urms/internal/core/domain"
	"github.com/audetv/urms/internal/core/ports"
	"github.com/audetv/urms/internal/infrastructure/email/search_strategies"
)

// YandexPipelineStrategy implements ports.PipelineStrategy for Yandex
type YandexPipelineStrategy struct {
	config *domain.EmailProviderConfig
	logger ports.Logger
}

// NewYandexPipelineStrategy creates a new Yandex-optimized strategy
func NewYandexPipelineStrategy(
	config *domain.EmailProviderConfig,
	logger ports.Logger,
) *YandexPipelineStrategy {
	return &YandexPipelineStrategy{
		config: config,
		logger: logger,
	}
}

// GetBatchSize returns the optimal batch size for Yandex from configuration
func (s *YandexPipelineStrategy) GetBatchSize() int {
	return s.config.PipelineConfig.GetFetchBatchSize()
}

// GetWorkerCount returns the optimal number of workers for Yandex from configuration
func (s *YandexPipelineStrategy) GetWorkerCount() int {
	return s.config.PipelineConfig.GetWorkerCount()
}

// GetQueueSize returns the optimal queue size for Yandex from configuration
func (s *YandexPipelineStrategy) GetQueueSize() int {
	if s.config.PipelineConfig.QueueSize > 0 {
		return s.config.PipelineConfig.QueueSize
	}
	return 20
}

// GetFetchTimeout returns the fetch operation timeout from configuration
func (s *YandexPipelineStrategy) GetFetchTimeout() time.Duration {
	if s.config.PipelineConfig.FetchTimeout > 0 {
		return s.config.PipelineConfig.FetchTimeout
	}
	return 30 * time.Second
}

// GetProcessTimeout returns the message processing timeout from configuration
func (s *YandexPipelineStrategy) GetProcessTimeout() time.Duration {
	if s.config.PipelineConfig.ProcessTimeout > 0 {
		return s.config.PipelineConfig.ProcessTimeout
	}
	return 60 * time.Second
}

// GetRetryPolicy returns the retry policy for Yandex operations
func (s *YandexPipelineStrategy) GetRetryPolicy() ports.RetryPolicy {
	maxRetries := s.config.PipelineConfig.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 2
	}

	return ports.RetryPolicy{
		MaxAttempts:   maxRetries,
		BaseDelay:     5 * time.Second,
		MaxDelay:      30 * time.Second,
		BackoffFactor: 1.5,
	}
}

// GetSearchStrategy returns the Yandex-specific search strategy
func (s *YandexPipelineStrategy) GetSearchStrategy() ports.SearchStrategy {
	return search_strategies.NewYandexSearchStrategy(s.config, s.logger)
}
