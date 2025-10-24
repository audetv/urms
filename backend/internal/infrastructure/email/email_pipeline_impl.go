// internal/infrastructure/email/email_pipeline_impl.go
package email

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/audetv/urms/internal/core/domain"
	"github.com/audetv/urms/internal/core/ports"
)

// EmailPipelineImpl реализует ports.EmailPipeline
type EmailPipelineImpl struct {
	fetcher         ports.EmailFetcher
	queue           ports.MessageQueue
	workerPool      ports.WorkerPool
	strategy        ports.PipelineStrategy
	searchFactory   ports.SearchStrategyFactory
	logger          ports.Logger
	criteriaBuilder *FetchCriteriaBuilder
	metrics         *PipelineMetricsCollector
	status          *PipelineStatus
	shutdownTimeout time.Duration
	mu              sync.RWMutex
}

// PipelineMetricsCollector собирает метрики pipeline
type PipelineMetricsCollector struct {
	startTime      time.Time
	totalProcessed int64
	totalFailed    int64
	lastProcessed  time.Time
	processingTime time.Duration
	mu             sync.RWMutex
}

// PipelineStatus отслеживает состояние pipeline
type PipelineStatus struct {
	status       string
	activeSince  time.Time
	currentPhase string
	lastError    string
	mu           sync.RWMutex
}

// NewEmailPipelineImpl создает новый экземпляр Email Pipeline
func NewEmailPipelineImpl(
	fetcher ports.EmailFetcher,
	queue ports.MessageQueue,
	workerPool ports.WorkerPool,
	strategy ports.PipelineStrategy,
	searchFactory ports.SearchStrategyFactory,
	logger ports.Logger,
) *EmailPipelineImpl {

	return &EmailPipelineImpl{
		fetcher:         fetcher,
		queue:           queue,
		workerPool:      workerPool,
		strategy:        strategy,
		searchFactory:   searchFactory,
		logger:          logger,
		criteriaBuilder: NewFetchCriteriaBuilder(searchFactory, fetcher, logger),
		metrics:         &PipelineMetricsCollector{startTime: time.Now()},
		status:          &PipelineStatus{status: "created"},
		shutdownTimeout: 30 * time.Second,
	}
}

// Start запускает pipeline processing
func (p *EmailPipelineImpl) Start(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.logger.Info(ctx, "🚀 Starting email processing pipeline")

	// Проверяем здоровье компонентов
	if err := p.healthCheck(ctx); err != nil {
		return fmt.Errorf("pipeline health check failed: %w", err)
	}

	// Запускаем worker pool
	if err := p.workerPool.Start(ctx); err != nil {
		return fmt.Errorf("failed to start worker pool: %w", err)
	}

	// Обновляем статус
	p.status.setStatus("running", "initialization")
	p.status.activeSince = time.Now()

	p.logger.Info(ctx, "✅ Email pipeline started successfully",
		"provider_type", p.fetcher.GetProviderType(),
		"worker_count", p.strategy.GetWorkerCount(),
		"batch_size", p.strategy.GetBatchSize())

	return nil
}

// Stop останавливает pipeline
func (p *EmailPipelineImpl) Stop(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.logger.Info(ctx, "🛑 Stopping email processing pipeline")

	// Обновляем статус
	p.status.setStatus("stopping", "shutdown")

	// Создаем контекст с таймаутом для graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(ctx, p.shutdownTimeout)
	defer cancel()

	// Останавливаем worker pool
	if err := p.workerPool.Stop(shutdownCtx); err != nil {
		p.logger.Error(ctx, "Failed to stop worker pool gracefully",
			"error", err.Error())
	}

	// Очищаем очередь
	if err := p.queue.Clear(shutdownCtx); err != nil {
		p.logger.Warn(ctx, "Failed to clear message queue",
			"error", err.Error())
	}

	p.status.setStatus("stopped", "shutdown_complete")

	p.logger.Info(ctx, "✅ Email pipeline stopped successfully",
		"total_processed", p.metrics.getTotalProcessed(),
		"total_failed", p.metrics.getTotalFailed(),
		"uptime", time.Since(p.status.activeSince).String())

	return nil
}

