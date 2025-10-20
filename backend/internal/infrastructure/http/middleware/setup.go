// internal/infrastructure/http/middleware/setup.go
package middleware

import (
	"github.com/audetv/urms/internal/core/ports"
	"github.com/gin-gonic/gin"
)

// SetupMiddleware настраивает все middleware для приложения
func SetupMiddleware(router *gin.Engine, logger ports.Logger) {
	// Global middleware
	router.Use(gin.Recovery())                 // Базовая защита от паник Gin
	router.Use(RecoveryMiddleware(logger))     // Наша улучшенная обработка паник
	router.Use(LoggingMiddleware(logger))      // Структурированное логирование
	router.Use(ErrorHandlerMiddleware(logger)) // Обработка ошибок
	router.Use(CORSMiddleware())               // CORS
	router.Use(ContextMiddleware())            // Дополнительный контекст
}
