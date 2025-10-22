// backend/internal/infrastructure/email/mime_parser.go

package email

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime"
	"strings"

	"github.com/emersion/go-message"

	"github.com/audetv/urms/internal/core/domain"
	"github.com/audetv/urms/internal/core/ports"
)

// MIMEParser парсер MIME сообщений
type MIMEParser struct {
	logger ports.Logger
}

// NewMIMEParser создает новый MIME парсер
func NewMIMEParser(logger ports.Logger) *MIMEParser {
	return &MIMEParser{
		logger: logger,
	}
}

// ParseMessage парсит RFC 5322 сообщение
func (p *MIMEParser) ParseMessage(rawMessage []byte) (*ParsedMessage, error) {
	result := &ParsedMessage{
		Text:        "",
		HTML:        "",
		Attachments: []domain.Attachment{},
		Headers:     make(map[string][]string),
	}

	if len(rawMessage) == 0 {
		p.logger.Debug(context.Background(), "MIME parser: empty raw message")
		return result, nil
	}

	// ✅ ОПТИМИЗИРУЕМ: Одна строка вместо трех диагностических
	p.logger.Debug(context.Background(), "Starting MIME parsing",
		"raw_data_length", len(rawMessage))

	// Создаем reader для MIME парсера
	reader := bytes.NewReader(rawMessage)

	// Парсим MIME сообщение
	entity, err := message.Read(reader)
	if err != nil {
		p.logger.Error(context.Background(), "MIME parsing failed",
			"error", err.Error())
		return nil, fmt.Errorf("failed to parse MIME message: %w", err)
	}

	// ✅ УБИРАЕМ диагностику структуры entity - это техническая деталь

	// Извлекаем заголовки
	p.extractHeaders(entity, result)

	// Обрабатываем тело сообщения
	err = p.parseEntity(entity, result)
	if err != nil {
		p.logger.Error(context.Background(), "Failed to parse message body",
			"error", err.Error())
		return nil, fmt.Errorf("failed to parse message body: %w", err)
	}

	// ✅ ОПТИМИЗИРУЕМ финальное логирование - только ключевые метрики
	p.logger.Debug(context.Background(), "MIME parsing completed",
		"text_length", len(result.Text),
		"html_length", len(result.HTML),
		"attachments_count", len(result.Attachments))

	return result, nil
}

// extractHeaders извлекает заголовки из MIME entity
func (p *MIMEParser) extractHeaders(entity *message.Entity, result *ParsedMessage) {
	// Копируем все заголовки
	for field := entity.Header.Fields(); field.Next(); {
		key := field.Key()
		value := field.Value()
		result.Headers[key] = append(result.Headers[key], value)
	}
}

// parseEntity рекурсивно парсит MIME entity
func (p *MIMEParser) parseEntity(entity *message.Entity, result *ParsedMessage) error {
	contentType := entity.Header.Get("Content-Type")
	contentDisposition := entity.Header.Get("Content-Disposition")

	// Проверяем multipart сообщение
	if mr := entity.MultipartReader(); mr != nil {
		return p.parseMultipart(mr, result)
	}

	// Обрабатываем одиночную часть
	return p.parseSinglePart(entity, contentType, contentDisposition, result)
}

// parseMultipart парсит multipart сообщение
func (p *MIMEParser) parseMultipart(mr message.MultipartReader, result *ParsedMessage) error {
	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read multipart part: %w", err)
		}

		contentType := part.Header.Get("Content-Type")
		contentDisposition := part.Header.Get("Content-Disposition")

		// Рекурсивно парсим части
		if err := p.parsePart(part, contentType, contentDisposition, result); err != nil {
			return err
		}
	}

	return nil
}

// parsePart парсит отдельную часть сообщения
func (p *MIMEParser) parsePart(part *message.Entity, contentType, contentDisposition string, result *ParsedMessage) error {
	// Если это вложенный multipart, обрабатываем рекурсивно
	if mr := part.MultipartReader(); mr != nil {
		return p.parseMultipart(mr, result)
	}

	// Обрабатываем одиночную часть
	return p.parseSinglePart(part, contentType, contentDisposition, result)
}

// parseSinglePart парсит одиночную часть сообщения
func (p *MIMEParser) parseSinglePart(part *message.Entity, contentType, contentDisposition string, result *ParsedMessage) error {
	// Читаем данные части
	data, err := io.ReadAll(part.Body)
	if err != nil {
		p.logger.Warn(context.Background(), "Failed to read part body",
			"content_type", contentType,
			"error", err.Error())
		return fmt.Errorf("failed to read part body: %w", err)
	}

	// ✅ УБИРАЕМ диагностику данных части - это основной источник шума
	// Вместо 3+ строк на каждую часть - ничего (или одна строка в summary)

	// Определяем тип контента
	isAttachment := strings.Contains(strings.ToLower(contentDisposition), "attachment")

	if isAttachment {
		// Это вложение
		filename := p.extractFilename(part.Header, contentType)
		attachment := domain.Attachment{
			Name:        filename,
			ContentType: contentType,
			Size:        int64(len(data)),
			Data:        data,
		}
		result.Attachments = append(result.Attachments, attachment)

		// ✅ Логируем только факт добавления вложения (если нужно)
		p.logger.Debug(context.Background(), "Attachment added",
			"name", filename,
			"size", len(data))
	} else if strings.Contains(contentType, "text/plain") && result.Text == "" {
		// Текстовое тело (берем только первое найденное)
		result.Text = string(data)
	} else if strings.Contains(contentType, "text/html") && result.HTML == "" {
		// HTML тело (берем только первое найденное)
		result.HTML = string(data)
	} else if contentType == "" && result.Text == "" {
		// Если тип не указан, пробуем как текст
		result.Text = string(data)
	}

	return nil
}

// extractFilename извлекает имя файла из заголовков
func (p *MIMEParser) extractFilename(header message.Header, contentType string) string {
	// Пробуем Content-Disposition
	if disposition := header.Get("Content-Disposition"); disposition != "" {
		_, params, err := mime.ParseMediaType(disposition)
		if err == nil {
			if filename, exists := params["filename"]; exists {
				return filename
			}
		}
	}

	// Пробуем Content-Type
	if contentType != "" {
		_, params, err := mime.ParseMediaType(contentType)
		if err == nil {
			if name, exists := params["name"]; exists {
				return name
			}
		}
	}

	return "unknown"
}

// ParsedMessage результат парсинга сообщения
type ParsedMessage struct {
	Text        string
	HTML        string
	Attachments []domain.Attachment
	Headers     map[string][]string
}