// ProcessBatch обрабатывает один батч сообщений
func (p *EmailPipelineImpl) ProcessBatch(ctx context.Context) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.status.getStatus() != "running" {
		return fmt.Errorf("pipeline is not running, current status: %s", p.status.getStatus())
	}

	p.logger.Info(ctx, "🔍 Starting email batch processing")

	startTime := time.Now()
	p.status.setCurrentPhase("fetching")

	// 1. PHASE: Fetch emails
	messages, err := p.fetcher.FetchBatch(ctx, p.createFetchCriteria())
	if err != nil {
		p.metrics.recordFailure()
		p.status.setLastError(fmt.Sprintf("fetch failed: %v", err))
		return fmt.Errorf("failed to fetch emails: %w", err)
	}

	p.logger.Info(ctx, "✅ Email fetch completed",
		"message_count", len(messages),
		"fetch_duration", time.Since(startTime).String())

	if len(messages) == 0 {
		p.logger.Info(ctx, "📭 No new messages to process")
		return nil
	}

	p.status.setCurrentPhase("queuing")

	// 2. PHASE: Enqueue messages
	if err := p.queue.Enqueue(ctx, messages); err != nil {
		p.metrics.recordFailure()
		p.status.setLastError(fmt.Sprintf("enqueue failed: %v", err))
		return fmt.Errorf("failed to enqueue messages: %w", err)
	}

	p.logger.Debug(ctx, "✅ Messages enqueued successfully",
		"queue_size_after", p.getQueueSize(ctx))

	p.status.setCurrentPhase("processing")

	// 3. PHASE: Process messages through worker pool
	processedCount, failedCount := p.processMessages(ctx, messages)

	// 4. PHASE: Update metrics
	totalDuration := time.Since(startTime)
	p.metrics.recordProcessing(processedCount, failedCount, totalDuration)

	p.logger.Info(ctx, "🎉 Email batch processing completed",
		"total_messages", len(messages),
		"processed", processedCount,
		"failed", failedCount,
		"success_rate", fmt.Sprintf("%.1f%%", float64(processedCount)/float64(len(messages))*100),
		"total_duration", totalDuration.String(),
		"throughput", fmt.Sprintf("%.1f msg/sec", float64(len(messages))/totalDuration.Seconds()))

	p.status.setCurrentPhase("idle")

	return nil
}

// Health возвращает статус здоровья pipeline
func (p *EmailPipelineImpl) Health(ctx context.Context) (*ports.PipelineHealth, error) {
	components := make(map[string]string)

	// Проверяем здоровье всех компонентов
	if err := p.fetcher.Health(ctx); err != nil {
		components["fetcher"] = fmt.Sprintf("unhealthy: %v", err)
	} else {
		components["fetcher"] = "healthy"
	}

	if err := p.queue.Health(ctx); err != nil {
		components["queue"] = fmt.Sprintf("unhealthy: %v", err)
	} else {
		components["queue"] = "healthy"
	}

	if err := p.workerPool.Health(ctx); err != nil {
		components["worker_pool"] = fmt.Sprintf("unhealthy: %v", err)
	} else {
		components["worker_pool"] = "healthy"
	}

	if err := p.searchFactory.Health(ctx); err != nil {
		components["search_factory"] = fmt.Sprintf("unhealthy: %v", err)
	} else {
		components["search_factory"] = "healthy"
	}

	// Определяем общий статус
	overallStatus := "healthy"
	for _, status := range components {
		if status != "healthy" {
			overallStatus = "degraded"
			break
		}
	}

	return &ports.PipelineHealth{
		Status:     overallStatus,
		Timestamp:  time.Now(),
		Components: components,
	}, nil
}

