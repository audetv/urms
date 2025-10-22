// backend/internal/infrastructure/email/header_filter_test.go
package email_test

import (
	"context"
	"testing"
	"time"

	"github.com/audetv/urms/internal/core/domain"
	"github.com/audetv/urms/internal/infrastructure/email"
	"github.com/audetv/urms/internal/infrastructure/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHeaderFilter_FilterEssentialHeaders(t *testing.T) {
	logger := logging.NewTestLogger()
	filter := email.NewHeaderFilter(logger)

	emailMsg := &domain.EmailMessage{
		MessageID:  "test-message@example.com",
		InReplyTo:  "parent@example.com",
		References: []string{"ref1@example.com"},
		Subject:    "Test Email",
		From:       "sender@example.com",
		To:         []domain.EmailAddress{"to@example.com"},
		CC:         []domain.EmailAddress{"cc@example.com"},
		CreatedAt:  time.Now(),
		Headers: map[string][]string{
			"Content-Type":           {"text/plain; charset=utf-8"},
			"Priority":               {"high"},
			"Importance":             {"high"},
			"Date":                   {time.Now().Format(time.RFC1123Z)},
			"Received":               {"from mail.example.com"},
			"X-Originating-IP":       {"192.168.1.1"},
			"DKIM-Signature":         {"test-signature"},
			"Authentication-Results": {"test-auth"},
		},
	}

	headers, err := filter.FilterEssentialHeaders(context.Background(), emailMsg)
	require.NoError(t, err)

	// Проверяем что essential headers сохранены
	assert.Equal(t, "test-message@example.com", headers.MessageID)
	assert.Equal(t, "Test Email", headers.Subject)
	assert.Equal(t, domain.EmailAddress("sender@example.com"), headers.From)
	assert.Equal(t, "text/plain; charset=utf-8", headers.ContentType)
	assert.Equal(t, "high", headers.Priority)

	// Проверяем что sensitive headers НЕ сохранены в EmailHeaders
	// (они остаются только в raw headers для диагностики)
}

func TestHeaderFilter_SanitizeHeaders(t *testing.T) {
	logger := logging.NewTestLogger()
	filter := email.NewHeaderFilter(logger)

	rawHeaders := map[string][]string{
		"Message-ID":       {"test@example.com"},
		"Subject":          {"Test"},
		"From":             {"sender@example.com"},
		"Received":         {"from mail.example.com"},
		"X-Originating-IP": {"192.168.1.1"},
		"DKIM-Signature":   {"test-signature"},
		"Content-Type":     {"text/plain"},
	}

	sanitized := filter.SanitizeHeaders(context.Background(), rawHeaders)

	// Проверяем что essential headers остались
	assert.Contains(t, sanitized, "Message-ID")
	assert.Contains(t, sanitized, "Subject")
	assert.Contains(t, sanitized, "From")
	assert.Contains(t, sanitized, "Content-Type")

	// Проверяем что sensitive headers удалены
	assert.NotContains(t, sanitized, "Received")
	assert.NotContains(t, sanitized, "X-Originating-IP")
	assert.NotContains(t, sanitized, "DKIM-Signature")

	assert.Len(t, sanitized, 4) // Only essential headers remain
}

func TestHeaderFilter_ExtractThreadingData(t *testing.T) {
	logger := logging.NewTestLogger()
	filter := email.NewHeaderFilter(logger)

	headers := &domain.EmailHeaders{
		MessageID:  "test@example.com",
		InReplyTo:  "parent@example.com",
		References: []string{"ref1@example.com", "ref2@example.com"},
		Subject:    "Test",
		From:       "sender@example.com",
	}

	inReplyTo, references, err := filter.ExtractThreadingData(context.Background(), headers)
	require.NoError(t, err)

	assert.Equal(t, "parent@example.com", inReplyTo)
	assert.Equal(t, []string{"ref1@example.com", "ref2@example.com"}, references)
}
