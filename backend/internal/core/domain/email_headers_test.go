// backend/internal/core/domain/email_headers_test.go
package domain_test

import (
	"testing"
	"time"

	"github.com/audetv/urms/internal/core/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEmailHeaders_NewEmailHeaders(t *testing.T) {
	email := &domain.EmailMessage{
		MessageID:  "test-message-id@example.com",
		InReplyTo:  "parent-message-id@example.com",
		References: []string{"ref1@example.com", "ref2@example.com"},
		Subject:    "Test Subject",
		From:       "sender@example.com",
		To:         []domain.EmailAddress{"to1@example.com", "to2@example.com"},
		CC:         []domain.EmailAddress{"cc@example.com"},
		CreatedAt:  time.Now(),
		Headers: map[string][]string{
			"Content-Type": {"text/plain"},
			"Priority":     {"high"},
			"Date":         {time.Now().Format(time.RFC1123Z)},
		},
	}

	headers, err := domain.NewEmailHeaders(email)
	require.NoError(t, err)

	assert.Equal(t, "test-message-id@example.com", headers.MessageID)
	assert.Equal(t, "parent-message-id@example.com", headers.InReplyTo)
	assert.Equal(t, []string{"ref1@example.com", "ref2@example.com"}, headers.References)
	assert.Equal(t, "Test Subject", headers.Subject)
	assert.Equal(t, domain.EmailAddress("sender@example.com"), headers.From)
	assert.Equal(t, "text/plain", headers.ContentType)
	assert.Equal(t, "high", headers.Priority)
	assert.True(t, headers.HasThreadingData())
}

func TestEmailHeaders_Validation(t *testing.T) {
	tests := []struct {
		name      string
		email     *domain.EmailMessage
		wantError bool
	}{
		{
			name: "valid headers",
			email: &domain.EmailMessage{
				MessageID: "test@example.com",
				Subject:   "Test",
				From:      "sender@example.com",
				CreatedAt: time.Now(),
				Headers:   map[string][]string{},
			},
			wantError: false,
		},
		{
			name: "missing message id",
			email: &domain.EmailMessage{
				Subject:   "Test",
				From:      "sender@example.com",
				CreatedAt: time.Now(),
				Headers:   map[string][]string{},
			},
			wantError: true,
		},
		{
			name: "missing subject",
			email: &domain.EmailMessage{
				MessageID: "test@example.com",
				From:      "sender@example.com",
				CreatedAt: time.Now(),
				Headers:   map[string][]string{},
			},
			wantError: true,
		},
		{
			name: "missing from",
			email: &domain.EmailMessage{
				MessageID: "test@example.com",
				Subject:   "Test",
				CreatedAt: time.Now(),
				Headers:   map[string][]string{},
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			headers, err := domain.NewEmailHeaders(tt.email)

			if tt.wantError {
				assert.Error(t, err)
				assert.Nil(t, headers)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, headers)
			}
		})
	}
}

func TestEmailHeaders_ToSourceMeta(t *testing.T) {
	email := &domain.EmailMessage{
		MessageID:  "test@example.com",
		InReplyTo:  "parent@example.com",
		References: []string{"ref1@example.com"},
		Subject:    "Test Subject",
		From:       "sender@example.com",
		To:         []domain.EmailAddress{"to@example.com"},
		CC:         []domain.EmailAddress{"cc@example.com"},
		CreatedAt:  time.Now(),
		Headers: map[string][]string{
			"Content-Type": {"text/plain"},
			"Priority":     {"high"},
		},
	}

	headers, err := domain.NewEmailHeaders(email)
	require.NoError(t, err)

	sourceMeta := headers.ToSourceMeta()

	assert.Equal(t, "test@example.com", sourceMeta["message_id"])
	assert.Equal(t, "parent@example.com", sourceMeta["in_reply_to"])
	assert.Equal(t, []string{"ref1@example.com"}, sourceMeta["references"])

	essentialHeaders := sourceMeta["essential_headers"].(map[string]interface{})
	assert.Equal(t, "sender@example.com", essentialHeaders["From"])
	assert.Equal(t, "Test Subject", essentialHeaders["Subject"])
	assert.Equal(t, "text/plain", essentialHeaders["Content-Type"])
}

func TestEmailHeaders_ThreadingData(t *testing.T) {
	tests := []struct {
		name           string
		email          *domain.EmailMessage
		wantInReplyTo  string
		wantReferences []string
		wantHasData    bool
	}{
		{
			name: "with threading data",
			email: &domain.EmailMessage{
				MessageID:  "test@example.com",
				InReplyTo:  "parent@example.com",
				References: []string{"ref1@example.com", "ref2@example.com"},
				Subject:    "Test",
				From:       "sender@example.com",
				CreatedAt:  time.Now(),
				Headers:    map[string][]string{},
			},
			wantInReplyTo:  "parent@example.com",
			wantReferences: []string{"ref1@example.com", "ref2@example.com"},
			wantHasData:    true,
		},
		{
			name: "without threading data",
			email: &domain.EmailMessage{
				MessageID: "test@example.com",
				Subject:   "Test",
				From:      "sender@example.com",
				CreatedAt: time.Now(),
				Headers:   map[string][]string{},
			},
			wantInReplyTo:  "",
			wantReferences: []string{},
			wantHasData:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			headers, err := domain.NewEmailHeaders(tt.email)
			require.NoError(t, err)

			inReplyTo, references := headers.GetThreadingData()
			assert.Equal(t, tt.wantInReplyTo, inReplyTo)
			assert.Equal(t, tt.wantReferences, references)
			assert.Equal(t, tt.wantHasData, headers.HasThreadingData())
		})
	}
}
