package ports

import (
	"context"

	"github.com/audetv/urms/internal/core/domain"
)

// MessageProcessor обрабатывает входящие email сообщения
// Это ЕДИНСТВЕННЫЙ интерфейс для будущей интеграции с TicketManagement
type MessageProcessor interface {
	// Основной процесс обработки входящего email
	ProcessIncomingEmail(ctx context.Context, email domain.EmailMessage) error

	// Обработка исходящих сообщений
	ProcessOutgoingEmail(ctx context.Context, email domain.EmailMessage) error
}

// ProcessingResult результат обработки сообщения
type ProcessingResult struct {
	Success      bool
	ActionsTaken []string
	Error        error
}
