// internal/core/services/task_service_test.go
package services_test

import (
	"context"
	"testing"
	"time"

	"github.com/audetv/urms/internal/core/domain"
	"github.com/audetv/urms/internal/core/ports"
	"github.com/audetv/urms/internal/core/services"
	"github.com/audetv/urms/internal/infrastructure/persistence/task/inmemory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTaskService_CreateTask(t *testing.T) {
	ctx := context.Background()
	logger := &services.MockLogger{}
	taskRepo := inmemory.NewTaskRepository(logger)
	customerRepo := inmemory.NewCustomerRepository(logger)
	userRepo := inmemory.NewUserRepository(logger)

	taskService := services.NewTaskService(taskRepo, customerRepo, userRepo, logger)

	tests := []struct {
		name        string
		request     ports.CreateTaskRequest
		wantErr     bool
		errContains string
	}{
		{
			name: "successfully create support task",
			request: ports.CreateTaskRequest{
				Type:        domain.TaskTypeSupport,
				Subject:     "Test Support Task",
				Description: "Test Description",
				CustomerID:  stringPtr("customer-1"),
				ReporterID:  "user-1",
				Source:      domain.SourceEmail,
				Priority:    domain.PriorityMedium,
				Category:    "technical",
			},
			wantErr: false,
		},
		{
			name: "successfully create internal task",
			request: ports.CreateTaskRequest{
				Type:        domain.TaskTypeInternal,
				Subject:     "Test Internal Task",
				Description: "Test Description",
				ReporterID:  "user-1",
				Priority:    domain.PriorityHigh,
				Category:    "development",
			},
			wantErr: false,
		},
		{
			name: "fail with empty subject",
			request: ports.CreateTaskRequest{
				Type:        domain.TaskTypeSupport,
				Subject:     "",
				Description: "Test Description",
				ReporterID:  "user-1",
			},
			wantErr:     true,
			errContains: "subject is required",
		},
		{
			name: "fail with empty description",
			request: ports.CreateTaskRequest{
				Type:        domain.TaskTypeSupport,
				Subject:     "Test Subject",
				Description: "",
				ReporterID:  "user-1",
			},
			wantErr:     true,
			errContains: "description is required",
		},
		{
			name: "fail with empty reporter",
			request: ports.CreateTaskRequest{
				Type:        domain.TaskTypeSupport,
				Subject:     "Test Subject",
				Description: "Test Description",
				ReporterID:  "",
			},
			wantErr:     true,
			errContains: "reporter ID is required",
		},
		{
			name: "fail with invalid task type",
			request: ports.CreateTaskRequest{
				Type:        domain.TaskType("invalid"),
				Subject:     "Test Subject",
				Description: "Test Description",
				ReporterID:  "user-1",
			},
			wantErr:     true,
			errContains: "invalid task type",
		},
		{
			name: "fail to create subtask without parent",
			request: ports.CreateTaskRequest{
				Type:        domain.TaskTypeSubTask,
				Subject:     "Test Subtask",
				Description: "Test Description",
				ReporterID:  "user-1",
			},
			wantErr:     true,
			errContains: "parent ID is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task, err := taskService.CreateTask(ctx, tt.request)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}

			require.NoError(t, err)
			require.NotNil(t, task)
			assert.Equal(t, tt.request.Subject, task.Subject)
			assert.Equal(t, tt.request.Description, task.Description)
			assert.Equal(t, tt.request.Type, task.Type)
			assert.Equal(t, domain.TaskStatusOpen, task.Status)
			assert.NotEmpty(t, task.ID)
			assert.NotZero(t, task.CreatedAt)
			assert.NotZero(t, task.UpdatedAt)
		})
	}
}

