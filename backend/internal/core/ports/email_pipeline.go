// internal/core/ports/email_pipeline.go
package ports

import (
	"context"
	"time"

	"github.com/audetv/urms/internal/core/domain"
	"github.com/emersion/go-imap"
)

// EmailPipeline defines the main pipeline contract for processing emails
type EmailPipeline interface {
	// Start begins the pipeline processing
	Start(ctx context.Context) error

	// Stop gracefully shuts down the pipeline
	Stop(ctx context.Context) error

	// Health returns the current health status of the pipeline
	Health(ctx context.Context) (*PipelineHealth, error)

	// GetMetrics returns performance and operational metrics
	GetMetrics(ctx context.Context) (*PipelineMetrics, error)

	// ProcessBatch processes a single batch of emails (for testing/manual execution)
	ProcessBatch(ctx context.Context) error
}

// EmailFetcher handles batch email fetching with provider-specific strategies
type EmailFetcher interface {
	// FetchBatch retrieves a batch of emails based on criteria
	FetchBatch(ctx context.Context, criteria FetchCriteria) ([]domain.EmailMessage, error)

	// GetProviderType returns the email provider type (gmail, yandex, etc.)
	GetProviderType() string

	// GetProgress returns the current fetch operation progress
	GetProgress(ctx context.Context) *FetchProgress

	// Health checks the health of the fetcher component
	Health(ctx context.Context) error
}

// MessageQueue provides buffering between fetcher and processors
type MessageQueue interface {
	// Enqueue adds messages to the queue
	Enqueue(ctx context.Context, messages []domain.EmailMessage) error

	// Dequeue retrieves messages from the queue
	Dequeue(ctx context.Context, batchSize int) ([]domain.EmailMessage, error)

	// Size returns the current number of messages in the queue
	Size(ctx context.Context) (int, error)

	// Health checks the health of the queue component
	Health(ctx context.Context) error

	// Clear removes all messages from the queue (for testing/reset)
	Clear(ctx context.Context) error
}

// WorkerPool manages concurrent message processing
type WorkerPool interface {
	// Start begins the worker pool
	Start(ctx context.Context) error

	// Stop gracefully shuts down the worker pool
	Stop(ctx context.Context) error

	// Submit sends a message to the worker pool for processing
	Submit(ctx context.Context, message domain.EmailMessage) error

	// GetMetrics returns worker pool performance metrics
	GetMetrics(ctx context.Context) *WorkerMetrics

	// Health checks the health of the worker pool
	Health(ctx context.Context) error
}

// PipelineStrategy defines provider-specific pipeline configuration
type PipelineStrategy interface {
	// GetBatchSize returns the optimal batch size for the provider
	GetBatchSize() int

	// GetWorkerCount returns the optimal number of workers
	GetWorkerCount() int

	// GetQueueSize returns the optimal queue size
	GetQueueSize() int

	// GetFetchTimeout returns the fetch operation timeout
	GetFetchTimeout() time.Duration

	// GetProcessTimeout returns the message processing timeout
	GetProcessTimeout() time.Duration

	// GetRetryPolicy returns the retry policy for operations
	GetRetryPolicy() RetryPolicy

	// GetSearchStrategy returns the provider-specific search strategy
	GetSearchStrategy() SearchStrategy
}

// SearchStrategy defines provider-specific search behavior
type SearchStrategy interface {
	// Configure настраивает стратегию с предоставленной конфигурацией
	Configure(config *domain.SearchStrategyConfig) error

	// CreateThreadSearchCriteria создает IMAP критерии поиска
	CreateThreadSearchCriteria(threadData ThreadSearchCriteria) (*imap.SearchCriteria, error)

	// GetComplexity возвращает уровень сложности поиска
	GetComplexity() domain.SearchComplexity

	// GetMaxMessageIDs возвращает максимальное количество Message-ID
	GetMaxMessageIDs() int

	// GetTimeframeDays возвращает временной диапазон
	GetTimeframeDays() int
}

// PipelineStrategyFactory creates pipeline strategies based on provider type
type PipelineStrategyFactory interface {
	// GetStrategy returns the appropriate strategy for the provider
	GetStrategy(providerType string) PipelineStrategy

	// RegisterStrategy registers a new strategy implementation
	RegisterStrategy(providerType string, strategy PipelineStrategy)

	// GetSupportedProviders returns list of supported providers
	GetSupportedProviders() []string
}
