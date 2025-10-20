// internal/infrastructure/http/handlers/health_handler.go
package handlers

import (
	"net/http"
	"time"

	"github.com/audetv/urms/internal/core/ports"
	"github.com/audetv/urms/internal/infrastructure/http/dto"
	"github.com/gin-gonic/gin"
)

type HealthHandler struct {
	aggregator ports.HealthAggregator
}

func NewHealthHandler(aggregator ports.HealthAggregator) *HealthHandler {
	return &HealthHandler{
		aggregator: aggregator,
	}
}

// HealthCheck возвращает детальный статус здоровья системы
// @Summary Health check
// @Description Возвращает статус здоровья всех компонентов системы
// @Tags system
// @Produce json
// @Success 200 {object} dto.BaseResponse
// @Success 503 {object} dto.BaseResponse
// @Router /health [get]
func (h *HealthHandler) HealthCheck(c *gin.Context) {
	ctx := c.Request.Context()

	overallStatus := h.aggregator.GetOverallStatus(ctx)
	detailedStatus := h.aggregator.CheckAll(ctx)

	response := map[string]interface{}{
		"status":    string(overallStatus),
		"timestamp": time.Now().Format(time.RFC3339),
		"services":  detailedStatus,
	}

	// Устанавливаем HTTP статус в зависимости от здоровья системы
	var httpStatus int
	switch overallStatus {
	case ports.HealthStatusUp:
		httpStatus = http.StatusOK
	case ports.HealthStatusDegraded:
		httpStatus = http.StatusOK // 200 но с degraded статусом
	case ports.HealthStatusDown:
		httpStatus = http.StatusServiceUnavailable
	default:
		httpStatus = http.StatusInternalServerError
	}

	c.JSON(httpStatus, dto.NewSuccessResponse(response))
}

// ReadyCheck проверяет готовность сервиса к работе
// @Summary Ready check
// @Description Проверяет готовность сервиса принимать трафик
// @Tags system
// @Produce json
// @Success 200 {object} dto.BaseResponse
// @Success 503 {object} dto.BaseResponse
// @Router /ready [get]
func (h *HealthHandler) ReadyCheck(c *gin.Context) {
	ctx := c.Request.Context()

	status := h.aggregator.GetOverallStatus(ctx)

	if status == ports.HealthStatusUp {
		c.JSON(http.StatusOK, dto.NewSuccessResponse(map[string]string{
			"status": "READY",
		}))
	} else {
		c.JSON(http.StatusServiceUnavailable, dto.NewErrorResponse(
			"SERVICE_NOT_READY",
			"Сервис не готов к работе",
			"",
		))
	}
}

// LiveCheck проверяет что сервис жив
// @Summary Liveness check
// @Description Проверяет что сервис работает (минимальная проверка)
// @Tags system
// @Produce json
// @Success 200 {object} dto.BaseResponse
// @Router /live [get]
func (h *HealthHandler) LiveCheck(c *gin.Context) {
	c.JSON(http.StatusOK, dto.NewSuccessResponse(map[string]string{
		"status": "ALIVE",
	}))
}
