// backend/internal/infrastructure/email/imap_health_check.go
package email

import (
	"context"
	"time"

	"github.com/audetv/urms/internal/core/ports"
)

// IMAPHealthChecker реализует HealthChecker для IMAP соединения
type IMAPHealthChecker struct {
	adapter *IMAPAdapter
	name    string
}

// NewIMAPHealthChecker создает новый health checker для IMAP
func NewIMAPHealthChecker(adapter *IMAPAdapter) *IMAPHealthChecker {
	return &IMAPHealthChecker{
		adapter: adapter,
		name:    "imap_email_gateway",
	}
}

// CheckHealth выполняет проверку здоровья IMAP соединения
func (h *IMAPHealthChecker) CheckHealth(ctx context.Context) *ports.HealthStatus {
	status := &ports.HealthStatus{
		Name:      h.name,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Details:   make(map[string]interface{}),
	}

	// Проверяем соединение
	if err := h.adapter.HealthCheck(ctx); err != nil {
		status.Status = ports.HealthStatusDown
		status.Message = "IMAP connection failed"
		status.Details["error"] = err.Error()
		return status
	}

	// Получаем информацию о почтовом ящике для дополнительной проверки
	mailboxInfo, err := h.adapter.GetMailboxInfo(ctx, "INBOX")
	if err != nil {
		status.Status = ports.HealthStatusDegraded
		status.Message = "IMAP connected but mailbox info unavailable"
		status.Details["warning"] = err.Error()
	} else {
		status.Status = ports.HealthStatusUp
		status.Message = "IMAP connection healthy"
		status.Details["mailbox"] = mailboxInfo.Name
		status.Details["total_messages"] = mailboxInfo.Messages
		status.Details["unseen_messages"] = mailboxInfo.Unseen
	}

	return status
}

// GetName возвращает имя компонента
func (h *IMAPHealthChecker) GetName() string {
	return h.name
}
