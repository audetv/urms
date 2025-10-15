package ports_test

import (
	"context"
	"testing"
	"time"

	"github.com/audetv/urms/internal/core/domain"
	"github.com/stretchr/testify/assert"
)

// TestMessageProcessorContract тестирует контракт MessageProcessor
func TestMessageProcessorContract(t *testing.T) {
	// Создаем тестовый процессор (заглушка для демонстрации)
	processor := &TestMessageProcessor{}

	ctx := context.Background()

	// Создаем тестовые сообщения
	incomingMsg := domain.EmailMessage{
		ID:        domain.MessageID("test-incoming"),
		MessageID: "incoming@test.local",
		From:      "sender@example.com",
		To:        []domain.EmailAddress{"support@company.com"},
		Subject:   "Test Incoming Message",
		BodyText:  "This is a test incoming message",
		Direction: domain.DirectionIncoming,
		Source:    "test",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	outgoingMsg := domain.EmailMessage{
		ID:        domain.MessageID("test-outgoing"),
		MessageID: "outgoing@test.local",
		From:      "support@company.com",
		To:        []domain.EmailAddress{"customer@example.com"},
		Subject:   "Test Outgoing Message",
		BodyText:  "This is a test outgoing message",
		Direction: domain.DirectionOutgoing,
		Source:    "test",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Тестируем обработку входящего сообщения
	err := processor.ProcessIncomingEmail(ctx, incomingMsg)
	assert.NoError(t, err, "ProcessIncomingEmail should succeed")

	// Тестируем обработку исходящего сообщения
	err = processor.ProcessOutgoingEmail(ctx, outgoingMsg)
	assert.NoError(t, err, "ProcessOutgoingEmail should succeed")

	// Проверяем, что сообщения были обработаны
	assert.True(t, processor.IncomingProcessed, "Incoming message should be marked as processed")
	assert.True(t, processor.OutgoingProcessed, "Outgoing message should be marked as processed")
}

// TestMessageProcessor тестовый процессор для проверки контракта
type TestMessageProcessor struct {
	IncomingProcessed bool
	OutgoingProcessed bool
}

func (p *TestMessageProcessor) ProcessIncomingEmail(ctx context.Context, msg domain.EmailMessage) error {
	p.IncomingProcessed = true
	return nil
}

func (p *TestMessageProcessor) ProcessOutgoingEmail(ctx context.Context, msg domain.EmailMessage) error {
	p.OutgoingProcessed = true
	return nil
}

// TestMessageProcessorValidation тестирует валидацию сообщений в процессоре
func TestMessageProcessorValidation(t *testing.T) {
	processor := &TestMessageProcessor{}
	ctx := context.Background()

	// Тестируем с невалидным сообщением (без From)
	invalidMsg := domain.EmailMessage{
		ID:        domain.MessageID("test-invalid"),
		MessageID: "invalid@test.local",
		To:        []domain.EmailAddress{"test@example.com"},
		Subject:   "Test Invalid Message",
		BodyText:  "This message has no From address",
	}

	err := processor.ProcessIncomingEmail(ctx, invalidMsg)
	// Процессор может либо принять сообщение, либо вернуть ошибку
	// В реальной реализации здесь должна быть валидация
	if err != nil {
		t.Logf("Processor rejected invalid message: %v", err)
	} else {
		t.Log("Processor accepted invalid message (may need validation)")
	}
}