func TestTaskService_CreateSpecificTaskTypes(t *testing.T) {
	ctx := context.Background()
	logger := &services.MockLogger{}
	taskRepo := inmemory.NewTaskRepository(logger)
	customerRepo := inmemory.NewCustomerRepository(logger)
	userRepo := inmemory.NewUserRepository(logger)

	taskService := services.NewTaskService(taskRepo, customerRepo, userRepo, logger)

	t.Run("create support task", func(t *testing.T) {
		req := ports.CreateSupportTaskRequest{
			Subject:     "Support Request",
			Description: "Need help with product",
			CustomerID:  "customer-1",
			ReporterID:  "user-1",
			Source:      domain.SourceEmail,
			Priority:    domain.PriorityHigh,
			Category:    "support",
		}

		task, err := taskService.CreateSupportTask(ctx, req)
		require.NoError(t, err)
		assert.Equal(t, domain.TaskTypeSupport, task.Type)
		assert.Equal(t, "customer-1", *task.CustomerID)
		assert.Equal(t, domain.SourceEmail, task.Source)
	})

	t.Run("create internal task", func(t *testing.T) {
		req := ports.CreateInternalTaskRequest{
			Subject:     "Internal Task",
			Description: "Internal development task",
			ReporterID:  "user-2",
			Priority:    domain.PriorityMedium,
			Category:    "development",
		}

		task, err := taskService.CreateInternalTask(ctx, req)
		require.NoError(t, err)
		assert.Equal(t, domain.TaskTypeInternal, task.Type)
		assert.Nil(t, task.CustomerID)
	})

	t.Run("create subtask", func(t *testing.T) {
		// Сначала создаем родительскую задачу
		parentReq := ports.CreateInternalTaskRequest{
			Subject:     "Parent Task",
			Description: "Parent task description",
			ReporterID:  "user-1",
		}
		parentTask, err := taskService.CreateInternalTask(ctx, parentReq)
		require.NoError(t, err)

		// Создаем подзадачу
		subtaskReq := ports.CreateSubTaskRequest{
			ParentID:    parentTask.ID,
			Subject:     "Subtask",
			Description: "Subtask description",
			ReporterID:  "user-1",
		}

		subtask, err := taskService.CreateSubTask(ctx, subtaskReq)
		require.NoError(t, err)
		assert.Equal(t, domain.TaskTypeSubTask, subtask.Type)
		assert.Equal(t, parentTask.ID, *subtask.ParentID)
	})
}

func TestTaskService_GetAndUpdateTask(t *testing.T) {
	ctx := context.Background()
	logger := &services.MockLogger{}
	taskRepo := inmemory.NewTaskRepository(logger)
	customerRepo := inmemory.NewCustomerRepository(logger)
	userRepo := inmemory.NewUserRepository(logger)

	taskService := services.NewTaskService(taskRepo, customerRepo, userRepo, logger)

	// Создаем задачу для тестов
	createReq := ports.CreateTaskRequest{
		Type:        domain.TaskTypeInternal,
		Subject:     "Test Task",
		Description: "Test Description",
		ReporterID:  "user-1",
		Priority:    domain.PriorityMedium,
		Category:    "test",
	}

	createdTask, err := taskService.CreateTask(ctx, createReq)
	require.NoError(t, err)

	t.Run("get existing task", func(t *testing.T) {
		task, err := taskService.GetTask(ctx, createdTask.ID)
		require.NoError(t, err)
		assert.Equal(t, createdTask.ID, task.ID)
		assert.Equal(t, createdTask.Subject, task.Subject)
	})

	t.Run("get non-existing task", func(t *testing.T) {
		task, err := taskService.GetTask(ctx, "non-existing-id")
		require.Error(t, err)
		assert.Nil(t, task)
		assert.Contains(t, err.Error(), "failed to get task")
	})

	t.Run("update task", func(t *testing.T) {
		newSubject := "Updated Subject"
		newPriority := domain.PriorityHigh
		updateReq := ports.UpdateTaskRequest{
			Subject:  &newSubject,
			Priority: &newPriority,
		}

		// Сохраняем время до обновления
		timeBeforeUpdate := time.Now()
		time.Sleep(1 * time.Millisecond) // Добавляем небольшую задержку

		updatedTask, err := taskService.UpdateTask(ctx, createdTask.ID, updateReq)
		require.NoError(t, err)
		assert.Equal(t, newSubject, updatedTask.Subject)
		assert.Equal(t, newPriority, updatedTask.Priority)

		// Проверяем, что время обновления изменилось и оно после нашего времени
		assert.True(t, updatedTask.UpdatedAt.After(timeBeforeUpdate) ||
			updatedTask.UpdatedAt.Equal(timeBeforeUpdate))
	})

	t.Run("delete task", func(t *testing.T) {
		err := taskService.DeleteTask(ctx, createdTask.ID)
		require.NoError(t, err)

		// Проверяем, что задача действительно удалена
		task, err := taskService.GetTask(ctx, createdTask.ID)
		require.Error(t, err)
		assert.Nil(t, task)
	})
}

