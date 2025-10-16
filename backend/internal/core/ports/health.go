// backend/internal/core/ports/health.go
package ports

import "context"

// HealthChecker определяет контракт для проверки здоровья сервисов
type HealthChecker interface {
	// CheckHealth выполняет проверку здоровья компонента
	CheckHealth(ctx context.Context) *HealthStatus

	// GetName возвращает имя компонента для проверки
	GetName() string
}

// HealthStatus представляет статус здоровья компонента
type HealthStatus struct {
	Name      string                 `json:"name"`
	Status    HealthStatusValue      `json:"status"`
	Message   string                 `json:"message,omitempty"`
	Details   map[string]interface{} `json:"details,omitempty"`
	Timestamp string                 `json:"timestamp"`
}

// HealthStatusValue представляет возможные статусы здоровья
type HealthStatusValue string

const (
	HealthStatusUp       HealthStatusValue = "UP"
	HealthStatusDown     HealthStatusValue = "DOWN"
	HealthStatusDegraded HealthStatusValue = "DEGRADED"
	HealthStatusUnknown  HealthStatusValue = "UNKNOWN"
)

// HealthAggregator агрегирует статусы всех компонентов
type HealthAggregator interface {
	// CheckAll выполняет проверку всех зарегистрированных компонентов
	CheckAll(ctx context.Context) map[string]*HealthStatus

	// Register регистрирует новый компонент для проверки
	Register(checker HealthChecker)

	// GetOverallStatus возвращает общий статус системы
	GetOverallStatus(ctx context.Context) HealthStatusValue
}
