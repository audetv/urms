// internal/infrastructure/email/pipeline_strategies/generic_strategy.go
package pipeline_strategies

import (
	"time"

	"github.com/audetv/urms/internal/core/domain"
	"github.com/audetv/urms/internal/core/ports"
	"github.com/audetv/urms/internal/infrastructure/email/search_strategies"
)

// GenericPipelineStrategy implements ports.PipelineStrategy for generic providers
type GenericPipelineStrategy struct {
	config *domain.EmailProviderConfig
	logger ports.Logger
}

// NewGenericPipelineStrategy creates a new generic strategy
func NewGenericPipelineStrategy(
	config *domain.EmailProviderConfig,
	logger ports.Logger,
) *GenericPipelineStrategy {
	return &GenericPipelineStrategy{
		config: config,
		logger: logger,
	}
}

// GetBatchSize returns the batch size from configuration or default
func (s *GenericPipelineStrategy) GetBatchSize() int {
	if s.config.PipelineConfig.FetchBatchSize > 0 {
		return s.config.PipelineConfig.FetchBatchSize
	}
	// Default for generic providers
	return 25
}

// GetWorkerCount returns the worker count from configuration or default
func (s *GenericPipelineStrategy) GetWorkerCount() int {
	if s.config.PipelineConfig.WorkerCount > 0 {
		return s.config.PipelineConfig.WorkerCount
	}
	// Default for generic providers
	return 3
}

// GetQueueSize returns the queue size from configuration or default
func (s *GenericPipelineStrategy) GetQueueSize() int {
	if s.config.PipelineConfig.QueueSize > 0 {
		return s.config.PipelineConfig.QueueSize
	}
	// Default for generic providers
	return 50
}

// GetFetchTimeout returns the fetch operation timeout from configuration
func (s *GenericPipelineStrategy) GetFetchTimeout() time.Duration {
	if s.config.PipelineConfig.FetchTimeout > 0 {
		return s.config.PipelineConfig.FetchTimeout
	}
	// Default for generic providers
	return 45 * time.Second
}

// GetProcessTimeout returns the message processing timeout from configuration
func (s *GenericPipelineStrategy) GetProcessTimeout() time.Duration {
	if s.config.PipelineConfig.ProcessTimeout > 0 {
		return s.config.PipelineConfig.ProcessTimeout
	}
	// Default for generic providers
	return 90 * time.Second
}

// GetRetryPolicy returns the retry policy from configuration
func (s *GenericPipelineStrategy) GetRetryPolicy() ports.RetryPolicy {
	maxRetries := s.config.PipelineConfig.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 2 // Default for generic
	}

	return ports.RetryPolicy{
		MaxAttempts:   maxRetries,
		BaseDelay:     3 * time.Second,
		MaxDelay:      45 * time.Second,
		BackoffFactor: 1.8,
	}
}

// GetSearchStrategy returns the generic search strategy
func (s *GenericPipelineStrategy) GetSearchStrategy() ports.SearchStrategy {
	return search_strategies.NewGenericSearchStrategy(s.config, s.logger)
}
