// internal/infrastructure/email/worker_pool_impl.go
package email

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/audetv/urms/internal/core/domain"
	"github.com/audetv/urms/internal/core/ports"
)

// WorkerPoolImpl реализует ports.WorkerPool
type WorkerPoolImpl struct {
	workers          []*Worker
	jobQueue         chan domain.EmailMessage
	workerCount      int
	messageProcessor ports.MessageProcessor
	logger           ports.Logger
	metrics          *WorkerPoolMetrics
	shutdownChan     chan struct{}
	wg               sync.WaitGroup
	mu               sync.RWMutex
	isRunning        bool
}

// Worker представляет отдельного воркера
type Worker struct {
	id        int
	jobQueue  <-chan domain.EmailMessage
	processor ports.MessageProcessor
	logger    ports.Logger
	metrics   *WorkerPoolMetrics
	shutdown  <-chan struct{}
	wg        *sync.WaitGroup
}

// WorkerPoolMetrics собирает метрики worker pool
type WorkerPoolMetrics struct {
	totalProcessed      int64
	totalFailed         int64
	activeWorkers       int32
	idleWorkers         int32
	totalProcessingTime time.Duration
	lastActivity        time.Time
	mu                  sync.RWMutex
}

// NewWorkerPoolImpl создает новый worker pool
func NewWorkerPoolImpl(
	workerCount int,
	queueSize int,
	messageProcessor ports.MessageProcessor,
	logger ports.Logger,
) *WorkerPoolImpl {

	return &WorkerPoolImpl{
		workers:          make([]*Worker, 0, workerCount),
		jobQueue:         make(chan domain.EmailMessage, queueSize),
		workerCount:      workerCount,
		messageProcessor: messageProcessor,
		logger:           logger,
		metrics:          &WorkerPoolMetrics{},
		shutdownChan:     make(chan struct{}),
		isRunning:        false,
	}
}

// Start запускает worker pool
func (wp *WorkerPoolImpl) Start(ctx context.Context) error {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	if wp.isRunning {
		return fmt.Errorf("worker pool is already running")
	}

	wp.logger.Info(ctx, "🚀 Starting worker pool",
		"worker_count", wp.workerCount,
		"queue_size", cap(wp.jobQueue))

	// Создаем воркеры
	for i := 0; i < wp.workerCount; i++ {
		worker := &Worker{
			id:        i + 1,
			jobQueue:  wp.jobQueue,
			processor: wp.messageProcessor,
			logger:    wp.logger,
			metrics:   wp.metrics,
			shutdown:  wp.shutdownChan,
			wg:        &wp.wg,
		}
		wp.workers = append(wp.workers, worker)
		wp.wg.Add(1)
		go worker.start(ctx)
	}

	wp.isRunning = true
	atomic.StoreInt32(&wp.metrics.idleWorkers, int32(wp.workerCount))

	wp.logger.Info(ctx, "✅ Worker pool started successfully",
		"total_workers", len(wp.workers),
		"queue_capacity", cap(wp.jobQueue))

	return nil
}

// Stop останавливает worker pool
func (wp *WorkerPoolImpl) Stop(ctx context.Context) error {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	if !wp.isRunning {
		return nil // Уже остановлен
	}

	wp.logger.Info(ctx, "🛑 Stopping worker pool")

	// Закрываем канал shutdown чтобы сигнализировать воркерам
	close(wp.shutdownChan)

	// Ждем завершения воркеров
	done := make(chan struct{})
	go func() {
		wp.wg.Wait()
		close(done)
	}()

	// Ждем с таймаутом
	select {
	case <-done:
		wp.logger.Info(ctx, "✅ Worker pool stopped gracefully")
	case <-ctx.Done():
		wp.logger.Warn(ctx, "Worker pool stop timed out")
		return ctx.Err()
	}

	// Закрываем job queue
	close(wp.jobQueue)
	wp.isRunning = false

	wp.logger.Info(ctx, "🎯 Worker pool shutdown completed",
		"total_processed", wp.metrics.getTotalProcessed(),
		"total_failed", wp.metrics.getTotalFailed(),
		"avg_process_time", wp.metrics.getAvgProcessTime())

	return nil
}

// Submit отправляет сообщение на обработку в worker pool
func (wp *WorkerPoolImpl) Submit(ctx context.Context, message domain.EmailMessage) error {
	if !wp.isRunning {
		return fmt.Errorf("worker pool is not running")
	}

	select {
	case wp.jobQueue <- message:
		wp.logger.Debug(ctx, "Message submitted to worker pool",
			"message_id", message.MessageID,
			"queue_size", len(wp.jobQueue),
			"queue_capacity", cap(wp.jobQueue))
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-wp.shutdownChan:
		return fmt.Errorf("worker pool is shutting down")
	default:
		// Очередь заполнена
		wp.logger.Warn(ctx, "Worker pool queue is full, message rejected",
			"message_id", message.MessageID,
			"queue_size", len(wp.jobQueue))
		return fmt.Errorf("worker pool queue is full")
	}
}

