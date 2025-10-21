// internal/core/services/task_service.go
package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/audetv/urms/internal/core/domain"
	"github.com/audetv/urms/internal/core/ports"
)

type TaskService struct {
	taskRepo     ports.TaskRepository
	customerRepo ports.CustomerRepository
	userRepo     ports.UserRepository
	logger       ports.Logger
}

func NewTaskService(
	taskRepo ports.TaskRepository,
	customerRepo ports.CustomerRepository,
	userRepo ports.UserRepository,
	logger ports.Logger,
) *TaskService {
	return &TaskService{
		taskRepo:     taskRepo,
		customerRepo: customerRepo,
		userRepo:     userRepo,
		logger:       logger,
	}
}

// CreateTask создает новую задачу
func (s *TaskService) CreateTask(ctx context.Context, req ports.CreateTaskRequest) (*domain.Task, error) {
	if err := s.validateCreateTaskRequest(req); err != nil {
		return nil, err
	}

	var task *domain.Task
	var err error

	switch req.Type {
	case domain.TaskTypeSupport:
		if req.CustomerID == nil {
			return nil, errors.New("customer ID is required for support tasks")
		}
		task, err = domain.NewSupportTask(
			req.Subject,
			req.Description,
			*req.CustomerID,
			req.ReporterID,
			req.Source,
			req.SourceMeta, // ✅ ПЕРЕДАЕМ SourceMeta
		)
	case domain.TaskTypeInternal:
		task, err = domain.NewTask(
			domain.TaskTypeInternal,
			req.Subject,
			req.Description,
			req.ReporterID,
			req.SourceMeta, // ✅ ПЕРЕДАЕМ SourceMeta
		)
	case domain.TaskTypeSubTask:
		if req.ParentID == nil {
			return nil, errors.New("parent ID is required for subtasks")
		}
		task, err = domain.NewSubTask(
			*req.ParentID,
			req.Subject,
			req.Description,
			req.ReporterID,
			req.SourceMeta, // ✅ ПЕРЕДАЕМ SourceMeta
		)
	default:
		return nil, fmt.Errorf("unsupported task type: %s", req.Type)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create task: %w", err)
	}

	// Устанавливаем дополнительные поля
	if req.Priority != "" {
		task.Priority = req.Priority
	}
	if req.Category != "" {
		task.Category = req.Category
	}
	if len(req.Tags) > 0 {
		task.Tags = req.Tags
	}
	if req.ProjectID != nil {
		task.ProjectID = req.ProjectID
	}

	// Сохраняем задачу
	if err := s.taskRepo.Save(ctx, task); err != nil {
		return nil, fmt.Errorf("failed to save task: %w", err)
	}

	s.logger.Info(ctx, "task created",
		"task_id", task.ID,
		"type", task.Type,
		"reporter_id", task.ReporterID,
	)

	return task, nil
}

// CreateSupportTask создает задачу поддержки
func (s *TaskService) CreateSupportTask(ctx context.Context, req ports.CreateSupportTaskRequest) (*domain.Task, error) {
	createReq := ports.CreateTaskRequest{
		Type:        domain.TaskTypeSupport,
		Subject:     req.Subject,
		Description: req.Description,
		CustomerID:  &req.CustomerID,
		ReporterID:  req.ReporterID,
		Source:      req.Source,
		SourceMeta:  req.SourceMeta,
		Priority:    req.Priority,
		Category:    req.Category,
		Tags:        req.Tags,
	}

	return s.CreateTask(ctx, createReq)
}

// CreateInternalTask создает внутреннюю задачу
func (s *TaskService) CreateInternalTask(ctx context.Context, req ports.CreateInternalTaskRequest) (*domain.Task, error) {
	createReq := ports.CreateTaskRequest{
		Type:        domain.TaskTypeInternal,
		Subject:     req.Subject,
		Description: req.Description,
		ReporterID:  req.ReporterID,
		Priority:    req.Priority,
		Category:    req.Category,
		Tags:        req.Tags,
		ProjectID:   req.ProjectID,
	}

	return s.CreateTask(ctx, createReq)
}

// CreateSubTask создает подзадачу
func (s *TaskService) CreateSubTask(ctx context.Context, req ports.CreateSubTaskRequest) (*domain.Task, error) {
	createReq := ports.CreateTaskRequest{
		Type:        domain.TaskTypeSubTask,
		Subject:     req.Subject,
		Description: req.Description,
		ReporterID:  req.ReporterID,
		Priority:    req.Priority,
		Category:    req.Category,
		Tags:        req.Tags,
		ParentID:    &req.ParentID,
	}

	return s.CreateTask(ctx, createReq)
}

// GetTask возвращает задачу по ID
func (s *TaskService) GetTask(ctx context.Context, id string) (*domain.Task, error) {
	if id == "" {
		return nil, errors.New("task ID is required")
	}

	task, err := s.taskRepo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get task: %w", err)
	}

	return task, nil
}

