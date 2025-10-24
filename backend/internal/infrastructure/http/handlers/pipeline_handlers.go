// internal/infrastructure/http/handlers/pipeline_handlers.go
package handlers

import (
	"net/http"
	"time"

	"github.com/audetv/urms/internal/core/ports"
	"github.com/gin-gonic/gin"
)

type PipelineHandler struct {
	pipeline ports.EmailPipeline
}

func NewPipelineHandler(pipeline ports.EmailPipeline) *PipelineHandler {
	return &PipelineHandler{
		pipeline: pipeline,
	}
}

// GetHealth возвращает статус здоровья pipeline
func (h *PipelineHandler) GetHealth(c *gin.Context) {
	health, err := h.pipeline.Health(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "unhealthy",
			"error":  err.Error(),
			"time":   time.Now().Format(time.RFC3339),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":     health.Status,
		"timestamp":  health.Timestamp.Format(time.RFC3339),
		"components": health.Components,
	})
}

// GetMetrics возвращает метрики pipeline
func (h *PipelineHandler) GetMetrics(c *gin.Context) {
	metrics, err := h.pipeline.GetMetrics(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed to get pipeline metrics",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"provider_type":      metrics.ProviderType,
		"uptime":             metrics.Uptime.String(),
		"total_processed":    metrics.TotalProcessed,
		"total_failed":       metrics.TotalFailed,
		"current_queue_size": metrics.CurrentQueueSize,
		"workers_active":     metrics.WorkersActive,
		"workers_total":      metrics.WorkersTotal,
		"last_processed":     metrics.LastProcessed.Format(time.RFC3339),
		"avg_process_time":   metrics.AvgProcessTime.String(),
	})
}

// ProcessBatch запускает обработку батча вручную
func (h *PipelineHandler) ProcessBatch(c *gin.Context) {
	if err := h.pipeline.ProcessBatch(c.Request.Context()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "batch processing failed",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "batch processing completed",
		"time":    time.Now().Format(time.RFC3339),
	})
}
