// backend/internal/infrastructure/email/header_filter.go
package email

import (
	"context"
	"fmt"

	"github.com/audetv/urms/internal/core/domain"
	"github.com/audetv/urms/internal/core/ports"
)

// HeaderFilter сервис для фильтрации и обработки email заголовков
type HeaderFilter struct {
	logger ports.Logger
}

// NewHeaderFilter создает новый HeaderFilter
func NewHeaderFilter(logger ports.Logger) *HeaderFilter {
	return &HeaderFilter{
		logger: logger,
	}
}

// FilterEssentialHeaders создает EmailHeaders из существующего EmailMessage
func (f *HeaderFilter) FilterEssentialHeaders(ctx context.Context, email *domain.EmailMessage) (*domain.EmailHeaders, error) {
	if email == nil {
		return nil, fmt.Errorf("email message cannot be nil")
	}

	f.logger.Debug(ctx, "Filtering essential headers from email message",
		"message_id", email.MessageID,
		"raw_headers_count", len(email.Headers),
		"essential_keys_count", len(domain.EssentialHeaderKeys))

	// Логируем все raw headers для диагностики (в debug режиме)
	for key, values := range email.Headers {
		f.logger.Debug(ctx, "Raw header found",
			"key", key,
			"values_count", len(values),
			"first_value_preview", f.getPreview(values[0], 50))
	}

	// Создаем EmailHeaders value object из EmailMessage
	emailHeaders, err := domain.NewEmailHeaders(email)
	if err != nil {
		f.logger.Error(ctx, "Failed to create email headers from email message",
			"message_id", email.MessageID,
			"error", err.Error())
		return nil, fmt.Errorf("failed to create email headers: %w", err)
	}

	f.logger.Info(ctx, "Essential headers filtered successfully",
		"message_id", emailHeaders.MessageID,
		"subject", emailHeaders.Subject,
		"from", emailHeaders.From,
		"has_threading_data", emailHeaders.HasThreadingData(),
		"references_count", len(emailHeaders.References),
		"filtered_headers_count", f.countFilteredHeaders(email.Headers))

	return emailHeaders, nil
}

// countFilteredHeaders подсчитывает сколько заголовков было отфильтровано
func (f *HeaderFilter) countFilteredHeaders(rawHeaders map[string][]string) int {
	count := 0
	for key := range rawHeaders {
		if domain.EssentialHeaderKeys[key] {
			count++
		}
	}
	return count
}

// getPreview вспомогательный метод для preview данных
func (f *HeaderFilter) getPreview(text string, length int) string {
	if text == "" {
		return "[empty]"
	}
	if len(text) <= length {
		return text
	}
	return text[:length] + "..."
}

// SanitizeHeaders удаляет sensitive information из заголовков
func (f *HeaderFilter) SanitizeHeaders(ctx context.Context, rawHeaders map[string][]string) map[string][]string {
	sanitized := make(map[string][]string)

	// Список sensitive headers которые нужно удалить
	sensitiveHeaders := map[string]bool{
		"Received":                   true,
		"Return-Path":                true,
		"Authentication-Results":     true,
		"DKIM-Signature":             true,
		"ARC-Seal":                   true,
		"ARC-Message-Signature":      true,
		"ARC-Authentication-Results": true,
		"X-Google-DKIM-Signature":    true,
		"X-Gm-Message-State":         true,
		"X-Received":                 true,
		"X-Forwarded-For":            true,
		"X-Originating-IP":           true,
		"X-Mailer":                   true,
		"X-Priority":                 true,
		"X-MS-Mail-Priority":         true,
		"X-MSMail-Priority":          true,
	}

	for key, values := range rawHeaders {
		// Пропускаем sensitive headers
		if sensitiveHeaders[key] {
			f.logger.Debug(ctx, "Removed sensitive header",
				"header", key)
			continue
		}

		// Сохраняем остальные заголовки
		sanitized[key] = values
	}

	f.logger.Debug(ctx, "Headers sanitization completed",
		"original_count", len(rawHeaders),
		"sanitized_count", len(sanitized),
		"removed_count", len(rawHeaders)-len(sanitized))

	return sanitized
}

// ExtractThreadingData извлекает данные для email threading
func (f *HeaderFilter) ExtractThreadingData(ctx context.Context, headers *domain.EmailHeaders) (string, []string, error) {
	if headers == nil {
		return "", nil, fmt.Errorf("headers cannot be nil")
	}

	inReplyTo, references := headers.GetThreadingData()

	f.logger.Debug(ctx, "Threading data extracted",
		"message_id", headers.MessageID,
		"in_reply_to", inReplyTo,
		"references_count", len(references),
		"references", references)

	return inReplyTo, references, nil
}
