// backend/internal/core/ports/email_contract_test.go
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

	// ✅ НОВЫЙ МЕТОД: Thread-aware поиск сообщений
	SearchThreadMessages(ctx context.Context, criteria ThreadSearchCriteria) ([]domain.EmailMessage, error)

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
	Subject    string // ✅ ДОБАВЛЕНО: для subject-based поиска
}

type MailboxInfo struct {
	Name     string
	Messages int
	Unseen   int
	Recent   int
}

// ✅ НОВЫЙ ТИП: ThreadSearchCriteria для thread-aware поиска
type ThreadSearchCriteria struct {
	MessageID  string   // Message-ID текущего сообщения
	InReplyTo  string   // In-Reply-To header
	References []string // References headers
	Subject    string   // Нормализованный subject (без Re:/Fwd:)
	Mailbox    string   // Почтовый ящик для поиска
}
