package ports

import (
	"context"
	"time"

	"github.com/audetv/urms/internal/core/domain"
)

// Logger определяет контракт для системы логирования
type Logger interface {
	Debug(ctx context.Context, msg string, fields ...interface{})
	Info(ctx context.Context, msg string, fields ...interface{})
	Warn(ctx context.Context, msg string, fields ...interface{})
	Error(ctx context.Context, msg string, fields ...interface{})
	WithContext(ctx context.Context) context.Context
}

// CorrelationKeyType тип для ключа correlation ID в context
type CorrelationKeyType string

const (
	// CorrelationIDKey ключ для хранения correlation ID в context
	CorrelationIDKey CorrelationKeyType = "correlation_id"
)

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
