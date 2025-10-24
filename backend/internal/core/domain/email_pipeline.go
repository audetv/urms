// internal/core/domain/email_pipeline.go
package domain

import (
	"time"
)

// EmailProviderConfig unified configuration for all email providers
type EmailProviderConfig struct {
	ProviderType     string
	PipelineStrategy string
	SearchConfig     SearchStrategyConfig
	PipelineConfig   PipelineRuntimeConfig
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// SearchStrategyConfig defines search strategy parameters
type SearchStrategyConfig struct {
	Complexity      SearchComplexity
	MaxMessageIDs   int
	TimeframeDays   int
	SubjectPrefixes []string
	Enabled         bool
}

// PipelineRuntimeConfig defines pipeline runtime parameters
type PipelineRuntimeConfig struct {
	FetchBatchSize   int
	WorkerCount      int
	QueueSize        int
	FetchTimeout     time.Duration
	ProcessTimeout   time.Duration
	MaxRetries       int
	EnableMonitoring bool
	EnableMetrics    bool
}

type SearchComplexity int

const (
	SearchComplexitySimple SearchComplexity = iota
	SearchComplexityModerate
	SearchComplexityComplex
)

// PipelineMetrics represents business metrics for pipeline observability
type PipelineMetrics struct {
	ProviderType      string
	StartTime         time.Time
	FetchDuration     time.Duration
	QueueSize         int
	WorkersActive     int
	MessagesProcessed int
	MessagesFailed    int
	TotalDuration     time.Duration
	Throughput        float64 // messages per second
	ErrorRate         float64
}

// FetchProgress tracks email fetching progress
type FetchProgress struct {
	TotalMessages      int
	ProcessedCount     int
	LastFetchTime      time.Time
	CurrentBatch       int
	EstimatedRemaining time.Duration
	StatusMessage      string
}

// PipelineStatus represents the current state of the pipeline
type PipelineStatus struct {
	Status          string
	ActiveSince     time.Time
	CurrentPhase    string
	MessagesInQueue int
	WorkersBusy     int
	LastError       string
}

// WorkerStats contains statistics for worker performance
type WorkerStats struct {
	WorkerID       string
	TasksProcessed int
	TasksFailed    int
	AvgProcessTime time.Duration
	LastActivity   time.Time
	Status         string
}