// GetMetrics возвращает метрики worker pool
func (wp *WorkerPoolImpl) GetMetrics(ctx context.Context) *ports.WorkerMetrics {
	wp.metrics.mu.RLock()
	defer wp.metrics.mu.RUnlock()

	activeWorkers := atomic.LoadInt32(&wp.metrics.activeWorkers)
	idleWorkers := atomic.LoadInt32(&wp.metrics.idleWorkers)

	return &ports.WorkerMetrics{
		WorkersActive:  int(activeWorkers),
		WorkersIdle:    int(idleWorkers),
		WorkersTotal:   wp.workerCount,
		TasksProcessed: wp.metrics.getTotalProcessed(),
		TasksFailed:    wp.metrics.getTotalFailed(),
		AvgProcessTime: wp.metrics.getAvgProcessTime(),
		QueueWaitTime:  wp.calculateQueueWaitTime(),
		LastActivity:   wp.metrics.lastActivity,
	}
}

// Health проверяет здоровье worker pool
func (wp *WorkerPoolImpl) Health(ctx context.Context) error {
	wp.mu.RLock()
	defer wp.mu.RUnlock()

	if !wp.isRunning {
		return fmt.Errorf("worker pool is not running")
	}

	// Проверяем что есть активные воркеры
	activeWorkers := atomic.LoadInt32(&wp.metrics.activeWorkers)
	if activeWorkers == 0 && wp.metrics.getTotalProcessed() > 0 {
		return fmt.Errorf("no active workers but pool has processed messages")
	}

	// Проверяем не переполнена ли очередь
	queueUtilization := float64(len(wp.jobQueue)) / float64(cap(wp.jobQueue)) * 100
	if queueUtilization > 80 {
		wp.logger.Warn(ctx, "Worker pool queue utilization is high",
			"utilization_percent", fmt.Sprintf("%.1f%%", queueUtilization),
			"queue_size", len(wp.jobQueue),
			"queue_capacity", cap(wp.jobQueue))
	}

	wp.logger.Debug(ctx, "Worker pool health check passed",
		"active_workers", activeWorkers,
		"idle_workers", atomic.LoadInt32(&wp.metrics.idleWorkers),
		"queue_utilization", fmt.Sprintf("%.1f%%", queueUtilization))

	return nil
}

// calculateQueueWaitTime вычисляет среднее время ожидания в очереди
func (wp *WorkerPoolImpl) calculateQueueWaitTime() time.Duration {
	// Простая эвристика - можно сделать сложнее с реальными метриками
	queueSize := len(wp.jobQueue)
	if queueSize == 0 {
		return 0
	}

	// Предполагаем что каждый воркер обрабатывает 1 сообщение в секунду
	estimatedWait := time.Duration(queueSize) * time.Second / time.Duration(wp.workerCount)
	return estimatedWait
}

// start запускает воркера
func (w *Worker) start(ctx context.Context) {
	defer w.wg.Done()

	w.logger.Info(ctx, "👷 Worker started",
		"worker_id", w.id)

	for {
		select {
		case message, ok := <-w.jobQueue:
			if !ok {
				// Канал закрыт
				w.logger.Debug(ctx, "Worker stopping - job queue closed",
					"worker_id", w.id)
				return
			}
			w.processMessage(ctx, message)

		case <-w.shutdown:
			w.logger.Debug(ctx, "Worker stopping - shutdown signal",
				"worker_id", w.id)
			return

		case <-ctx.Done():
			w.logger.Debug(ctx, "Worker stopping - context cancelled",
				"worker_id", w.id)
			return
		}
	}
}

// processMessage обрабатывает сообщение
func (w *Worker) processMessage(ctx context.Context, message domain.EmailMessage) {
	startTime := time.Now()

	// Обновляем метрики
	atomic.AddInt32(&w.metrics.activeWorkers, 1)
	atomic.AddInt32(&w.metrics.idleWorkers, -1)
	defer func() {
		atomic.AddInt32(&w.metrics.activeWorkers, -1)
		atomic.AddInt32(&w.metrics.idleWorkers, 1)
	}()

	w.logger.Debug(ctx, "Worker processing message",
		"worker_id", w.id,
		"message_id", message.MessageID,
		"subject", message.Subject)

	// Обрабатываем сообщение
	err := w.processor.ProcessIncomingEmail(ctx, message)
	processingTime := time.Since(startTime)

	// Обновляем метрики
	w.metrics.recordProcessing(processingTime, err == nil)

	if err != nil {
		w.logger.Error(ctx, "Worker failed to process message",
			"worker_id", w.id,
			"message_id", message.MessageID,
			"error", err.Error(),
			"processing_time", processingTime.String())
	} else {
		w.logger.Debug(ctx, "Worker successfully processed message",
			"worker_id", w.id,
			"message_id", message.MessageID,
			"processing_time", processingTime.String())
	}
}

// Методы для WorkerPoolMetrics
func (m *WorkerPoolMetrics) recordProcessing(duration time.Duration, success bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if success {
		m.totalProcessed++
	} else {
		m.totalFailed++
	}
	m.totalProcessingTime += duration
	m.lastActivity = time.Now()
}

func (m *WorkerPoolMetrics) getTotalProcessed() int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.totalProcessed
}

func (m *WorkerPoolMetrics) getTotalFailed() int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.totalFailed
}

func (m *WorkerPoolMetrics) getAvgProcessTime() time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.totalProcessed == 0 {
		return 0
	}
	return m.totalProcessingTime / time.Duration(m.totalProcessed)
}
