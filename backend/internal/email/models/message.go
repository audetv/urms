// backend/internal/email/models/message.go
package models

import (
	"time"

	"github.com/google/uuid"
)

type EmailMessage struct {
	ID         uuid.UUID `json:"id" db:"id"`
	MessageID  string    `json:"message_id" db:"message_id"`   // RFC Message-ID header
	InReplyTo  string    `json:"in_reply_to" db:"in_reply_to"` // RFC In-Reply-To header
	References []string  `json:"references" db:"references"`   // RFC References headers
	ThreadID   string    `json:"thread_id" db:"thread_id"`     // Our internal thread ID

	From    string   `json:"from" db:"from_email"`
	To      []string `json:"to" db:"to_emails"`
	CC      []string `json:"cc" db:"cc_emails"`
	Subject string   `json:"subject" db:"subject"`

	BodyHTML string `json:"body_html" db:"body_html"`
	BodyText string `json:"body_text" db:"body_text"`

	Attachments []Attachment `json:"attachments" db:"attachments"`
	Direction   Direction    `json:"direction" db:"direction"` // incoming, outgoing
	Source      string       `json:"source" db:"source"`       // imap, smtp, web

	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

type Attachment struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	ContentType string    `json:"content_type"`
	Size        int64     `json:"size"`
	Data        []byte    `json:"data,omitempty"` // omitempty для избежания больших JSON
	ContentID   string    `json:"content_id"`     // для inline attachments
}

type Direction string

const (
	DirectionIncoming Direction = "incoming"
	DirectionOutgoing Direction = "outgoing"
)

// EnvelopeInfo содержит базовую информацию о письме из IMAP envelope
type EnvelopeInfo struct {
	From      []string  `json:"from"`
	To        []string  `json:"to"`
	CC        []string  `json:"cc"`
	BCC       []string  `json:"bcc"`
	Subject   string    `json:"subject"`
	MessageID string    `json:"message_id"`
	InReplyTo string    `json:"in_reply_to"`
	Date      time.Time `json:"date"`
}