// UpdateTask обновляет задачу
func (s *TaskService) UpdateTask(ctx context.Context, id string, req ports.UpdateTaskRequest) (*domain.Task, error) {
	task, err := s.taskRepo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to find task: %w", err)
	}

	// Сохраняем старое время для сравнения
	// oldUpdatedAt := task.UpdatedAt // ДОБАВИТЬ ЭТУ СТРОКУ

	// Обновляем поля если они предоставлены
	if req.Subject != nil {
		task.Subject = *req.Subject
	}
	if req.Description != nil {
		task.Description = *req.Description
	}
	if req.Priority != nil {
		task.Priority = *req.Priority
	}
	if req.Category != nil {
		task.Category = *req.Category
	}
	if req.Tags != nil {
		task.Tags = *req.Tags
	}
	if req.DueDate != nil {
		dueDate, err := time.Parse(time.RFC3339, *req.DueDate)
		if err != nil {
			return nil, fmt.Errorf("invalid due date format: %w", err)
		}
		task.DueDate = &dueDate
	}

	task.UpdatedAt = time.Now()

	if err := s.taskRepo.Update(ctx, task); err != nil {
		return nil, fmt.Errorf("failed to update task: %w", err)
	}

	s.logger.Info(ctx, "task updated", "task_id", task.ID)
	return task, nil
}

// DeleteTask удаляет задачу
func (s *TaskService) DeleteTask(ctx context.Context, id string) error {
	if id == "" {
		return errors.New("task ID is required")
	}

	// Проверяем существование задачи
	_, err := s.taskRepo.FindByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to find task: %w", err)
	}

	if err := s.taskRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete task: %w", err)
	}

	s.logger.Info(ctx, "task deleted", "task_id", id)
	return nil
}

// ChangeStatus изменяет статус задачи
func (s *TaskService) ChangeStatus(ctx context.Context, id string, status domain.TaskStatus, userID string) (*domain.Task, error) {
	task, err := s.taskRepo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to find task: %w", err)
	}

	if err := task.ChangeStatus(status, userID); err != nil {
		return nil, fmt.Errorf("failed to change status: %w", err)
	}

	// Явно обновляем UpdatedAt
	task.UpdatedAt = time.Now() // ДОБАВИТЬ ЭТУ СТРОКУ

	if err := s.taskRepo.Update(ctx, task); err != nil {
		return nil, fmt.Errorf("failed to update task: %w", err)
	}

	s.logger.Info(ctx, "task status changed",
		"task_id", task.ID,
		"status", status,
		"user_id", userID,
	)

	return task, nil
}

// AssignTask назначает исполнителя задачи
func (s *TaskService) AssignTask(ctx context.Context, id string, assigneeID string, userID string) (*domain.Task, error) {
	if assigneeID == "" {
		return nil, errors.New("assignee ID is required")
	}

	task, err := s.taskRepo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to find task: %w", err)
	}

	if err := task.Assign(assigneeID, userID); err != nil {
		return nil, fmt.Errorf("failed to assign task: %w", err)
	}

	// Явно обновляем UpdatedAt
	task.UpdatedAt = time.Now() // ДОБАВИТЬ ЭТУ СТРОКУ

	if err := s.taskRepo.Update(ctx, task); err != nil {
		return nil, fmt.Errorf("failed to update task: %w", err)
	}

	s.logger.Info(ctx, "task assigned",
		"task_id", task.ID,
		"assignee_id", assigneeID,
		"assigned_by", userID,
	)

	return task, nil
}

// AddMessage добавляет сообщение в задачу
func (s *TaskService) AddMessage(ctx context.Context, id string, req ports.AddMessageRequest) (*domain.Task, error) {
	task, err := s.taskRepo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to find task: %w", err)
	}

	messageType := domain.MessageTypeCustomer
	if req.IsPrivate {
		messageType = domain.MessageTypeInternal
	}

	if err := task.AddMessage(req.AuthorID, req.Content, messageType); err != nil {
		return nil, fmt.Errorf("failed to add message: %w", err)
	}

	if err := s.taskRepo.Update(ctx, task); err != nil {
		return nil, fmt.Errorf("failed to update task: %w", err)
	}

	s.logger.Info(ctx, "message added to task",
		"task_id", task.ID,
		"author_id", req.AuthorID,
		"message_type", messageType,
	)

	return task, nil
}

// AddInternalNote добавляет внутреннее сообщение
func (s *TaskService) AddInternalNote(ctx context.Context, id string, authorID, content string) (*domain.Task, error) {
	req := ports.AddMessageRequest{
		AuthorID:  authorID,
		Content:   content,
		IsPrivate: true,
	}

	return s.AddMessage(ctx, id, req)
}

