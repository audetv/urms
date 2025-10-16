// backend/internal/infrastructure/http/health_handler.go
package http

import (
	"encoding/json"
	"net/http"

	"github.com/audetv/urms/internal/core/ports"
)

// HealthHandler обрабатывает HTTP запросы для health checks
type HealthHandler struct {
	aggregator ports.HealthAggregator
}

// NewHealthHandler создает новый HTTP handler для health checks
func NewHealthHandler(aggregator ports.HealthAggregator) *HealthHandler {
	return &HealthHandler{
		aggregator: aggregator,
	}
}

// HealthCheckHandler возвращает статус здоровья системы
func (h *HealthHandler) HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	overallStatus := h.aggregator.GetOverallStatus(ctx)
	detailedStatus := h.aggregator.CheckAll(ctx)

	response := map[string]interface{}{
		"status":    overallStatus,
		"timestamp": detailedStatus,
		"services":  detailedStatus,
	}

	w.Header().Set("Content-Type", "application/json")

	// Устанавливаем соответствующий HTTP статус код
	switch overallStatus {
	case ports.HealthStatusUp:
		w.WriteHeader(http.StatusOK)
	case ports.HealthStatusDegraded:
		w.WriteHeader(http.StatusOK) // 200 но с degraded статусом
	case ports.HealthStatusDown:
		w.WriteHeader(http.StatusServiceUnavailable)
	default:
		w.WriteHeader(http.StatusInternalServerError)
	}

	json.NewEncoder(w).Encode(response)
}

// ReadyCheckHandler проверяет готовность сервиса к работе
func (h *HealthHandler) ReadyCheckHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	status := h.aggregator.GetOverallStatus(ctx)

	if status == ports.HealthStatusUp {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "READY"}`))
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(`{"status": "NOT_READY"}`))
	}
}

// LiveCheckHandler проверяет что сервис жив (минимальная проверка)
func (h *HealthHandler) LiveCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "ALIVE"}`))
}
