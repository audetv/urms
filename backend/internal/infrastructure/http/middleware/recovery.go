// internal/infrastructure/http/middleware/recovery.go
package middleware

import (
	"net/http"

	"github.com/audetv/urms/internal/core/ports"
	"github.com/audetv/urms/internal/infrastructure/http/dto"
	"github.com/gin-gonic/gin"
)

// RecoveryMiddleware обрабатывает паники и возвращает структурированные ошибки
func RecoveryMiddleware(logger ports.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				ctx := c.Request.Context()

				logger.Error(ctx, "HTTP request panic recovered",
					"panic", err,
					"method", c.Request.Method,
					"path", c.Request.URL.Path,
				)

				c.JSON(http.StatusInternalServerError, dto.NewErrorResponse(
					"INTERNAL_SERVER_ERROR",
					"Внутренняя ошибка сервера",
					"",
				))

				c.Abort()
			}
		}()

		c.Next()
	}
}