// GetMetrics возвращает метрики pipeline
func (p *EmailPipelineImpl) GetMetrics(ctx context.Context) (*ports.PipelineMetrics, error) {
	queueSize, _ := p.queue.Size(ctx)
	workerMetrics := p.workerPool.GetMetrics(ctx)

	return &ports.PipelineMetrics{
		ProviderType:     p.fetcher.GetProviderType(),
		Uptime:           time.Since(p.metrics.startTime),
		TotalProcessed:   p.metrics.getTotalProcessed(),
		TotalFailed:      p.metrics.getTotalFailed(),
		CurrentQueueSize: queueSize,
		WorkersActive:    workerMetrics.WorkersActive,
		WorkersTotal:     workerMetrics.WorkersTotal,
		LastProcessed:    p.metrics.lastProcessed,
		AvgProcessTime:   p.metrics.getAvgProcessTime(),
	}, nil
}

// createFetchCriteria создает критерии для выборки сообщений
func (p *EmailPipelineImpl) createFetchCriteria() ports.FetchCriteria {
	return p.criteriaBuilder.BuildStandardCriteria(context.Background())
}

// processMessages обрабатывает сообщения через worker pool
func (p *EmailPipelineImpl) processMessages(ctx context.Context, messages []domain.EmailMessage) (int, int) {
	var wg sync.WaitGroup
	processed := 0
	failed := 0
	var mu sync.Mutex

	for i, msg := range messages {
		select {
		case <-ctx.Done():
			p.logger.Warn(ctx, "Message processing cancelled by context",
				"processed", processed, "failed", failed, "remaining", len(messages)-i)
			return processed, failed
		default:
			wg.Add(1)
			go func(msg domain.EmailMessage) {
				defer wg.Done()

				if err := p.workerPool.Submit(ctx, msg); err != nil {
					p.logger.Error(ctx, "Failed to submit message to worker pool",
						"message_id", msg.MessageID, "error", err.Error())
					mu.Lock()
					failed++
					mu.Unlock()
				} else {
					mu.Lock()
					processed++
					mu.Unlock()
				}
			}(msg)
		}
	}

	wg.Wait()
	return processed, failed
}

// healthCheck проверяет здоровье всех компонентов
func (p *EmailPipelineImpl) healthCheck(ctx context.Context) error {
	components := []struct {
		name   string
		health func(ctx context.Context) error
	}{
		{"fetcher", p.fetcher.Health},
		{"queue", p.queue.Health},
		{"worker_pool", p.workerPool.Health},
		{"search_factory", p.searchFactory.Health},
	}

	for _, component := range components {
		if err := component.health(ctx); err != nil {
			return fmt.Errorf("%s health check failed: %w", component.name, err)
		}
	}

	return nil
}

// getQueueSize возвращает размер очереди
func (p *EmailPipelineImpl) getQueueSize(ctx context.Context) int {
	size, err := p.queue.Size(ctx)
	if err != nil {
		p.logger.Warn(ctx, "Failed to get queue size", "error", err.Error())
		return 0
	}
	return size
}

// Методы для PipelineMetricsCollector
func (m *PipelineMetricsCollector) recordProcessing(processed, failed int, duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.totalProcessed += int64(processed)
	m.totalFailed += int64(failed)
	m.processingTime += duration
	m.lastProcessed = time.Now()
}

func (m *PipelineMetricsCollector) recordFailure() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.totalFailed++
}

func (m *PipelineMetricsCollector) getTotalProcessed() int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.totalProcessed
}

func (m *PipelineMetricsCollector) getTotalFailed() int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.totalFailed
}

func (m *PipelineMetricsCollector) getAvgProcessTime() time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.totalProcessed == 0 {
		return 0
	}
	return m.processingTime / time.Duration(m.totalProcessed)
}

// Методы для PipelineStatus
func (s *PipelineStatus) setStatus(status, phase string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.status = status
	s.currentPhase = phase
}

func (s *PipelineStatus) getStatus() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.status
}

func (s *PipelineStatus) setCurrentPhase(phase string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.currentPhase = phase
}

func (s *PipelineStatus) setLastError(error string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastError = error
}
