// backend/internal/infrastructure/email/retry_manager.go
package email

import (
	"context"
	"fmt"
	"log"
	"math"
	"time"
)

// RetryConfig конфигурация retry механизма
type RetryConfig struct {
	MaxAttempts   int
	BaseDelay     time.Duration
	MaxDelay      time.Duration
	BackoffFactor float64
}

// DefaultRetryConfig возвращает конфигурацию по умолчанию
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts:   3,
		BaseDelay:     2 * time.Second,
		MaxDelay:      30 * time.Second,
		BackoffFactor: 1.5,
	}
}

// RetryManager управляет повторными попытками выполнения операций
type RetryManager struct {
	config RetryConfig
}

// NewRetryManager создает новый менеджер повторных попыток
func NewRetryManager(config RetryConfig) *RetryManager {
	return &RetryManager{
		config: config,
	}
}

// ExecuteWithRetry выполняет операцию с повторными попытками при временных ошибках
func (m *RetryManager) ExecuteWithRetry(ctx context.Context, operation string, fn func() error) error {
	var lastErr error

	for attempt := 1; attempt <= m.config.MaxAttempts; attempt++ {
		log.Printf("🔄 %s attempt %d/%d", operation, attempt, m.config.MaxAttempts)

		err := fn()
		if err == nil {
			if attempt > 1 {
				log.Printf("✅ %s succeeded on attempt %d", operation, attempt)
			}
			return nil
		}

		lastErr = err

		// Проверяем, является ли ошибка временной
		if retryableErr, ok := err.(interface{ IsRetryable() bool }); ok {
			if !retryableErr.IsRetryable() {
				log.Printf("❌ %s failed with permanent error: %v", operation, err)
				return err
			}
		}

		// Если это последняя попытка, выходим
		if attempt == m.config.MaxAttempts {
			log.Printf("❌ %s failed after %d attempts: %v", operation, m.config.MaxAttempts, err)
			return fmt.Errorf("operation failed after %d attempts: %w", m.config.MaxAttempts, err)
		}

		// Вычисляем задержку для следующей попытки
		delay := m.calculateDelay(attempt)
		log.Printf("⏳ %s failed, retrying in %v: %v", operation, delay, err)

		// Ждем перед следующей попыткой
		select {
		case <-time.After(delay):
			// Продолжаем
		case <-ctx.Done():
			return fmt.Errorf("operation cancelled: %w", ctx.Err())
		}
	}

	return lastErr
}

// calculateDelay вычисляет задержку для попытки
func (m *RetryManager) calculateDelay(attempt int) time.Duration {
	delay := time.Duration(float64(m.config.BaseDelay) * math.Pow(m.config.BackoffFactor, float64(attempt-1)))
	if delay > m.config.MaxDelay {
		return m.config.MaxDelay
	}
	return delay
}
