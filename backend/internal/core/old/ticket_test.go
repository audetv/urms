// internal/core/domain/ticket_test.go
package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTicket(t *testing.T) {
	ticket, err := NewTicket(
		"Test Subject",
		"Test Description",
		SourceEmail,
		"customer-123",
		"user-456",
	)

	require.NoError(t, err)
	assert.NotEmpty(t, ticket.ID)
	assert.Equal(t, "Test Subject", ticket.Subject)
	assert.Equal(t, StatusOpen, ticket.Status)
	assert.Equal(t, PriorityMedium, ticket.Priority)
	assert.Equal(t, SourceEmail, ticket.Source)
	assert.Len(t, ticket.Participants, 1)
	assert.Equal(t, "user-456", ticket.Participants[0].UserID)
}

func TestTicket_AddMessage(t *testing.T) {
	ticket, _ := NewTicket("Test", "Desc", SourceEmail, "cust-1", "user-1")

	err := ticket.AddMessage("user-2", "Test message", MessageTypeInternal)
	require.NoError(t, err)

	assert.Len(t, ticket.Messages, 1)
	assert.Equal(t, "Test message", ticket.Messages[0].Content)
	assert.Len(t, ticket.Participants, 2) // Автор + новый участник
}

func TestTicket_ChangeStatus(t *testing.T) {
	ticket, _ := NewTicket("Test", "Desc", SourceEmail, "cust-1", "user-1")

	// Valid transition
	err := ticket.ChangeStatus(StatusInProgress)
	require.NoError(t, err)
	assert.Equal(t, StatusInProgress, ticket.Status)

	// Invalid transition
	err = ticket.ChangeStatus(StatusOpen) // Можно вернуться к Open
	require.NoError(t, err)

	// Should add system message
	assert.Len(t, ticket.Messages, 2) // 2 системных сообщения о смене статуса
}

func TestTicket_Assign(t *testing.T) {
	ticket, _ := NewTicket("Test", "Desc", SourceEmail, "cust-1", "user-1")

	err := ticket.Assign("user-789")
	require.NoError(t, err)

	assert.Equal(t, "user-789", ticket.AssigneeID)
	assert.Len(t, ticket.Participants, 2) // Reporter + Assignee
}

func TestTicket_AddTag(t *testing.T) {
	ticket, _ := NewTicket("Test", "Desc", SourceEmail, "cust-1", "user-1")

	ticket.AddTag("urgent")
	ticket.AddTag("bug")
	ticket.AddTag("urgent") // Duplicate

	assert.Len(t, ticket.Tags, 2)
	assert.Contains(t, ticket.Tags, "urgent")
	assert.Contains(t, ticket.Tags, "bug")
}

func TestTicket_CreateSubTicket(t *testing.T) {
	ticket, _ := NewTicket("Parent", "Desc", SourceEmail, "cust-1", "user-1")

	subTicket, err := ticket.CreateSubTicket("Sub task", "Sub desc", "user-2")
	require.NoError(t, err)

	assert.NotEmpty(t, subTicket.ID)
	assert.Equal(t, ticket.ID, subTicket.ParentID)
	assert.Equal(t, "Sub task", subTicket.Subject)
	assert.Len(t, ticket.SubTickets, 1)
}
