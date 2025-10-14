package ports

import (
	"context"
)

// HealthChecker общий интерфейс для проверки здоровья сервисов
type HealthChecker interface {
	HealthCheck(ctx context.Context) error
}

// Logger интерфейс для логирования
type Logger interface {
	Debug(ctx context.Context, msg string, fields ...interface{})
	Info(ctx context.Context, msg string, fields ...interface{})
	Warn(ctx context.Context, msg string, fields ...interface{})
	Error(ctx context.Context, msg string, fields ...interface{})
}

// IDGenerator для генерации идентификаторов
type IDGenerator interface {
	GenerateID() string
	GenerateMessageID() string
}
