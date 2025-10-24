// internal/infrastructure/email/pipeline_strategies/gmail_strategy.go
package pipeline_strategies

import (
	"time"

	"github.com/audetv/urms/internal/core/domain"
	"github.com/audetv/urms/internal/core/ports"
	"github.com/audetv/urms/internal/infrastructure/email/search_strategies"
)

// GmailPipelineStrategy implements ports.PipelineStrategy for Gmail
type GmailPipelineStrategy struct {
	config *domain.EmailProviderConfig
	logger ports.Logger
}

// NewGmailPipelineStrategy creates a new Gmail-optimized strategy
func NewGmailPipelineStrategy(
	config *domain.EmailProviderConfig,
	logger ports.Logger,
) *GmailPipelineStrategy {
	return &GmailPipelineStrategy{
		config: config,
		logger: logger,
	}
}

// GetBatchSize returns the optimal batch size for Gmail from configuration
func (s *GmailPipelineStrategy) GetBatchSize() int {
	if s.config.PipelineConfig.FetchBatchSize > 0 {
		return s.config.PipelineConfig.FetchBatchSize
	}
	// Default for Gmail if not configured
	return 50
}

// GetWorkerCount returns the optimal number of workers for Gmail from configuration
func (s *GmailPipelineStrategy) GetWorkerCount() int {
	if s.config.PipelineConfig.WorkerCount > 0 {
		return s.config.PipelineConfig.WorkerCount
	}
	// Default for Gmail if not configured
	return 5
}

// GetQueueSize returns the optimal queue size for Gmail from configuration
func (s *GmailPipelineStrategy) GetQueueSize() int {
	if s.config.PipelineConfig.QueueSize > 0 {
		return s.config.PipelineConfig.QueueSize
	}
	// Default for Gmail if not configured
	return 100
}

// GetFetchTimeout returns the fetch operation timeout from configuration
func (s *GmailPipelineStrategy) GetFetchTimeout() time.Duration {
	if s.config.PipelineConfig.FetchTimeout > 0 {
		return s.config.PipelineConfig.FetchTimeout
	}
	// Default for Gmail if not configured
	return 60 * time.Second
}

// GetProcessTimeout returns the message processing timeout from configuration
func (s *GmailPipelineStrategy) GetProcessTimeout() time.Duration {
	if s.config.PipelineConfig.ProcessTimeout > 0 {
		return s.config.PipelineConfig.ProcessTimeout
	}
	// Default for Gmail if not configured
	return 120 * time.Second
}

// GetRetryPolicy returns the retry policy for Gmail operations
func (s *GmailPipelineStrategy) GetRetryPolicy() ports.RetryPolicy {
	maxRetries := s.config.PipelineConfig.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 3 // Default for Gmail
	}

	return ports.RetryPolicy{
		MaxAttempts:   maxRetries,
		BaseDelay:     2 * time.Second,
		MaxDelay:      60 * time.Second,
		BackoffFactor: 2.0,
	}
}

// GetSearchStrategy returns the Gmail-specific search strategy
func (s *GmailPipelineStrategy) GetSearchStrategy() ports.SearchStrategy {
	return search_strategies.NewGmailSearchStrategy(s.config, s.logger)
}