func TestTaskService_ChangeStatus(t *testing.T) {
	ctx := context.Background()
	logger := &services.MockLogger{}
	taskRepo := inmemory.NewTaskRepository(logger)
	customerRepo := inmemory.NewCustomerRepository(logger)
	userRepo := inmemory.NewUserRepository(logger)

	taskService := services.NewTaskService(taskRepo, customerRepo, userRepo, logger)

	// Создаем задачу
	createReq := ports.CreateTaskRequest{
		Type:        domain.TaskTypeInternal,
		Subject:     "Status Test Task",
		Description: "Test Description",
		ReporterID:  "user-1",
	}

	task, err := taskService.CreateTask(ctx, createReq)
	require.NoError(t, err)
	assert.Equal(t, domain.TaskStatusOpen, task.Status)

	tests := []struct {
		name          string
		newStatus     domain.TaskStatus
		expectedError bool
	}{
		{
			name:          "change to in_progress",
			newStatus:     domain.TaskStatusInProgress,
			expectedError: false,
		},
		{
			name:          "change to resolved",
			newStatus:     domain.TaskStatusResolved,
			expectedError: false,
		},
		{
			name:          "change to closed",
			newStatus:     domain.TaskStatusClosed,
			expectedError: false,
		},
		{
			name:          "invalid transition from resolved to review",
			newStatus:     domain.TaskStatusReview,
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			timeBeforeChange := time.Now()
			time.Sleep(1 * time.Millisecond)
			updatedTask, err := taskService.ChangeStatus(ctx, task.ID, tt.newStatus, "user-1")

			if tt.expectedError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.newStatus, updatedTask.Status)
			assert.True(t, updatedTask.UpdatedAt.After(timeBeforeChange) ||
				updatedTask.UpdatedAt.Equal(timeBeforeChange))

			// Обновляем исходную задачу для следующей итерации
			task = updatedTask
		})
	}
}

func TestTaskService_AssignTask(t *testing.T) {
	ctx := context.Background()
	logger := &services.MockLogger{}
	taskRepo := inmemory.NewTaskRepository(logger)
	customerRepo := inmemory.NewCustomerRepository(logger)
	userRepo := inmemory.NewUserRepository(logger)

	taskService := services.NewTaskService(taskRepo, customerRepo, userRepo, logger)

	// Создаем задачу
	createReq := ports.CreateTaskRequest{
		Type:        domain.TaskTypeInternal,
		Subject:     "Assignment Test Task",
		Description: "Test Description",
		ReporterID:  "user-1",
	}

	task, err := taskService.CreateTask(ctx, createReq)
	require.NoError(t, err)
	assert.Empty(t, task.AssigneeID)

	t.Run("assign task to user", func(t *testing.T) {
		timeBeforeAssign := time.Now()
		time.Sleep(1 * time.Millisecond)
		assigneeID := "user-2"
		updatedTask, err := taskService.AssignTask(ctx, task.ID, assigneeID, "user-1")

		require.NoError(t, err)
		assert.Equal(t, assigneeID, updatedTask.AssigneeID)
		assert.True(t, updatedTask.UpdatedAt.After(timeBeforeAssign) ||
			updatedTask.UpdatedAt.Equal(timeBeforeAssign))
	})

	t.Run("fail to assign with empty assignee", func(t *testing.T) {
		updatedTask, err := taskService.AssignTask(ctx, task.ID, "", "user-1")
		require.Error(t, err)
		assert.Nil(t, updatedTask)
		assert.Contains(t, err.Error(), "assignee ID is required")
	})
}

