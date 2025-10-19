// internal/core/domain/task_test.go
package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTask(t *testing.T) {
	task, err := NewTask(
		TaskTypeSupport,
		"Test Subject",
		"Test Description",
		"user-456",
	)

	require.NoError(t, err)
	assert.NotEmpty(t, task.ID)
	assert.Equal(t, "Test Subject", task.Subject)
	assert.Equal(t, TaskStatusOpen, task.Status)
	assert.Equal(t, TaskTypeSupport, task.Type)
	assert.Equal(t, "user-456", task.Participants[0].UserID)
}

func TestNewSupportTask(t *testing.T) {
	task, err := NewSupportTask(
		"Support Subject",
		"Support Description",
		"customer-123",
		"user-456",
		SourceEmail,
	)

	require.NoError(t, err)
	assert.Equal(t, TaskTypeSupport, task.Type)
	assert.Equal(t, "customer-123", *task.CustomerID)
	assert.Equal(t, SourceEmail, task.Source)
}

func TestTask_AddMessage(t *testing.T) {
	task, _ := NewTask(TaskTypeInternal, "Test", "Desc", "user-1")

	err := task.AddMessage("user-2", "Test message", MessageTypeInternal)
	require.NoError(t, err)

	assert.Len(t, task.Messages, 1)
	assert.Equal(t, "Test message", task.Messages[0].Content)
	assert.Len(t, task.Participants, 2) // Reporter + новый участник
}

func TestTask_ChangeStatus(t *testing.T) {
	task, _ := NewTask(TaskTypeInternal, "Test", "Desc", "user-1")

	// Valid transition
	err := task.ChangeStatus(TaskStatusInProgress, "user-1")
	require.NoError(t, err)
	assert.Equal(t, TaskStatusInProgress, task.Status)

	// Should add history event
	assert.Len(t, task.History, 2) // created + status_changed
}

func TestTask_Assign(t *testing.T) {
	task, _ := NewTask(TaskTypeInternal, "Test", "Desc", "user-1")

	err := task.Assign("user-789", "user-1")
	require.NoError(t, err)

	assert.Equal(t, "user-789", task.AssigneeID)
	assert.Len(t, task.Participants, 2) // Reporter + Assignee
	assert.Len(t, task.History, 2)      // created + assignee_changed
}

func TestTask_AddTag(t *testing.T) {
	task, _ := NewTask(TaskTypeInternal, "Test", "Desc", "user-1")

	task.AddTag("urgent")
	task.AddTag("bug")
	task.AddTag("urgent") // Duplicate

	assert.Len(t, task.Tags, 2)
	assert.Contains(t, task.Tags, "urgent")
	assert.Contains(t, task.Tags, "bug")
}

func TestNewSubTask(t *testing.T) {
	parentTask, _ := NewTask(TaskTypeSupport, "Parent", "Desc", "user-1")

	subTask, err := NewSubTask(parentTask.ID, "Sub task", "Sub desc", "user-2")
	require.NoError(t, err)

	assert.Equal(t, TaskTypeSubTask, subTask.Type)
	assert.Equal(t, parentTask.ID, *subTask.ParentID)
}
