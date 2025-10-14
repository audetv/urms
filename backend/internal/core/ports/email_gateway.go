package ports

import (
	"context"
	"time"

	"github.com/audetv/urms/internal/core/domain"
)

// EmailGateway определяет контракт для работы с email провайдерами
type EmailGateway interface {
	// Connection Management
	Connect(ctx context.Context) error
	Disconnect() error
	HealthCheck(ctx context.Context) error

	// Message Operations
	FetchMessages(ctx context.Context, criteria FetchCriteria) ([]domain.EmailMessage, error)
	SendMessage(ctx context.Context, msg domain.EmailMessage) error
	MarkAsRead(ctx context.Context, messageIDs []string) error
	MarkAsProcessed(ctx context.Context, messageIDs []string) error

	// Mailbox Operations
	ListMailboxes(ctx context.Context) ([]MailboxInfo, error)
	SelectMailbox(ctx context.Context, name string) error
	GetMailboxInfo(ctx context.Context, name string) (*MailboxInfo, error)
}

// EmailRepository определяет контракт для хранения email сообщений
type EmailRepository interface {
	// Basic CRUD
	Save(ctx context.Context, msg *domain.EmailMessage) error
	FindByID(ctx context.Context, id domain.MessageID) (*domain.EmailMessage, error)
	FindByMessageID(ctx context.Context, messageID string) (*domain.EmailMessage, error)
	Update(ctx context.Context, msg *domain.EmailMessage) error
	Delete(ctx context.Context, id domain.MessageID) error

	// Query methods
	FindUnprocessed(ctx context.Context) ([]domain.EmailMessage, error)
	FindByPeriod(ctx context.Context, from, to time.Time) ([]domain.EmailMessage, error)

	// Thread-related queries (для будущей интеграции с TicketManagement)
	FindByInReplyTo(ctx context.Context, inReplyTo string) ([]domain.EmailMessage, error)
	FindByReferences(ctx context.Context, references []string) ([]domain.EmailMessage, error)
}

// EmailConfigProvider для управления конфигурацией email каналов
type EmailConfigProvider interface {
	GetConfig(ctx context.Context, channelID string) (*domain.EmailChannelConfig, error)
	SaveConfig(ctx context.Context, config *domain.EmailChannelConfig) error
	ListConfigs(ctx context.Context) ([]domain.EmailChannelConfig, error)
	ValidateConfig(ctx context.Context, config *domain.EmailChannelConfig) error
}

// Supporting types for EmailGateway
type FetchCriteria struct {
	Since      time.Time
	SinceUID   uint32
	Mailbox    string
	Limit      int
	UnseenOnly bool
}

type MailboxInfo struct {
	Name     string
	Messages int
	Unseen   int
	Recent   int
}
