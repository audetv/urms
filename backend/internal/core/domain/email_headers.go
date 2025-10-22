// backend/internal/core/domain/email_headers.go
package domain

import (
	"fmt"
	"strings"
	"time"
)

// EmailHeaders представляет value object для email заголовков
// Содержит только бизнес-значимые заголовки для threading и отображения
type EmailHeaders struct {
	// Essential headers для email threading
	MessageID  string   `json:"message_id"`
	InReplyTo  string   `json:"in_reply_to"`
	References []string `json:"references"`

	// Basic email metadata
	Subject string         `json:"subject"`
	From    EmailAddress   `json:"from"`
	To      []EmailAddress `json:"to"`
	Cc      []EmailAddress `json:"cc"`
	Date    time.Time      `json:"date"`

	// Content information
	ContentType string `json:"content_type"`

	// Additional business headers
	Priority   string `json:"priority,omitempty"`
	Importance string `json:"importance,omitempty"`
}

// EssentialHeaderKeys определяет какие заголовки считаются бизнес-значимыми
var EssentialHeaderKeys = map[string]bool{
	"Message-ID":   true,
	"In-Reply-To":  true,
	"References":   true,
	"Subject":      true,
	"From":         true,
	"To":           true,
	"Cc":           true,
	"Date":         true,
	"Content-Type": true,
	"Priority":     true,
	"Importance":   true,
}

// NewEmailHeaders создает новый EmailHeaders из существующего EmailMessage
func NewEmailHeaders(email *EmailMessage) (*EmailHeaders, error) {
	if email == nil {
		return nil, fmt.Errorf("email message cannot be nil")
	}

	headers := &EmailHeaders{
		MessageID:  email.MessageID,
		InReplyTo:  email.InReplyTo,
		References: email.References,
		Subject:    email.Subject,
		From:       email.From,
		To:         email.To,
		Cc:         email.CC,
		Date:       email.CreatedAt,
	}

	// Извлекаем дополнительные заголовки из raw headers
	if err := headers.extractAdditionalHeaders(email.Headers); err != nil {
		return nil, fmt.Errorf("failed to extract additional headers: %w", err)
	}

	// Валидация обязательных полей
	if err := headers.Validate(); err != nil {
		return nil, fmt.Errorf("email headers validation failed: %w", err)
	}

	return headers, nil
}

// extractAdditionalHeaders извлекает дополнительные заголовки из raw headers
func (h *EmailHeaders) extractAdditionalHeaders(rawHeaders map[string][]string) error {
	for key, values := range rawHeaders {
		if !EssentialHeaderKeys[key] || len(values) == 0 {
			continue
		}

		value := strings.TrimSpace(values[0])

		switch key {
		case "Content-Type":
			h.ContentType = value
		case "Priority":
			h.Priority = value
		case "Importance":
			h.Importance = value
		case "Date":
			if parsed, err := h.parseDate(value); err == nil {
				h.Date = parsed
			}
		}
	}
	return nil
}

// parseDate парсит Date header в time.Time
func (h *EmailHeaders) parseDate(dateStr string) (time.Time, error) {
	dateStr = strings.TrimSpace(dateStr)

	// Пробуем различные форматы дат
	formats := []string{
		time.RFC1123Z,
		time.RFC1123,
		time.RFC822Z,
		time.RFC822,
		"Mon, 2 Jan 2006 15:04:05 -0700",
		"2 Jan 2006 15:04:05 -0700",
	}

	for _, format := range formats {
		if parsed, err := time.Parse(format, dateStr); err == nil {
			return parsed, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse date: %s", dateStr)
}

// Validate проверяет валидность EmailHeaders
func (h *EmailHeaders) Validate() error {
	if h.MessageID == "" {
		return fmt.Errorf("Message-ID is required")
	}
	if h.From == "" {
		return fmt.Errorf("From is required")
	}
	if h.Subject == "" {
		return fmt.Errorf("Subject is required")
	}
	return nil
}

// HasThreadingData проверяет есть ли данные для email threading
func (h *EmailHeaders) HasThreadingData() bool {
	return h.InReplyTo != "" || len(h.References) > 0
}

// GetThreadingData возвращает данные для email threading
func (h *EmailHeaders) GetThreadingData() (string, []string) {
	// Гарантируем что возвращаем non-nil слайс
	if h.References == nil {
		return h.InReplyTo, []string{}
	}
	return h.InReplyTo, h.References
}

// ToSourceMeta конвертирует EmailHeaders в map для сохранения в source_meta
func (h *EmailHeaders) ToSourceMeta() map[string]interface{} {
	meta := map[string]interface{}{
		"message_id":  h.MessageID,
		"in_reply_to": h.InReplyTo,
		"references":  h.References,
		"essential_headers": map[string]interface{}{
			"Message-ID":   h.MessageID,
			"In-Reply-To":  h.InReplyTo,
			"References":   h.References,
			"Subject":      h.Subject,
			"From":         string(h.From),
			"To":           h.addressesToStrings(h.To),
			"Cc":           h.addressesToStrings(h.Cc),
			"Date":         h.Date.Format(time.RFC3339),
			"Content-Type": h.ContentType,
		},
	}

	if h.Priority != "" {
		meta["priority"] = h.Priority
	}
	if h.Importance != "" {
		meta["importance"] = h.Importance
	}

	return meta
}

// addressesToStrings конвертирует []EmailAddress в []string
func (h *EmailHeaders) addressesToStrings(addresses []EmailAddress) []string {
	if len(addresses) == 0 {
		return nil
	}

	result := make([]string, len(addresses))
	for i, addr := range addresses {
		result[i] = string(addr)
	}
	return result
}

// String возвращает строковое представление для логирования
func (h *EmailHeaders) String() string {
	return fmt.Sprintf("EmailHeaders{MessageID: %s, Subject: %s, From: %s, Threading: %v}",
		h.MessageID, h.Subject, h.From, h.HasThreadingData())
}
