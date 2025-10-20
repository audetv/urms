// internal/infrastructure/http/middleware/error_handler.go
package middleware

import (
	"github.com/audetv/urms/internal/core/ports"
	"github.com/audetv/urms/internal/infrastructure/http/dto"
	"github.com/gin-gonic/gin"
)

// ErrorHandlerMiddleware централизованно обрабатывает ошибки
func ErrorHandlerMiddleware(logger ports.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next() // Сначала выполняем обработчики

		// Проверяем есть ли ошибки
		if len(c.Errors) > 0 {
			ctx := c.Request.Context()

			// Берем последнюю ошибку
			err := c.Errors.Last().Err

			logger.Error(ctx, "HTTP request error",
				"error", err.Error(),
				"method", c.Request.Method,
				"path", c.Request.URL.Path,
				"status", c.Writer.Status(),
			)

			// Возвращаем структурированную ошибку
			c.JSON(c.Writer.Status(), dto.NewErrorResponse(
				"REQUEST_ERROR",
				"Ошибка обработки запроса",
				err.Error(),
			))
		}
	}
}
