package domain_test

import (
	"testing"

	"github.com/audetv/urms/internal/core/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockIDGenerator для тестирования
type MockIDGenerator struct {
	FixedID string
}

func (m *MockIDGenerator) GenerateID() string {
	if m.FixedID != "" {
		return m.FixedID
	}
	return "test-id-123"
}

func (m *MockIDGenerator) GenerateMessageID() string {
	return "<test-message-id@urms.local>"
}

func (m *MockIDGenerator) GenerateThreadID() string {
	return "thread-test-123"
}

func TestNewIncomingEmail(t *testing.T) {
	mockID := &MockIDGenerator{}

	t.Run("successful creation", func(t *testing.T) {
		from := domain.EmailAddress("test@example.com")
		to := []domain.EmailAddress{"support@company.com"}
		subject := "Test Subject"
		messageID := "<original-message-id@example.com>"

		email, err := domain.NewIncomingEmail(from, to, subject, messageID, mockID)

		require.NoError(t, err)
		assert.Equal(t, from, email.From)
		assert.Equal(t, to, email.To)
		assert.Equal(t, subject, email.Subject)
		assert.Equal(t, messageID, email.MessageID)
		assert.Equal(t, domain.DirectionIncoming, email.Direction)
		assert.Equal(t, "imap", email.Source)
		assert.False(t, email.Processed)
	})

	t.Run("invalid sender email", func(t *testing.T) {
		_, err := domain.NewIncomingEmail(
			domain.EmailAddress("invalid-email"),
			[]domain.EmailAddress{"support@company.com"},
			"Subject",
			"message-id",
			mockID,
		)

		assert.Error(t, err)
		assert.Equal(t, domain.ErrInvalidEmailAddress, err)
	})

	t.Run("empty subject", func(t *testing.T) {
		_, err := domain.NewIncomingEmail(
			domain.EmailAddress("test@example.com"),
			[]domain.EmailAddress{"support@company.com"},
			"",
			"message-id",
			mockID,
		)

		assert.Error(t, err)
		assert.Equal(t, domain.ErrEmptySubject, err)
	})
}

func TestEmailMessage_Validate(t *testing.T) {
	mockID := &MockIDGenerator{}

	t.Run("valid email", func(t *testing.T) {
		email, err := domain.NewIncomingEmail(
			domain.EmailAddress("test@example.com"),
			[]domain.EmailAddress{"support@company.com"},
			"Test Subject",
			"message-id",
			mockID,
		)
		require.NoError(t, err)
		email.BodyText = "Test body content"

		err = email.Validate()
		assert.NoError(t, err)
	})

	t.Run("empty body", func(t *testing.T) {
		email, err := domain.NewIncomingEmail(
			domain.EmailAddress("test@example.com"),
			[]domain.EmailAddress{"support@company.com"},
			"Test Subject",
			"message-id",
			mockID,
		)
		require.NoError(t, err)
		// BodyText и BodyHTML остаются пустыми

		err = email.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "must have either text or HTML body")
	})
}

func TestEmailMessage_IsSpam(t *testing.T) {
	mockID := &MockIDGenerator{}
	policy := domain.EmailProcessingPolicy{
		SpamFilter: true,
		BlockedSenders: []domain.EmailAddress{
			"spam@example.com",
		},
	}

	t.Run("spam content", func(t *testing.T) {
		email, err := domain.NewIncomingEmail(
			domain.EmailAddress("test@example.com"),
			[]domain.EmailAddress{"support@company.com"},
			"Win a lottery now!",
			"message-id",
			mockID,
		)
		require.NoError(t, err)
		email.BodyText = "Click here to claim your prize"

		isSpam := email.IsSpam(policy)
		assert.True(t, isSpam)
	})

	t.Run("blocked sender", func(t *testing.T) {
		email, err := domain.NewIncomingEmail(
			domain.EmailAddress("spam@example.com"),
			[]domain.EmailAddress{"support@company.com"},
			"Normal Subject",
			"message-id",
			mockID,
		)
		require.NoError(t, err)
		email.BodyText = "Normal content"

		isSpam := email.IsSpam(policy)
		assert.True(t, isSpam)
	})

	t.Run("not spam", func(t *testing.T) {
		email, err := domain.NewIncomingEmail(
			domain.EmailAddress("customer@example.com"),
			[]domain.EmailAddress{"support@company.com"},
			"Help with my account",
			"message-id",
			mockID,
		)
		require.NoError(t, err)
		email.BodyText = "I need help with my account"

		isSpam := email.IsSpam(policy)
		assert.False(t, isSpam)
	})
}

func TestEmailMessage_MarkAsProcessed(t *testing.T) {
	mockID := &MockIDGenerator{}

	email, err := domain.NewIncomingEmail(
		domain.EmailAddress("test@example.com"),
		[]domain.EmailAddress{"support@company.com"},
		"Test Subject",
		"message-id",
		mockID,
	)
	require.NoError(t, err)

	ticketID := "ticket-123"
	email.MarkAsProcessed(&ticketID)

	assert.True(t, email.Processed)
	assert.NotZero(t, email.ProcessedAt)
	assert.Equal(t, &ticketID, email.RelatedTicketID)
}
