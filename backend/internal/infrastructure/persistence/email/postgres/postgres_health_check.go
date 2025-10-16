// backend/internal/infrastructure/persistence/email/postgres/postgres_health_check.go
package postgres

import (
	"context"
	"time"

	"github.com/audetv/urms/internal/core/ports"
	"github.com/jmoiron/sqlx"
)

// PostgresHealthChecker реализует HealthChecker для PostgreSQL
type PostgresHealthChecker struct {
	db   *sqlx.DB
	name string
}

// NewPostgresHealthChecker создает новый health checker для PostgreSQL
func NewPostgresHealthChecker(db *sqlx.DB) *PostgresHealthChecker {
	return &PostgresHealthChecker{
		db:   db,
		name: "postgres_email_repository",
	}
}

// CheckHealth выполняет проверку здоровья PostgreSQL соединения
func (h *PostgresHealthChecker) CheckHealth(ctx context.Context) *ports.HealthStatus {
	status := &ports.HealthStatus{
		Name:      h.name,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Details:   make(map[string]interface{}),
	}

	// Проверяем соединение с базой
	if err := h.db.PingContext(ctx); err != nil {
		status.Status = ports.HealthStatusDown
		status.Message = "PostgreSQL connection failed"
		status.Details["error"] = err.Error()
		return status
	}

	// Проверяем доступность таблицы email_messages
	var tableCount int
	err := h.db.GetContext(ctx, &tableCount,
		"SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'email_messages'")

	if err != nil {
		status.Status = ports.HealthStatusDegraded
		status.Message = "PostgreSQL connected but table check failed"
		status.Details["warning"] = err.Error()
		return status
	}

	if tableCount == 0 {
		status.Status = ports.HealthStatusDegraded
		status.Message = "PostgreSQL connected but email tables missing"
		status.Details["warning"] = "email_messages table not found"
		return status
	}

	// Получаем статистику по таблице
	var messageCount int
	h.db.GetContext(ctx, &messageCount, "SELECT COUNT(*) FROM email_messages")

	status.Status = ports.HealthStatusUp
	status.Message = "PostgreSQL connection healthy"
	status.Details["table_exists"] = true
	status.Details["message_count"] = messageCount

	return status
}

// GetName возвращает имя компонента
func (h *PostgresHealthChecker) GetName() string {
	return h.name
}
