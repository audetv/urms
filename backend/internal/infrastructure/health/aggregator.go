// backend/internal/infrastructure/health/aggregator.go
package health

import (
	"context"
	"sync"

	"github.com/audetv/urms/internal/core/ports"
)

// HealthAggregatorImpl реализует HealthAggregator
type HealthAggregatorImpl struct {
	checkers []ports.HealthChecker
	mu       sync.RWMutex
}

// NewHealthAggregator создает новый агрегатор health checks
func NewHealthAggregator() *HealthAggregatorImpl {
	return &HealthAggregatorImpl{
		checkers: make([]ports.HealthChecker, 0),
	}
}

// Register регистрирует новый компонент для проверки
func (a *HealthAggregatorImpl) Register(checker ports.HealthChecker) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.checkers = append(a.checkers, checker)
}

// CheckAll выполняет проверку всех зарегистрированных компонентов
func (a *HealthAggregatorImpl) CheckAll(ctx context.Context) map[string]*ports.HealthStatus {
	a.mu.RLock()
	defer a.mu.RUnlock()

	results := make(map[string]*ports.HealthStatus)

	for _, checker := range a.checkers {
		results[checker.GetName()] = checker.CheckHealth(ctx)
	}

	return results
}

// GetOverallStatus возвращает общий статус системы
func (a *HealthAggregatorImpl) GetOverallStatus(ctx context.Context) ports.HealthStatusValue {
	statuses := a.CheckAll(ctx)

	hasDown := false
	hasDegraded := false

	for _, status := range statuses {
		switch status.Status {
		case ports.HealthStatusDown:
			hasDown = true
		case ports.HealthStatusDegraded:
			hasDegraded = true
		}
	}

	if hasDown {
		return ports.HealthStatusDown
	}
	if hasDegraded {
		return ports.HealthStatusDegraded
	}

	return ports.HealthStatusUp
}
