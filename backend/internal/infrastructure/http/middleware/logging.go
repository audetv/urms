// internal/infrastructure/http/middleware/logging.go
package middleware

import (
	"context"
	"fmt"
	"time"

	"github.com/audetv/urms/internal/core/ports"
	"github.com/gin-gonic/gin"
)

// generateCorrelationID генерирует уникальный ID для корреляции запросов
func generateCorrelationID() string {
	return fmt.Sprintf("req-%d", time.Now().UnixNano())
}

// LoggingMiddleware добавляет структурированное логирование для HTTP запросов
func LoggingMiddleware(logger ports.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Создаем контекст с correlation ID
		ctx := c.Request.Context()
		correlationID := c.GetHeader("X-Correlation-ID")
		if correlationID == "" {
			correlationID = generateCorrelationID()
		}

		ctx = context.WithValue(ctx, ports.CorrelationIDKey, correlationID)
		c.Request = c.Request.WithContext(ctx)

		// Логируем начало запроса
		logger.Info(ctx, "HTTP request started",
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"remote_addr", c.Request.RemoteAddr,
			"user_agent", c.Request.UserAgent(),
			"correlation_id", correlationID,
		)

		// Продолжаем обработку
		c.Next()

		// Логируем завершение запроса
		latency := time.Since(start)
		status := c.Writer.Status()

		// УБИРАЕМ неиспользуемую переменную logLevel
		// Просто логируем с уровнем info, а в логгере можно фильтровать по статусу
		// т.к. интерфейс ports.Logger не поддерживает динамический уровень логирования через строку.
		logger.Info(ctx, "HTTP request completed",
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", status,
			"latency", latency.String(),
			"bytes_out", c.Writer.Size(),
			"correlation_id", correlationID,
		)
	}
}
