// internal/infrastructure/email/email_gateway_health_adapter.go
package email

import (
	"context"
	"time"

	"github.com/audetv/urms/internal/core/ports"
)

// EmailGatewayHealthAdapter адаптирует ports.EmailGateway к ports.HealthChecker
type EmailGatewayHealthAdapter struct {
	gateway ports.EmailGateway
	name    string
}

// NewEmailGatewayHealthAdapter создает адаптер для health checking
func NewEmailGatewayHealthAdapter(gateway ports.EmailGateway) *EmailGatewayHealthAdapter {
	return &EmailGatewayHealthAdapter{
		gateway: gateway,
		name:    "email_gateway",
	}
}

// CheckHealth выполняет проверку здоровья через EmailGateway
func (a *EmailGatewayHealthAdapter) CheckHealth(ctx context.Context) *ports.HealthStatus {
	status := &ports.HealthStatus{
		Name:      a.name,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Details:   make(map[string]interface{}),
	}

	// Проверяем соединение через EmailGateway.HealthCheck
	if err := a.gateway.HealthCheck(ctx); err != nil {
		status.Status = ports.HealthStatusDown
		status.Message = "Email gateway health check failed"
		status.Details["error"] = err.Error()
		return status
	}

	// Дополнительная проверка - получаем информацию о почтовом ящике
	mailboxInfo, err := a.gateway.GetMailboxInfo(ctx, "INBOX")
	if err != nil {
		status.Status = ports.HealthStatusDegraded
		status.Message = "Email gateway connected but mailbox info unavailable"
		status.Details["warning"] = err.Error()
	} else {
		status.Status = ports.HealthStatusUp
		status.Message = "Email gateway connection healthy"
		status.Details["mailbox"] = mailboxInfo.Name
		status.Details["total_messages"] = mailboxInfo.Messages
		status.Details["unseen_messages"] = mailboxInfo.Unseen
	}

	return status
}

// GetName возвращает имя компонента
func (a *EmailGatewayHealthAdapter) GetName() string {
	return a.name
}
