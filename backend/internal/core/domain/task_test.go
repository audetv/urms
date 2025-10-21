// internal/core/domain/task_test.go
package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTask(t *testing.T) {
	sourceMeta := map[string]interface{}{
		"message_id":  "<test@message.id>",
		"in_reply_to": "<parent@message.id>",
		"references":  []string{"<ref1>", "<ref2>"},
	}

	task, err := NewTask(
		TaskTypeSupport,
		"Test Subject",
		"Test Description",
		"user-456",
		sourceMeta, // ✅ Тестируем с SourceMeta
	)

	require.NoError(t, err)
	assert.NotEmpty(t, task.ID)
	assert.Equal(t, "Test Subject", task.Subject)
	assert.Equal(t, TaskStatusOpen, task.Status)
	assert.Equal(t, TaskTypeSupport, task.Type)
	assert.Equal(t, "user-456", task.Participants[0].UserID)
	assert.Equal(t, sourceMeta, task.SourceMeta) // ✅ Проверяем сохранение SourceMeta
}

func TestNewTask_WithNilSourceMeta(t *testing.T) {
	task, err := NewTask(
		TaskTypeInternal,
		"Test Subject",
		"Test Description",
		"user-456",
		nil, // ✅ Тестируем с nil SourceMeta
	)

	require.NoError(t, err)
	assert.NotNil(t, task.SourceMeta) // ✅ Должен создаться пустой map
	assert.Empty(t, task.SourceMeta)
}

func TestNewSupportTask(t *testing.T) {
	sourceMeta := map[string]interface{}{
		"message_id": "<email@message.id>",
		"headers":    map[string]interface{}{"X-IMAP-UID": "12345"},
	}

	task, err := NewSupportTask(
		"Support Subject",
		"Support Description",
		"customer-123",
		"user-456",
		SourceEmail,
		sourceMeta, // ✅ Тестируем с SourceMeta
	)

	require.NoError(t, err)
	assert.Equal(t, TaskTypeSupport, task.Type)
	assert.Equal(t, "customer-123", *task.CustomerID)
	assert.Equal(t, SourceEmail, task.Source)
	assert.Equal(t, sourceMeta, task.SourceMeta) // ✅ Проверяем сохранение SourceMeta
}

func TestNewSubTask(t *testing.T) {
	sourceMeta := map[string]interface{}{
		"parent_context": "main_task_123",
	}

	parentTask, _ := NewTask(TaskTypeSupport, "Parent", "Desc", "user-1", nil)

	subTask, err := NewSubTask(
		parentTask.ID,
		"Sub task",
		"Sub desc",
		"user-2",
		sourceMeta, // ✅ Тестируем с SourceMeta
	)

	require.NoError(t, err)

	assert.Equal(t, TaskTypeSubTask, subTask.Type)
	assert.Equal(t, parentTask.ID, *subTask.ParentID)
	assert.Equal(t, sourceMeta, subTask.SourceMeta) // ✅ Проверяем сохранение SourceMeta
}

func TestTask_AddMessage(t *testing.T) {
	task, _ := NewTask(TaskTypeInternal, "Test", "Desc", "user-1", nil)

	err := task.AddMessage("user-2", "Test message", MessageTypeInternal)
	require.NoError(t, err)

	assert.Len(t, task.Messages, 1)
	assert.Equal(t, "Test message", task.Messages[0].Content)
	assert.Len(t, task.Participants, 2) // Reporter + новый участник
}

func TestTask_ChangeStatus(t *testing.T) {
	task, _ := NewTask(TaskTypeInternal, "Test", "Desc", "user-1", nil)

	// Valid transition
	err := task.ChangeStatus(TaskStatusInProgress, "user-1")
	require.NoError(t, err)
	assert.Equal(t, TaskStatusInProgress, task.Status)

	// Should add history event
	assert.Len(t, task.History, 2) // created + status_changed
}

func TestTask_Assign(t *testing.T) {
	task, _ := NewTask(TaskTypeInternal, "Test", "Desc", "user-1", nil)

	err := task.Assign("user-789", "user-1")
	require.NoError(t, err)

	assert.Equal(t, "user-789", task.AssigneeID)
	assert.Len(t, task.Participants, 2) // Reporter + Assignee
	assert.Len(t, task.History, 2)      // created + assignee_changed
}

func TestTask_AddTag(t *testing.T) {
	task, _ := NewTask(TaskTypeInternal, "Test", "Desc", "user-1", nil)

	task.AddTag("urgent")
	task.AddTag("bug")
	task.AddTag("urgent") // Duplicate

	assert.Len(t, task.Tags, 2)
	assert.Contains(t, task.Tags, "urgent")
	assert.Contains(t, task.Tags, "bug")
}

// func TestTask_InvalidStatusTransition(t *testing.T) {
// 	task, _ := NewTask(TaskTypeInternal, "Test", "Desc", "user-1", nil)

// 	// Invalid transition: Open → Closed (should go through Resolved first)
// 	err := task.ChangeStatus(TaskStatusClosed, "user-1")
// 	assert.Error(t, err)
// 	assert.Equal(t, TaskStatusOpen, task.Status) // Status shouldn't change
// }

func TestTask_AddMessage_EmptyContent(t *testing.T) {
	task, _ := NewTask(TaskTypeInternal, "Test", "Desc", "user-1", nil)

	err := task.AddMessage("user-2", "", MessageTypeInternal)
	assert.Error(t, err)
	assert.Len(t, task.Messages, 0)
}

func TestGenerateIDs(t *testing.T) {
	taskID := GenerateTaskID()
	msgID := GenerateMessageID()
	eventID := GenerateEventID()

	assert.Contains(t, taskID, "TASK-")
	assert.Contains(t, msgID, "MSG-")
	assert.Contains(t, eventID, "EVT-")
}
