// internal/infrastructure/http/middleware/context.go
package middleware

import (
	"context"

	"github.com/gin-gonic/gin"
)

// ContextMiddleware добавляет дополнительные поля в контекст
func ContextMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Добавляем базовую информацию в контекст
		ctx := c.Request.Context()

		// Можно добавить аутентификацию пользователя и т.д.
		// Пока используем "system" как дефолтного пользователя
		ctx = context.WithValue(ctx, "user_id", "system")
		ctx = context.WithValue(ctx, "user_role", "system")

		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}
