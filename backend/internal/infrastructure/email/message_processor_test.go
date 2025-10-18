// internal/infrastructure/email/message_processor_test.go
package email

import (
	"context"
	"testing"
	"time"

	"github.com/audetv/urms/internal/core/domain"
	"github.com/audetv/urms/internal/infrastructure/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultMessageProcessor(t *testing.T) {
	logger := logging.NewTestLogger()
	processor := NewDefaultMessageProcessor(logger)
	ctx := context.Background()

	t.Run("ProcessIncomingEmail_Success", func(t *testing.T) {
		email := domain.EmailMessage{
			MessageID: "test-message-123",
			From:      domain.EmailAddress("customer@example.com"),
			To:        []domain.EmailAddress{"support@company.com"},
			Subject:   "Test Support Request",
			BodyText:  "I need help with my account",
			CreatedAt: time.Now(),
		}

		err := processor.ProcessIncomingEmail(ctx, email)
		require.NoError(t, err)
	})

	t.Run("ProcessIncomingEmail_ValidationError", func(t *testing.T) {
		// Email без MessageID должен вернуть ошибку
		email := domain.EmailMessage{
			From: domain.EmailAddress("customer@example.com"),
			To:   []domain.EmailAddress{"support@company.com"},
		}

		err := processor.ProcessIncomingEmail(ctx, email)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "message ID is required")
	})

	t.Run("ProcessOutgoingEmail_Success", func(t *testing.T) {
		email := domain.EmailMessage{
			MessageID: "outgoing-test-456",
			From:      domain.EmailAddress("support@company.com"),
			To:        []domain.EmailAddress{"customer@example.com"},
			Subject:   "Re: Test Support Request",
			BodyText:  "We're looking into your issue",
			CreatedAt: time.Now(),
		}

		err := processor.ProcessOutgoingEmail(ctx, email)
		require.NoError(t, err)
	})

	t.Run("ProcessOutgoingEmail_ValidationError", func(t *testing.T) {
		// Email без получателей должен вернуть ошибку
		email := domain.EmailMessage{
			MessageID: "invalid-outgoing",
			From:      domain.EmailAddress("support@company.com"),
			Subject:   "No recipients",
		}

		err := processor.ProcessOutgoingEmail(ctx, email)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "at least one recipient is required")
	})

	t.Run("ProcessOutgoingEmail_NoSubject", func(t *testing.T) {
		// Email без темы должен вернуть ошибку
		email := domain.EmailMessage{
			MessageID: "no-subject-email",
			From:      domain.EmailAddress("support@company.com"),
			To:        []domain.EmailAddress{"customer@example.com"},
			// Subject is empty
		}

		err := processor.ProcessOutgoingEmail(ctx, email)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "subject is required")
	})
}

func TestDefaultMessageProcessor_Analysis(t *testing.T) {
	logger := logging.NewTestLogger()
	processor := NewDefaultMessageProcessor(logger)
	ctx := context.Background()

	t.Run("EmailWithAttachments", func(t *testing.T) {
		email := domain.EmailMessage{
			MessageID: "with-attachments",
			From:      domain.EmailAddress("customer@example.com"),
			To:        []domain.EmailAddress{"support@company.com"},
			Subject:   "Email with attachments",
			BodyText:  "Please see attached files",
			Attachments: []domain.Attachment{
				{
					Name:        "log.txt",
					Size:        1024,
					ContentType: "text/plain",
				},
				{
					Name:        "screenshot.png",
					Size:        2048,
					ContentType: "image/png",
				},
			},
			CreatedAt: time.Now(),
		}

		err := processor.ProcessIncomingEmail(ctx, email)
		require.NoError(t, err)
	})

	t.Run("LongContentEmail", func(t *testing.T) {
		longText := ""
		for i := 0; i < 1500; i++ {
			longText += "This is a long email content. "
		}

		email := domain.EmailMessage{
			MessageID: "long-content",
			From:      domain.EmailAddress("customer@example.com"),
			To:        []domain.EmailAddress{"support@company.com"},
			Subject:   "Very detailed issue description",
			BodyText:  longText,
			CreatedAt: time.Now(),
		}

		err := processor.ProcessIncomingEmail(ctx, email)
		require.NoError(t, err)
	})

	t.Run("HTMLContentEmail", func(t *testing.T) {
		email := domain.EmailMessage{
			MessageID: "html-email",
			From:      domain.EmailAddress("customer@example.com"),
			To:        []domain.EmailAddress{"support@company.com"},
			Subject:   "HTML Email",
			BodyHTML:  "<html><body><h1>Hello</h1><p>This is HTML content</p></body></html>",
			CreatedAt: time.Now(),
		}

		err := processor.ProcessIncomingEmail(ctx, email)
		require.NoError(t, err)
	})
}

// Тест для проверки интеграции с реальным логгером
func TestDefaultMessageProcessor_Integration(t *testing.T) {
	// Используем реальный логгер вместо test логгера
	logger := logging.NewZerologLogger("info", "console")
	processor := NewDefaultMessageProcessor(logger)
	ctx := context.Background()

	t.Run("RealLoggerIntegration", func(t *testing.T) {
		email := domain.EmailMessage{
			MessageID: "integration-test",
			From:      domain.EmailAddress("test@example.com"),
			To:        []domain.EmailAddress{"support@company.com"},
			Subject:   "Integration Test",
			BodyText:  "Testing with real logger",
			CreatedAt: time.Now(),
		}

		err := processor.ProcessIncomingEmail(ctx, email)
		require.NoError(t, err)
	})
}
