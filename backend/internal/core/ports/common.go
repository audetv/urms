package ports

import (
	"context"
	"time"

	"github.com/audetv/urms/internal/core/domain" // Добавляем импорт domain
)

// HealthChecker общий интерфейс для проверки здоровья сервисов
// type HealthChecker interface {
// 	HealthCheck(ctx context.Context) error
// }

// Logger интерфейс для логирования
type Logger interface {
	Debug(ctx context.Context, msg string, fields ...interface{})
	Info(ctx context.Context, msg string, fields ...interface{})
	Warn(ctx context.Context, msg string, fields ...interface{})
	Error(ctx context.Context, msg string, fields ...interface{})
}

// DomainIDGenerator адаптер для доменного IDGenerator
// Реализует domain.IDGenerator из domain слоя
type DomainIDGenerator interface {
	domain.IDGenerator
}

// ConfigProvider общий интерфейс для работы с конфигурацией
type ConfigProvider interface {
	GetString(key string) string
	GetInt(key string) int
	GetBool(key string) bool
	GetDuration(key string) time.Duration
}
