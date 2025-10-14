package services

import (
	"context"

	"github.com/audetv/urms/internal/core/domain"
	"github.com/audetv/urms/internal/core/ports"
)

// DummyMessageProcessor заглушка для MessageProcessor
// Будет заменена на реальную реализацию когда добавим TicketManagement
type DummyMessageProcessor struct {
	logger ports.Logger
}

func NewDummyMessageProcessor(logger ports.Logger) *DummyMessageProcessor {
	return &DummyMessageProcessor{
		logger: logger,
	}
}

func (p *DummyMessageProcessor) ProcessIncomingEmail(ctx context.Context, email domain.EmailMessage) error {
	p.logger.Info(ctx, "Dummy processor: processing incoming email",
		"message_id", email.MessageID, "subject", email.Subject)

	// Заглушка - просто логируем
	// В будущем здесь будет логика создания тикетов и т.д.

	return nil
}

func (p *DummyMessageProcessor) ProcessOutgoingEmail(ctx context.Context, email domain.EmailMessage) error {
	p.logger.Info(ctx, "Dummy processor: processing outgoing email",
		"message_id", email.MessageID, "subject", email.Subject)

	// Заглушка - просто логируем

	return nil
}
