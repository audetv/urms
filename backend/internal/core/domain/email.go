package domain

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ===== INTEGRATION LAYER DOMAIN =====
// Email-specific domain models for Integration Layer

// Value Objects для Email домена
type MessageID string
type EmailAddress string
type Direction string

const (
	DirectionIncoming Direction = "incoming"
	DirectionOutgoing Direction = "outgoing"
)

// Domain Errors для Email домена
var (
	ErrInvalidEmailAddress = errors.New("invalid email address")
	ErrEmptySubject        = errors.New("email subject cannot be empty")
	ErrMessageTooLarge     = errors.New("email message size exceeds limit")
)

// EmailMessage - доменная сущность для email сообщений в Integration Layer
type EmailMessage struct {
	ID         MessageID
	MessageID  string   // RFC Message-ID header
	InReplyTo  string   // RFC In-Reply-To header
	References []string // RFC References headers

	// External Reference to TicketManagement domain
	// Это связь с другим bounded context
	RelatedTicketID *string `json:"related_ticket_id"` // Optional: ID связанной заявки

	From    EmailAddress
	To      []EmailAddress
	CC      []EmailAddress
	BCC     []EmailAddress
	Subject string

	BodyHTML string
	BodyText string

	Attachments []Attachment
	Direction   Direction
	Source      string // imap, smtp, web, api

	Headers map[string][]string

	// Metadata
	Processed   bool      `json:"processed"`    // Обработано ли системой
	ProcessedAt time.Time `json:"processed_at"` // Когда обработано
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Attachment - вложение email сообщения
type Attachment struct {
	ID          uuid.UUID
	Name        string
	ContentType string
	Size        int64
	ContentID   string
	Data        []byte
}

// EmailProcessingPolicy - политики обработки email
type EmailProcessingPolicy struct {
	ReadOnlyMode   bool
	AutoReply      bool
	AutoReplyText  string
	SpamFilter     bool
	MaxMessageSize int64
	AllowedSenders []EmailAddress
	BlockedSenders []EmailAddress
}

// EmailChannelConfig - конфигурация email канала
type EmailChannelConfig struct {
	Provider     string // imap, smtp, api
	Server       string
	Port         int
	Username     string
	Mailbox      string
	SSL          bool
	PollInterval time.Duration
}

// Domain Methods для Email домена

// NewIncomingEmail создает новое входящее email сообщение
func NewIncomingEmail(from EmailAddress, to []EmailAddress, subject, messageID string) (*EmailMessage, error) {
	if err := validateEmailAddress(from); err != nil {
		return nil, err
	}

	for _, addr := range to {
		if err := validateEmailAddress(addr); err != nil {
			return nil, err
		}
	}

	if strings.TrimSpace(subject) == "" {
		return nil, ErrEmptySubject
	}

	msg := &EmailMessage{
		ID:        MessageID(generateMessageID()),
		From:      from,
		To:        to,
		Subject:   subject,
		MessageID: messageID,
		Direction: DirectionIncoming,
		Source:    "imap",
		Processed: false,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Headers:   make(map[string][]string),
	}

	return msg, nil
}

// NewOutgoingEmail создает новое исходящее email сообщение
func NewOutgoingEmail(from EmailAddress, to []EmailAddress, subject string) (*EmailMessage, error) {
	msg, err := NewIncomingEmail(from, to, subject, generateMessageID())
	if err != nil {
		return nil, err
	}

	msg.Direction = DirectionOutgoing
	msg.Source = "internal"
	return msg, nil
}

// MarkAsProcessed отмечает сообщение как обработанное системой
func (m *EmailMessage) MarkAsProcessed(ticketID *string) {
	m.Processed = true
	m.ProcessedAt = time.Now()
	m.RelatedTicketID = ticketID
	m.UpdatedAt = time.Now()
}

// LinkToTicket связывает email с заявкой
func (m *EmailMessage) LinkToTicket(ticketID string) {
	m.RelatedTicketID = &ticketID
	m.UpdatedAt = time.Now()
}

// Validate проверяет валидность email сообщения
func (m *EmailMessage) Validate() error {
	if err := validateEmailAddress(m.From); err != nil {
		return fmt.Errorf("invalid sender: %w", err)
	}

	if len(m.To) == 0 {
		return errors.New("at least one recipient is required")
	}

	for _, addr := range m.To {
		if err := validateEmailAddress(addr); err != nil {
			return fmt.Errorf("invalid recipient: %w", err)
		}
	}

	if strings.TrimSpace(m.Subject) == "" {
		return ErrEmptySubject
	}

	return nil
}

// IsReply проверяет, является ли сообщение ответом
func (m *EmailMessage) IsReply() bool {
	return m.InReplyTo != "" || len(m.References) > 0
}

// AddAttachment добавляет вложение к сообщению
func (m *EmailMessage) AddAttachment(name, contentType string, data []byte) error {
	if int64(len(data)) > 10*1024*1024 {
		return errors.New("attachment size exceeds 10MB limit")
	}

	attachment := Attachment{
		ID:          uuid.New(),
		Name:        name,
		ContentType: contentType,
		Size:        int64(len(data)),
		Data:        data,
	}

	m.Attachments = append(m.Attachments, attachment)
	m.UpdatedAt = time.Now()

	return nil
}

// Business Rules для Email домена

// CanAutoReply проверяет, можно ли отправлять авто-ответ
func (m *EmailMessage) CanAutoReply(policy EmailProcessingPolicy) bool {
	if !policy.AutoReply {
		return false
	}

	// Бизнес-правило: не отправляем авто-ответы на авто-ответы
	if strings.Contains(strings.ToLower(m.Subject), "auto:") ||
		strings.Contains(strings.ToLower(m.Subject), "automatic") {
		return false
	}

	return true
}

// IsSpam проверяет, является ли сообщение спамом
func (m *EmailMessage) IsSpam(policy EmailProcessingPolicy) bool {
	if !policy.SpamFilter {
		return false
	}

	spamIndicators := []string{
		"viagra", "casino", "lottery", "prize", "winner",
	}

	content := strings.ToLower(m.Subject + " " + m.BodyText)
	for _, indicator := range spamIndicators {
		if strings.Contains(content, indicator) {
			return true
		}
	}

	return false
}

// Helper functions
func validateEmailAddress(addr EmailAddress) error {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(string(addr)) {
		return ErrInvalidEmailAddress
	}
	return nil
}

func generateMessageID() string {
	return fmt.Sprintf("%s@urms.local", uuid.New().String())
}