func TestTaskService_AddMessage(t *testing.T) {
	ctx := context.Background()
	logger := &services.MockLogger{}
	taskRepo := inmemory.NewTaskRepository(logger)
	customerRepo := inmemory.NewCustomerRepository(logger)
	userRepo := inmemory.NewUserRepository(logger)

	taskService := services.NewTaskService(taskRepo, customerRepo, userRepo, logger)

	// Создаем задачу
	createReq := ports.CreateTaskRequest{
		Type:        domain.TaskTypeSupport,
		Subject:     "Message Test Task",
		Description: "Test Description",
		ReporterID:  "user-1",
		CustomerID:  stringPtr("customer-1"),
	}

	task, err := taskService.CreateTask(ctx, createReq)
	require.NoError(t, err)
	assert.Empty(t, task.Messages)

	t.Run("add customer message", func(t *testing.T) {
		messageReq := ports.AddMessageRequest{
			AuthorID:  "customer-1",
			Content:   "Hello, I need help!",
			Type:      domain.MessageTypeCustomer,
			IsPrivate: false,
		}

		updatedTask, err := taskService.AddMessage(ctx, task.ID, messageReq)
		require.NoError(t, err)
		assert.Len(t, updatedTask.Messages, 1)
		assert.Equal(t, messageReq.Content, updatedTask.Messages[0].Content)
		assert.Equal(t, messageReq.AuthorID, updatedTask.Messages[0].AuthorID)
	})

	t.Run("add internal note", func(t *testing.T) {
		updatedTask, err := taskService.AddInternalNote(ctx, task.ID, "user-2", "Internal investigation note")
		require.NoError(t, err)
		assert.Len(t, updatedTask.Messages, 2)

		lastMessage := updatedTask.Messages[1]
		assert.Equal(t, "Internal investigation note", lastMessage.Content)
		assert.Equal(t, domain.MessageTypeInternal, lastMessage.Type)
	})
}

func TestTaskService_SearchTasks(t *testing.T) {
	ctx := context.Background()
	logger := &services.MockLogger{}
	taskRepo := inmemory.NewTaskRepository(logger)
	customerRepo := inmemory.NewCustomerRepository(logger)
	userRepo := inmemory.NewUserRepository(logger)

	taskService := services.NewTaskService(taskRepo, customerRepo, userRepo, logger)

	// Создаем несколько задач для тестирования поиска
	tasks := []ports.CreateTaskRequest{
		{
			Type:        domain.TaskTypeSupport,
			Subject:     "High Priority Bug",
			Description: "Critical bug in production",
			ReporterID:  "user-1",
			CustomerID:  stringPtr("customer-1"),
			Priority:    domain.PriorityHigh,
			Category:    "bugs",
		},
		{
			Type:        domain.TaskTypeInternal,
			Subject:     "Feature Development",
			Description: "Develop new feature",
			ReporterID:  "user-2",
			Priority:    domain.PriorityMedium,
			Category:    "development",
		},
		{
			Type:        domain.TaskTypeSupport,
			Subject:     "General Question",
			Description: "Customer has a question",
			ReporterID:  "user-1",
			CustomerID:  stringPtr("customer-2"),
			Priority:    domain.PriorityLow,
			Category:    "questions",
		},
	}

	for _, req := range tasks {
		_, err := taskService.CreateTask(ctx, req)
		require.NoError(t, err)
	}

	t.Run("search by type", func(t *testing.T) {
		query := ports.TaskQuery{
			Types: []domain.TaskType{domain.TaskTypeSupport},
			Limit: 10,
		}

		result, err := taskService.SearchTasks(ctx, query)
		require.NoError(t, err)
		assert.Len(t, result.Tasks, 2)
		for _, task := range result.Tasks {
			assert.Equal(t, domain.TaskTypeSupport, task.Type)
		}
	})

	t.Run("search by priority", func(t *testing.T) {
		query := ports.TaskQuery{
			Priorities: []domain.Priority{domain.PriorityHigh},
			Limit:      10,
		}

		result, err := taskService.SearchTasks(ctx, query)
		require.NoError(t, err)
		assert.Len(t, result.Tasks, 1)
		assert.Equal(t, domain.PriorityHigh, result.Tasks[0].Priority)
	})

	t.Run("search with pagination", func(t *testing.T) {
		query := ports.TaskQuery{
			Limit:  2,
			Offset: 0,
		}

		result, err := taskService.SearchTasks(ctx, query)
		require.NoError(t, err)
		assert.Len(t, result.Tasks, 2)
		assert.Equal(t, 3, result.TotalCount)
		assert.Equal(t, 2, result.TotalPages)
	})
}

// Вспомогательная функция
func stringPtr(s string) *string {
	return &s
}
