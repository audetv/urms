package domain

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// Value Objects для Email домена
type MessageID string
type EmailAddress string
type Direction string
type AttachmentID string

const (
	DirectionIncoming Direction = "incoming"
	DirectionOutgoing Direction = "outgoing"
)

// EmailMessage - доменная сущность для email сообщений
type EmailMessage struct {
	ID         MessageID
	MessageID  string   // RFC Message-ID header
	InReplyTo  string   // RFC In-Reply-To header
	References []string // RFC References headers

	// External Reference to TicketManagement domain
	RelatedTicketID *string `json:"related_ticket_id"`

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
	Processed   bool      `json:"processed"`
	ProcessedAt time.Time `json:"processed_at"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Attachment - вложение email сообщения
type Attachment struct {
	ID          AttachmentID
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

// Domain Methods

// NewIncomingEmail создает новое входящее email сообщение
func NewIncomingEmail(
	from EmailAddress,
	to []EmailAddress,
	subject string,
	messageID string,
	idGenerator IDGenerator,
) (*EmailMessage, error) {
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
		ID:        MessageID(idGenerator.GenerateID()),
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
func NewOutgoingEmail(
	from EmailAddress,
	to []EmailAddress,
	subject string,
	idGenerator IDGenerator,
) (*EmailMessage, error) {
	messageID := idGenerator.GenerateMessageID()
	msg, err := NewIncomingEmail(from, to, subject, messageID, idGenerator)
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

	if m.BodyText == "" && m.BodyHTML == "" {
		return errors.New("email must have either text or HTML body")
	}

	return nil
}

// IsReply проверяет, является ли сообщение ответом
func (m *EmailMessage) IsReply() bool {
	return m.InReplyTo != "" || len(m.References) > 0
}

// AddAttachment добавляет вложение к сообщению
func (m *EmailMessage) AddAttachment(
	name, contentType string,
	data []byte,
	idGenerator IDGenerator,
) error {
	if int64(len(data)) > 10*1024*1024 {
		return errors.New("attachment size exceeds 10MB limit")
	}

	attachment := Attachment{
		ID:          AttachmentID(idGenerator.GenerateID()),
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
		strings.Contains(strings.ToLower(m.Subject), "automatic") ||
		strings.Contains(strings.ToLower(m.Subject), "autoreply") {
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
		"credit card", "loan", "mortgage", "investment",
	}

	content := strings.ToLower(m.Subject + " " + m.BodyText)
	for _, indicator := range spamIndicators {
		if strings.Contains(content, indicator) {
			return true
		}
	}

	// Проверка заблокированных отправителей
	for _, blocked := range policy.BlockedSenders {
		if m.From == blocked {
			return true
		}
	}

	return false
}

// IsFromAllowedSender проверяет разрешенного отправителя
func (m *EmailMessage) IsFromAllowedSender(policy EmailProcessingPolicy) bool {
	if len(policy.AllowedSenders) == 0 {
		return true // Если список пустой, все разрешены
	}

	for _, allowed := range policy.AllowedSenders {
		if m.From == allowed {
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
