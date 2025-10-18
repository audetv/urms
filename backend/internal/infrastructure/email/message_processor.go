// internal/infrastructure/email/message_processor.go
package email

import (
	"context"
	"fmt"

	"github.com/audetv/urms/internal/core/domain"
	"github.com/audetv/urms/internal/core/ports"
)

// DefaultMessageProcessor базовая реализация обработки сообщений
type DefaultMessageProcessor struct {
	logger ports.Logger
}

// NewDefaultMessageProcessor создает новый экземпляр процессора
func NewDefaultMessageProcessor(logger ports.Logger) ports.MessageProcessor {
	return &DefaultMessageProcessor{
		logger: logger,
	}
}

// ProcessIncomingEmail обрабатывает входящие email сообщения
func (p *DefaultMessageProcessor) ProcessIncomingEmail(ctx context.Context, email domain.EmailMessage) error {
	p.logger.Info(ctx, "Processing incoming email",
		"message_id", email.MessageID,
		"from", email.From,
		"subject", email.Subject,
		"operation", "process_incoming_email")

	// Бизнес-логика обработки входящих сообщений
	actions := []string{}

	// 1. Валидация и нормализация
	if err := p.validateIncomingEmail(ctx, email); err != nil {
		p.logger.Error(ctx, "Incoming email validation failed",
			"message_id", email.MessageID,
			"error", err.Error())
		return fmt.Errorf("email validation failed: %w", err)
	}
	actions = append(actions, "validated")

	// 2. Анализ содержимого
	analysis := p.analyzeEmailContent(ctx, email)
	actions = append(actions, analysis...)

	// 3. Логика создания/обновления тикетов (заглушка для Phase 2)
	if p.shouldCreateTicket(ctx, email) {
		actions = append(actions, "ticket_creation_required")
		p.logger.Info(ctx, "Ticket creation required for email",
			"message_id", email.MessageID,
			"subject", email.Subject)
	}

	p.logger.Info(ctx, "Incoming email processed successfully",
		"message_id", email.MessageID,
		"actions", actions,
		"operation", "incoming_email_processed")

	return nil
}

// ProcessOutgoingEmail обрабатывает исходящие email сообщения
func (p *DefaultMessageProcessor) ProcessOutgoingEmail(ctx context.Context, email domain.EmailMessage) error {
	p.logger.Info(ctx, "Processing outgoing email",
		"message_id", email.MessageID,
		"to", email.To,
		"subject", email.Subject,
		"operation", "process_outgoing_email")

	// Бизнес-логика обработки исходящих сообщений
	actions := []string{}

	// 1. Валидация исходящего сообщения
	if err := p.validateOutgoingEmail(ctx, email); err != nil {
		p.logger.Error(ctx, "Outgoing email validation failed",
			"message_id", email.MessageID,
			"error", err.Error())
		return fmt.Errorf("outgoing email validation failed: %w", err)
	}
	actions = append(actions, "validated")

	// 2. Логирование исходящей переписки
	actions = append(actions, "correspondence_logged")

	// 3. Обновление статуса тикетов (заглушка для Phase 2)
	if p.shouldUpdateTicketStatus(ctx, email) {
		actions = append(actions, "ticket_status_update_required")
	}

	p.logger.Info(ctx, "Outgoing email processed successfully",
		"message_id", email.MessageID,
		"actions", actions,
		"operation", "outgoing_email_processed")

	return nil
}

// validateIncomingEmail проверяет входящее сообщение
func (p *DefaultMessageProcessor) validateIncomingEmail(ctx context.Context, email domain.EmailMessage) error {
	if email.MessageID == "" {
		return fmt.Errorf("message ID is required")
	}
	if email.From == "" {
		return fmt.Errorf("sender address is required")
	}
	if len(email.To) == 0 && len(email.CC) == 0 {
		return fmt.Errorf("no recipients found")
	}
	return nil
}

// validateOutgoingEmail проверяет исходящее сообщение
func (p *DefaultMessageProcessor) validateOutgoingEmail(ctx context.Context, email domain.EmailMessage) error {
	if email.MessageID == "" {
		return fmt.Errorf("message ID is required")
	}
	if len(email.To) == 0 {
		return fmt.Errorf("at least one recipient is required")
	}
	if email.Subject == "" {
		return fmt.Errorf("subject is required for outgoing emails")
	}
	return nil
}

// analyzeEmailContent анализирует содержимое email
func (p *DefaultMessageProcessor) analyzeEmailContent(ctx context.Context, email domain.EmailMessage) []string {
	actions := []string{"content_analyzed"}

	// Простой анализ на основе содержимого
	if email.BodyText != "" {
		textLength := len(email.BodyText)
		if textLength > 1000 {
			actions = append(actions, "long_content")
		} else if textLength < 50 {
			actions = append(actions, "short_content")
		}
	}

	if email.BodyHTML != "" {
		actions = append(actions, "html_content")
	}

	if len(email.Attachments) > 0 {
		actions = append(actions, fmt.Sprintf("attachments_%d", len(email.Attachments)))
	}

	return actions
}

// shouldCreateTicket определяет нужно ли создавать тикет
func (p *DefaultMessageProcessor) shouldCreateTicket(ctx context.Context, email domain.EmailMessage) bool {
	// Временная логика - создавать тикет для всех входящих сообщений
	// В Phase 2 будет интегрирована AI классификация
	return true
}

// shouldUpdateTicketStatus определяет нужно ли обновлять статус тикета
func (p *DefaultMessageProcessor) shouldUpdateTicketStatus(ctx context.Context, email domain.EmailMessage) bool {
	// Временная логика - обновлять для всех исходящих ответов
	// В Phase 2 будет интегрирована с Ticket Management
	return true
}
