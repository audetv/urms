// internal/core/domain/email_pipeline.go
package domain

import (
	"fmt"
	"time"
)

// SearchComplexity определяет уровни сложности поисковых операций
// Это ЧИСТЫЙ domain тип, без внешних зависимостей
type SearchComplexity int

const (
	SearchComplexitySimple SearchComplexity = iota
	SearchComplexityModerate
	SearchComplexityComplex
)

// String реализует Stringer interface для лучшего логирования
func (sc SearchComplexity) String() string {
	switch sc {
	case SearchComplexitySimple:
		return "SIMPLE"
	case SearchComplexityModerate:
		return "MODERATE"
	case SearchComplexityComplex:
		return "COMPLEX"
	default:
		return fmt.Sprintf("UNKNOWN(%d)", sc)
	}
}

// EmailProviderConfig содержит полную конфигурацию email провайдера
type EmailProviderConfig struct {
	ProviderType     string
	PipelineStrategy string
	SearchConfig     SearchStrategyConfig
	PipelineConfig   PipelineRuntimeConfig
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// Validate проверяет валидность конфигурации
func (c *EmailProviderConfig) Validate() error {
	if c.ProviderType == "" {
		return fmt.Errorf("provider type is required")
	}

	if err := c.SearchConfig.Validate(); err != nil {
		return fmt.Errorf("search config validation failed: %w", err)
	}

	if err := c.PipelineConfig.Validate(); err != nil {
		return fmt.Errorf("pipeline config validation failed: %w", err)
	}

	return nil
}

// GetSearchComplexity возвращает настроенную сложность или разумный дефолт
func (c *EmailProviderConfig) GetSearchComplexity() SearchComplexity {
	if c.SearchConfig.Complexity != 0 {
		return c.SearchConfig.Complexity
	}
	// Разумный дефолт для большинства провайдеров
	return SearchComplexityModerate
}

// GetMaxMessageIDs возвращает настроенное максимальное количество Message-ID
func (c *EmailProviderConfig) GetMaxMessageIDs() int {
	if c.SearchConfig.MaxMessageIDs > 0 {
		return c.SearchConfig.MaxMessageIDs
	}
	// Разумный дефолт для баланса между точностью и производительностью
	return 5
}

// GetTimeframeDays возвращает настроенный временной диапазон
func (c *EmailProviderConfig) GetTimeframeDays() int {
	if c.SearchConfig.TimeframeDays > 0 {
		return c.SearchConfig.TimeframeDays
	}
	// Разумный дефолт - 3 месяца для большинства бизнес-кейсов
	return 90
}

// SearchStrategyConfig конфигурация поисковой стратегии
type SearchStrategyConfig struct {
	Complexity      SearchComplexity
	MaxMessageIDs   int
	TimeframeDays   int
	SubjectPrefixes []string
	Enabled         bool
}

// Validate проверяет валидность поисковой конфигурации
func (c *SearchStrategyConfig) Validate() error {
	if c.MaxMessageIDs < 0 {
		return fmt.Errorf("max message IDs cannot be negative")
	}
	if c.TimeframeDays < 0 {
		return fmt.Errorf("timeframe days cannot be negative")
	}
	if c.TimeframeDays > 730 { // 2 years reasonable maximum
		return fmt.Errorf("timeframe days cannot exceed 730")
	}
	return nil
}

// PipelineRuntimeConfig конфигурация runtime параметров pipeline
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

// Validate проверяет валидность runtime конфигурации
func (c *PipelineRuntimeConfig) Validate() error {
	if c.FetchBatchSize <= 0 {
		return fmt.Errorf("fetch batch size must be positive")
	}
	if c.WorkerCount <= 0 {
		return fmt.Errorf("worker count must be positive")
	}
	if c.QueueSize <= 0 {
		return fmt.Errorf("queue size must be positive")
	}
	if c.FetchTimeout <= 0 {
		return fmt.Errorf("fetch timeout must be positive")
	}
	if c.ProcessTimeout <= 0 {
		return fmt.Errorf("process timeout must be positive")
	}
	if c.MaxRetries < 0 {
		return fmt.Errorf("max retries cannot be negative")
	}
	return nil
}

// GetFetchBatchSize возвращает размер батча с валидацией
func (c *PipelineRuntimeConfig) GetFetchBatchSize() int {
	if c.FetchBatchSize > 0 {
		return c.FetchBatchSize
	}
	return 25 // Разумный дефолт
}

// GetWorkerCount возвращает количество воркеров с валидацией
func (c *PipelineRuntimeConfig) GetWorkerCount() int {
	if c.WorkerCount > 0 {
		return c.WorkerCount
	}
	return 3 // Разумный дефолт
}

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
