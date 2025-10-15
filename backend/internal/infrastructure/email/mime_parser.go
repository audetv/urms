package email

import (
	"github.com/audetv/urms/internal/core/domain"
)

// MIMEParser парсер MIME сообщений (временная реализация)
type MIMEParser struct{}

// NewMIMEParser создает новый MIME парсер
func NewMIMEParser() *MIMEParser {
	return &MIMEParser{}
}

// ParseMessage парсит RFC 5322 сообщение (временная заглушка)
func (p *MIMEParser) ParseMessage(rawMessage []byte) (*ParsedMessage, error) {
	// Временная реализация - возвращаем базовую структуру
	return &ParsedMessage{
		Text:        "",
		HTML:        "",
		Attachments: []domain.Attachment{},
		Headers:     make(map[string][]string),
	}, nil
}

// ParsedMessage результат парсинга сообщения
type ParsedMessage struct {
	Text        string
	HTML        string
	Attachments []domain.Attachment
	Headers     map[string][]string
}