// Добавляем метод FindBySourceMeta в TaskService
func (s *TaskService) FindBySourceMeta(ctx context.Context, meta map[string]interface{}) ([]domain.Task, error) {
	if len(meta) == 0 {
		return nil, fmt.Errorf("search meta cannot be empty")
	}

	tasks, err := s.taskRepo.FindBySourceMeta(ctx, meta)
	if err != nil {
		return nil, fmt.Errorf("failed to find tasks by source meta: %w", err)
	}

	s.logger.Debug(ctx, "tasks found by source meta",
		"criteria", meta,
		"count", len(tasks))
	return tasks, nil
}

// SearchTasks ищет задачи по критериям
func (s *TaskService) SearchTasks(ctx context.Context, query ports.TaskQuery) (*ports.TaskSearchResult, error) {
	tasks, err := s.taskRepo.FindByQuery(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to search tasks: %w", err)
	}

	totalCount := len(tasks)

	// Применяем пагинацию
	start := query.Offset
	if start < 0 {
		start = 0
	}
	if start >= totalCount {
		start = totalCount
	}

	end := start + query.Limit
	if end > totalCount {
		end = totalCount
	}
	if query.Limit <= 0 {
		end = totalCount
	}

	pagedTasks := tasks[start:end]

	result := &ports.TaskSearchResult{
		Tasks:      pagedTasks,
		TotalCount: totalCount,
		Page:       query.Offset/query.Limit + 1,
		PageSize:   query.Limit,
		TotalPages: (totalCount + query.Limit - 1) / query.Limit,
	}

	return result, nil
}

// GetCustomerTasks возвращает задачи клиента
func (s *TaskService) GetCustomerTasks(ctx context.Context, customerID string) ([]domain.Task, error) {
	if customerID == "" {
		return nil, errors.New("customer ID is required")
	}

	tasks, err := s.taskRepo.FindByCustomerID(ctx, customerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get customer tasks: %w", err)
	}

	return tasks, nil
}

// GetSubtasks возвращает подзадачи
func (s *TaskService) GetSubtasks(ctx context.Context, parentID string) ([]domain.Task, error) {
	if parentID == "" {
		return nil, errors.New("parent ID is required")
	}

	tasks, err := s.taskRepo.FindSubtasks(ctx, parentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get subtasks: %w", err)
	}

	return tasks, nil
}

// GetStats возвращает статистику по задачам
func (s *TaskService) GetStats(ctx context.Context, query ports.StatsQuery) (*ports.TaskStats, error) {
	stats, err := s.taskRepo.GetStats(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get stats: %w", err)
	}

	return stats, nil
}

// Вспомогательные методы

func (s *TaskService) validateCreateTaskRequest(req ports.CreateTaskRequest) error {
	if req.Subject == "" {
		return errors.New("subject is required")
	}
	if req.Description == "" {
		return errors.New("description is required")
	}
	if req.ReporterID == "" {
		return errors.New("reporter ID is required")
	}
	if req.Type == "" {
		return errors.New("task type is required")
	}

	// Валидация типа задачи
	validTypes := map[domain.TaskType]bool{
		domain.TaskTypeSupport:  true,
		domain.TaskTypeInternal: true,
		domain.TaskTypeSubTask:  true,
	}

	if !validTypes[req.Type] {
		return fmt.Errorf("invalid task type: %s", req.Type)
	}

	return nil
}

// Методы требующие реализации (заглушки)

func (s *TaskService) GetUserTasks(ctx context.Context, userID string, userRole domain.UserRole) (*ports.UserTasks, error) {
	// TODO: Реализовать получение задач пользователя по роли
	return &ports.UserTasks{}, nil
}

func (s *TaskService) GetDashboard(ctx context.Context, userID string) (*ports.UserDashboard, error) {
	// TODO: Реализовать дашборд пользователя
	return &ports.UserDashboard{}, nil
}

func (s *TaskService) AutoAssignTasks(ctx context.Context) ([]ports.AutoAssignmentResult, error) {
	// TODO: Реализовать автоматическое назначение
	return []ports.AutoAssignmentResult{}, nil
}

func (s *TaskService) ProcessEscalations(ctx context.Context) ([]ports.EscalationResult, error) {
	// TODO: Реализовать обработку эскалаций
	return []ports.EscalationResult{}, nil
}

func (s *TaskService) BulkUpdateStatus(ctx context.Context, taskIDs []string, status domain.TaskStatus, userID string) ([]ports.BulkOperationResult, error) {
	// TODO: Реализовать массовое обновление статусов
	return []ports.BulkOperationResult{}, nil
}

func (s *TaskService) BulkAssign(ctx context.Context, taskIDs []string, assigneeID string, userID string) ([]ports.BulkOperationResult, error) {
	// TODO: Реализовать массовое назначение
	return []ports.BulkOperationResult{}, nil
}

func (s *TaskService) AddParticipant(ctx context.Context, id string, userID string, role domain.ParticipantRole) (*domain.Task, error) {
	// TODO: Реализовать добавление участника
	return &domain.Task{}, nil
}
