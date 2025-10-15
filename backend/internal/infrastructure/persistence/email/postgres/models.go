// backend/internal/infrastructure/persistence/email/models.go
package postgres

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/audetv/urms/internal/core/domain"
)

// PostgreSQL модели для маппинга domain сущностей

// EmailMessageModel представляет email сообщение в PostgreSQL
type EmailMessageModel struct {
	ID              string          `db:"id"`
	MessageID       string          `db:"message_id"`
	InReplyTo       string          `db:"in_reply_to"`
	ThreadID        string          `db:"thread_id"`
	FromEmail       string          `db:"from_email"`
	ToEmails        json.RawMessage `db:"to_emails"`
	CcEmails        json.RawMessage `db:"cc_emails"`
	BccEmails       json.RawMessage `db:"bcc_emails"`
	Subject         string          `db:"subject"`
	BodyText        string          `db:"body_text"`
	BodyHTML        string          `db:"body_html"`
	Direction       string          `db:"direction"`
	Source          string          `db:"source"`
	Headers         json.RawMessage `db:"headers"`
	Processed       bool            `db:"processed"`
	ProcessedAt     sql.NullTime    `db:"processed_at"` // Меняем на sql.NullTime
	RelatedTicketID *string         `db:"related_ticket_id"`
	CreatedAt       time.Time       `db:"created_at"`
	UpdatedAt       time.Time       `db:"updated_at"`
}

// AttachmentModel представляет вложение в PostgreSQL
type AttachmentModel struct {
	ID          string    `db:"id"`
	MessageID   string    `db:"message_id"`
	Name        string    `db:"name"`
	ContentType string    `db:"content_type"`
	Size        int64     `db:"size"`
	ContentID   string    `db:"content_id"`
	Data        []byte    `db:"data"`
	CreatedAt   time.Time `db:"created_at"`
}

// Helper функции для конвертации между domain и PostgreSQL моделями

// ToDomain конвертирует PostgreSQL модель в domain сущность
func (m *EmailMessageModel) ToDomain() (*domain.EmailMessage, error) {
	// Конвертируем JSONB поля
	var toEmails []string
	if len(m.ToEmails) > 0 {
		if err := json.Unmarshal(m.ToEmails, &toEmails); err != nil {
			return nil, err
		}
	}

	var ccEmails []string
	if len(m.CcEmails) > 0 {
		if err := json.Unmarshal(m.CcEmails, &ccEmails); err != nil {
			return nil, err
		}
	}

	var bccEmails []string
	if len(m.BccEmails) > 0 {
		if err := json.Unmarshal(m.BccEmails, &bccEmails); err != nil {
			return nil, err
		}
	}

	var headers map[string][]string
	if len(m.Headers) > 0 {
		if err := json.Unmarshal(m.Headers, &headers); err != nil {
			return nil, err
		}
	}

	// Конвертируем domain.EmailAddress
	domainTo := make([]domain.EmailAddress, len(toEmails))
	for i, email := range toEmails {
		domainTo[i] = domain.EmailAddress(email)
	}

	domainCc := make([]domain.EmailAddress, len(ccEmails))
	for i, email := range ccEmails {
		domainCc[i] = domain.EmailAddress(email)
	}

	domainBcc := make([]domain.EmailAddress, len(bccEmails))
	for i, email := range bccEmails {
		domainBcc[i] = domain.EmailAddress(email)
	}

	// Конвертируем ProcessedAt
	var processedAt time.Time
	if m.ProcessedAt.Valid {
		processedAt = m.ProcessedAt.Time
	}

	msg := &domain.EmailMessage{
		ID:              domain.MessageID(m.ID),
		MessageID:       m.MessageID,
		InReplyTo:       m.InReplyTo,
		From:            domain.EmailAddress(m.FromEmail),
		To:              domainTo,
		CC:              domainCc,
		BCC:             domainBcc,
		Subject:         m.Subject,
		BodyText:        m.BodyText,
		BodyHTML:        m.BodyHTML,
		Direction:       domain.Direction(m.Direction),
		Source:          m.Source,
		Headers:         headers,
		Processed:       m.Processed,
		ProcessedAt:     processedAt, // Теперь time.Time
		RelatedTicketID: m.RelatedTicketID,
		CreatedAt:       m.CreatedAt,
		UpdatedAt:       m.UpdatedAt,
	}

	// References будут загружаться отдельно если нужно
	return msg, nil
}

// FromDomain конвертирует domain сущность в PostgreSQL модель
func FromDomain(msg *domain.EmailMessage) (*EmailMessageModel, error) {
	// Конвертируем domain.EmailAddress в строки для JSONB
	toEmails := make([]string, len(msg.To))
	for i, email := range msg.To {
		toEmails[i] = string(email)
	}

	ccEmails := make([]string, len(msg.CC))
	for i, email := range msg.CC {
		ccEmails[i] = string(email)
	}

	bccEmails := make([]string, len(msg.BCC))
	for i, email := range msg.BCC {
		bccEmails[i] = string(email)
	}

	// Сериализуем в JSONB
	toJSON, err := json.Marshal(toEmails)
	if err != nil {
		return nil, err
	}

	ccJSON, err := json.Marshal(ccEmails)
	if err != nil {
		return nil, err
	}

	bccJSON, err := json.Marshal(bccEmails)
	if err != nil {
		return nil, err
	}

	headersJSON, err := json.Marshal(msg.Headers)
	if err != nil {
		return nil, err
	}

	// Конвертируем ProcessedAt
	var processedAt sql.NullTime
	if !msg.ProcessedAt.IsZero() {
		processedAt = sql.NullTime{
			Time:  msg.ProcessedAt,
			Valid: true,
		}
	}

	model := &EmailMessageModel{
		ID:              string(msg.ID),
		MessageID:       msg.MessageID,
		InReplyTo:       msg.InReplyTo,
		ThreadID:        "", // TODO: Реализовать thread management
		FromEmail:       string(msg.From),
		ToEmails:        toJSON,
		CcEmails:        ccJSON,
		BccEmails:       bccJSON,
		Subject:         msg.Subject,
		BodyText:        msg.BodyText,
		BodyHTML:        msg.BodyHTML,
		Direction:       string(msg.Direction),
		Source:          msg.Source,
		Headers:         headersJSON,
		Processed:       msg.Processed,
		ProcessedAt:     processedAt,
		RelatedTicketID: msg.RelatedTicketID,
		CreatedAt:       msg.CreatedAt,
		UpdatedAt:       msg.UpdatedAt,
	}

	return model, nil
}
