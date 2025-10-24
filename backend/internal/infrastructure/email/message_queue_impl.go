// internal/infrastructure/email/message_queue_impl.go
package email

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/audetv/urms/internal/core/domain"
	"github.com/audetv/urms/internal/core/ports"
)

// MessageQueueImpl реализует ports.MessageQueue
type MessageQueueImpl struct {
	messages []domain.EmailMessage
	maxSize  int
	mu       sync.RWMutex
	cond     *sync.Cond
	logger   ports.Logger
	metrics  *QueueMetrics
	isClosed bool
}

// QueueMetrics собирает метрики очереди
type QueueMetrics struct {
	totalEnqueued   int64
	totalDequeued   int64
	maxQueueSize    int
	currentSize     int
	enqueueWaitTime time.Duration
	dequeueWaitTime time.Duration
	mu              sync.RWMutex
}

// NewMessageQueueImpl создает новую in-memory очередь сообщений
func NewMessageQueueImpl(maxSize int, logger ports.Logger) *MessageQueueImpl {
	queue := &MessageQueueImpl{
		messages: make([]domain.EmailMessage, 0, maxSize),
		maxSize:  maxSize,
		logger:   logger,
		metrics:  &QueueMetrics{},
		isClosed: false,
	}
	queue.cond = sync.NewCond(&queue.mu)
	return queue
}

// Enqueue добавляет сообщения в очередь
func (q *MessageQueueImpl) Enqueue(ctx context.Context, messages []domain.EmailMessage) error {
	if len(messages) == 0 {
		return nil // Нет сообщений для добавления
	}

	q.mu.Lock()
	defer q.mu.Unlock()

	// Проверяем не закрыта ли очередь
	if q.isClosed {
		return fmt.Errorf("message queue is closed")
	}

	startTime := time.Now()

	// Проверяем достаточно ли места в очереди
	availableSpace := q.maxSize - len(q.messages)
	if availableSpace < len(messages) {
		// Очередь переполнена - ждем места или таймаута
		if err := q.waitForSpace(ctx, len(messages)); err != nil {
			return fmt.Errorf("failed to wait for queue space: %w", err)
		}
	}

	// Добавляем сообщения в очередь
	for _, msg := range messages {
		q.messages = append(q.messages, msg)
		q.metrics.recordEnqueue()
	}

	enqueueTime := time.Since(startTime)
	q.metrics.recordEnqueueWaitTime(enqueueTime)

	// Уведомляем ожидающих потребителей
	q.cond.Broadcast()

	q.logger.Debug(ctx, "✅ Messages enqueued successfully",
		"message_count", len(messages),
		"queue_size_after", len(q.messages),
		"max_queue_size", q.maxSize,
		"enqueue_time", enqueueTime.String())

	return nil
}

// Dequeue извлекает сообщения из очереди
func (q *MessageQueueImpl) Dequeue(ctx context.Context, batchSize int) ([]domain.EmailMessage, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	startTime := time.Now()

	// Ждем сообщений или таймаута
	if len(q.messages) == 0 {
		if err := q.waitForMessages(ctx); err != nil {
			return nil, fmt.Errorf("failed to wait for messages: %w", err)
		}
	}

	// Определяем сколько сообщений извлечь
	available := len(q.messages)
	toDequeue := min(batchSize, available)

	// Извлекаем сообщения
	messages := make([]domain.EmailMessage, toDequeue)
	copy(messages, q.messages[:toDequeue])
	q.messages = q.messages[toDequeue:]

	// Обновляем метрики
	for range messages {
		q.metrics.recordDequeue()
	}

	dequeueTime := time.Since(startTime)
	q.metrics.recordDequeueWaitTime(dequeueTime)

	q.logger.Debug(ctx, "📤 Messages dequeued successfully",
		"message_count", len(messages),
		"queue_size_after", len(q.messages),
		"batch_size_requested", batchSize,
		"dequeue_time", dequeueTime.String())

	return messages, nil
}

