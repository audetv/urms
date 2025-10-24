// internal/core/ports/pipeline_types.go
package ports

import (
	"time"
)

// SearchComplexity defines the complexity level of search operations
type SearchComplexity int

const (
	SearchComplexitySimple SearchComplexity = iota
	SearchComplexityModerate
	SearchComplexityComplex
)

// PipelineHealth represents pipeline health status
type PipelineHealth struct {
	Status     string
	Timestamp  time.Time
	Components map[string]string
	Message    string
}

// PipelineMetrics represents pipeline performance metrics
type PipelineMetrics struct {
	ProviderType     string
	Uptime           time.Duration
	TotalProcessed   int64
	TotalFailed      int64
	CurrentQueueSize int
	WorkersActive    int
	WorkersTotal     int
	LastProcessed    time.Time
	AvgProcessTime   time.Duration
	ErrorRate        float64
}

// WorkerMetrics represents worker pool metrics
type WorkerMetrics struct {
	WorkersActive  int
	WorkersIdle    int
	WorkersTotal   int
	TasksProcessed int64
	TasksFailed    int64
	AvgProcessTime time.Duration
	QueueWaitTime  time.Duration
	LastActivity   time.Time
}

// FetchProgress represents fetch operation progress
type FetchProgress struct {
	TotalMessages      int
	FetchedCount       int
	LastFetchTime      time.Time
	CurrentBatch       int
	EstimatedRemaining time.Duration
	Status             string
}

// RetryPolicy defines retry behavior for operations
type RetryPolicy struct {
	MaxAttempts   int
	BaseDelay     time.Duration
	MaxDelay      time.Duration
	BackoffFactor float64
}

// PipelineConfig contains complete pipeline configuration
type PipelineConfig struct {
	ProviderType    string
	Strategy        PipelineStrategy
	MaxConcurrent   int
	ShutdownTimeout time.Duration
}

// ComponentStatus represents status of individual pipeline components
type ComponentStatus struct {
	Name    string
	Status  string
	Message string
	Since   time.Time
}
