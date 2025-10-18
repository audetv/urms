// backend/internal/infrastructure/email/retry_manager.go
package email

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/audetv/urms/internal/core/ports"
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
	logger ports.Logger // ✅ ДОБАВЛЯЕМ logger
}

// NewRetryManager создает новый менеджер повторных попыток
func NewRetryManager(config RetryConfig, logger ports.Logger) *RetryManager {
	return &RetryManager{
		config: config,
		logger: logger,
	}
}

// ExecuteWithRetry выполняет операцию с повторными попытками при временных ошибках
func (m *RetryManager) ExecuteWithRetry(ctx context.Context, operation string, fn func() error) error {
	var lastErr error

	for attempt := 1; attempt <= m.config.MaxAttempts; attempt++ {
		m.logger.Info(ctx, "Retry attempt",
			"operation", operation,
			"attempt", attempt,
			"max_attempts", m.config.MaxAttempts)

		err := fn()
		if err == nil {
			if attempt > 1 {
				m.logger.Info(ctx, "Operation succeeded after retry",
					"operation", operation,
					"attempt", attempt)
			}
			return nil
		}

		lastErr = err

		// Проверяем, является ли ошибка временной
		if retryableErr, ok := err.(interface{ IsRetryable() bool }); ok {
			if !retryableErr.IsRetryable() {
				m.logger.Error(ctx, "Operation failed with permanent error",
					"operation", operation,
					"error", err.Error())
				return err
			}
		}

		// Если это последняя попытка, выходим
		if attempt == m.config.MaxAttempts {
			m.logger.Error(ctx, "Operation failed after all attempts",
				"operation", operation,
				"max_attempts", m.config.MaxAttempts,
				"error", err.Error())
			return fmt.Errorf("operation failed after %d attempts: %w", m.config.MaxAttempts, err)
		}

		// Вычисляем задержку для следующей попытки
		delay := m.calculateDelay(attempt)
		m.logger.Warn(ctx, "Operation failed, retrying after delay",
			"operation", operation,
			"attempt", attempt,
			"delay", delay.String(),
			"error", err.Error())

		// Ждем перед следующей попыткой
		select {
		case <-time.After(delay):
			// Продолжаем
		case <-ctx.Done():
			m.logger.Warn(ctx, "Operation cancelled during retry",
				"operation", operation,
				"error", ctx.Err().Error())
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