// Size возвращает текущий размер очереди
func (q *MessageQueueImpl) Size(ctx context.Context) (int, error) {
	q.mu.RLock()
	defer q.mu.RUnlock()

	size := len(q.messages)
	q.logger.Debug(ctx, "Queue size queried",
		"current_size", size,
		"max_size", q.maxSize)

	return size, nil
}

// Health проверяет здоровье очереди
func (q *MessageQueueImpl) Health(ctx context.Context) error {
	q.mu.RLock()
	defer q.mu.RUnlock()

	if q.isClosed {
		return fmt.Errorf("message queue is closed")
	}

	// Проверяем не переполнена ли очередь
	utilization := float64(len(q.messages)) / float64(q.maxSize) * 100
	if utilization > 90 {
		q.logger.Warn(ctx, "Queue utilization is high",
			"utilization_percent", fmt.Sprintf("%.1f%%", utilization),
			"current_size", len(q.messages),
			"max_size", q.maxSize)
	}

	q.logger.Debug(ctx, "Queue health check passed",
		"current_size", len(q.messages),
		"max_size", q.maxSize,
		"utilization_percent", fmt.Sprintf("%.1f%%", utilization))

	return nil
}

// Clear очищает очередь
func (q *MessageQueueImpl) Clear(ctx context.Context) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	clearedCount := len(q.messages)
	q.messages = make([]domain.EmailMessage, 0, q.maxSize)

	q.logger.Info(ctx, "🗑️ Queue cleared",
		"cleared_messages", clearedCount,
		"operation", "manual_clear")

	return nil
}

// Close закрывает очередь для новых операций
func (q *MessageQueueImpl) Close(ctx context.Context) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.isClosed = true
	q.cond.Broadcast() // Разбудить всех ожидающих

	q.logger.Info(ctx, "🔒 Message queue closed",
		"final_size", len(q.messages))

	return nil
}

// GetMetrics возвращает метрики очереди
func (q *MessageQueueImpl) GetMetrics(ctx context.Context) *ports.QueueMetrics {
	q.metrics.mu.RLock()
	defer q.metrics.mu.RUnlock()

	currentSize := len(q.messages)

	return &ports.QueueMetrics{
		TotalEnqueued:   q.metrics.totalEnqueued,
		TotalDequeued:   q.metrics.totalDequeued,
		MaxQueueSize:    q.metrics.maxQueueSize,
		CurrentSize:     currentSize,
		EnqueueWaitTime: q.metrics.enqueueWaitTime,
		DequeueWaitTime: q.metrics.dequeueWaitTime,
	}
}

// waitForSpace ждет пока в очереди появится достаточно места
func (q *MessageQueueImpl) waitForSpace(ctx context.Context, requiredSpace int) error {
	for len(q.messages)+requiredSpace > q.maxSize {
		// Проверяем контекст
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Ждем пока потребители освободят место
			q.cond.Wait()

			// Проверяем не закрыта ли очередь
			if q.isClosed {
				return fmt.Errorf("queue closed while waiting for space")
			}
		}
	}
	return nil
}

// waitForMessages ждет пока в очереди появятся сообщения
func (q *MessageQueueImpl) waitForMessages(ctx context.Context) error {
	for len(q.messages) == 0 {
		// Проверяем контекст
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Ждем пока производители добавят сообщения
			q.cond.Wait()

			// Проверяем не закрыта ли очередь
			if q.isClosed {
				return fmt.Errorf("queue closed while waiting for messages")
			}
		}
	}
	return nil
}

// Методы для QueueMetrics
func (m *QueueMetrics) recordEnqueue() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.totalEnqueued++
	m.currentSize++
	if m.currentSize > m.maxQueueSize {
		m.maxQueueSize = m.currentSize
	}
}

func (m *QueueMetrics) recordDequeue() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.totalDequeued++
	m.currentSize--
}

func (m *QueueMetrics) recordEnqueueWaitTime(duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.enqueueWaitTime = (m.enqueueWaitTime + duration) / 2 // Moving average
}

func (m *QueueMetrics) recordDequeueWaitTime(duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.dequeueWaitTime = (m.dequeueWaitTime + duration) / 2 // Moving average
}

// Вспомогательная функция
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
