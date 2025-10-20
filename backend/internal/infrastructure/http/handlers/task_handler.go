// internal/infrastructure/http/handlers/task_handler.go
package handlers

import (
	"net/http"
	"time"

	"github.com/audetv/urms/internal/core/domain"
	"github.com/audetv/urms/internal/core/ports"
	"github.com/audetv/urms/internal/infrastructure/http/dto"
	"github.com/gin-gonic/gin"
)

type TaskHandler struct {
	taskService ports.TaskService
	logger      ports.Logger
}

func NewTaskHandler(taskService ports.TaskService, logger ports.Logger) *TaskHandler {
	return &TaskHandler{
		taskService: taskService,
		logger:      logger,
	}
}

// CreateTask создает новую задачу
// @Summary Создать задачу
// @Description Создает новую задачу указанного типа
// @Tags tasks
// @Accept json
// @Produce json
// @Param request body dto.CreateTaskRequest true "Данные для создания задачи"
// @Success 201 {object} dto.BaseResponse{data=dto.TaskResponse}
// @Failure 400 {object} dto.BaseResponse
// @Failure 500 {object} dto.BaseResponse
// @Router /api/tasks [post]
func (h *TaskHandler) CreateTask(c *gin.Context) {
	ctx := c.Request.Context()
	var req dto.CreateTaskRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn(ctx, "Invalid create task request", "error", err.Error())
		c.JSON(http.StatusBadRequest, dto.NewErrorResponse(
			"INVALID_REQUEST",
			"Неверный формат запроса",
			err.Error(),
		))
		return
	}

	// Преобразуем DTO в портовый запрос
	createReq := ports.CreateTaskRequest{
		Type:        req.Type,
		Subject:     req.Subject,
		Description: req.Description,
		CustomerID:  req.CustomerID,
		ReporterID:  "system", // TODO: Заменить на ID авторизованного пользователя
		Source:      domain.SourceInternal,
		Priority:    req.Priority,
		Category:    req.Category,
		Tags:        req.Tags,
		ParentID:    req.ParentID,
		ProjectID:   req.ProjectID,
	}

	task, err := h.taskService.CreateTask(ctx, createReq)
	if err != nil {
		h.logger.Error(ctx, "Failed to create task", "error", err.Error())
		c.JSON(http.StatusInternalServerError, dto.NewErrorResponse(
			"TASK_CREATION_FAILED",
			"Не удалось создать задачу",
			err.Error(),
		))
		return
	}

	// Устанавливаем due date если указан
	if req.DueDate != nil {
		dueDateStr := req.DueDate.Format(time.RFC3339) // СОЗДАЕМ ПЕРЕМЕННУЮ
		updateReq := ports.UpdateTaskRequest{
			DueDate: &dueDateStr, // ПЕРЕДАЕМ АДРЕС ПЕРЕМЕННОЙ
		}
		task, err = h.taskService.UpdateTask(ctx, task.ID, updateReq)
		if err != nil {
			h.logger.Warn(ctx, "Failed to set due date for task",
				"task_id", task.ID, "error", err.Error())
		}
	}

	h.logger.Info(ctx, "Task created successfully", "task_id", task.ID)
	c.JSON(http.StatusCreated, dto.NewSuccessResponse(h.toTaskResponse(task)))
}

// CreateSupportTask создает задачу поддержки
// @Summary Создать задачу поддержки
// @Description Создает задачу поддержки для клиента
// @Tags tasks
// @Accept json
// @Produce json
// @Param request body dto.CreateSupportTaskRequest true "Данные для создания задачи поддержки"
// @Success 201 {object} dto.BaseResponse{data=dto.TaskResponse}
// @Failure 400 {object} dto.BaseResponse
// @Failure 500 {object} dto.BaseResponse
// @Router /api/tasks/support [post]
func (h *TaskHandler) CreateSupportTask(c *gin.Context) {
	ctx := c.Request.Context()
	var req dto.CreateSupportTaskRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn(ctx, "Invalid create support task request", "error", err.Error())
		c.JSON(http.StatusBadRequest, dto.NewErrorResponse(
			"INVALID_REQUEST",
			"Неверный формат запроса",
			err.Error(),
		))
		return
	}

	createReq := ports.CreateSupportTaskRequest{
		Subject:     req.Subject,
		Description: req.Description,
		CustomerID:  req.CustomerID,
		ReporterID:  "system", // TODO: Заменить на ID авторизованного пользователя
		Source:      domain.SourceInternal,
		Priority:    req.Priority,
		Category:    req.Category,
		Tags:        req.Tags,
	}

	task, err := h.taskService.CreateSupportTask(ctx, createReq)
	if err != nil {
		h.logger.Error(ctx, "Failed to create support task", "error", err.Error())
		c.JSON(http.StatusInternalServerError, dto.NewErrorResponse(
			"SUPPORT_TASK_CREATION_FAILED",
			"Не удалось создать задачу поддержки",
			err.Error(),
		))
		return
	}

	h.logger.Info(ctx, "Support task created successfully", "task_id", task.ID)
	c.JSON(http.StatusCreated, dto.NewSuccessResponse(h.toTaskResponse(task)))
}

// GetTask возвращает задачу по ID
// @Summary Получить задачу
// @Description Возвращает детальную информацию о задаче
// @Tags tasks
// @Produce json
// @Param id path string true "ID задачи"
// @Success 200 {object} dto.BaseResponse{data=dto.TaskResponse}
// @Failure 404 {object} dto.BaseResponse
// @Failure 500 {object} dto.BaseResponse
// @Router /api/tasks/{id} [get]
func (h *TaskHandler) GetTask(c *gin.Context) {
	ctx := c.Request.Context()
	taskID := c.Param("id")

	task, err := h.taskService.GetTask(ctx, taskID)
	if err != nil {
		h.logger.Error(ctx, "Failed to get task", "task_id", taskID, "error", err.Error())
		c.JSON(http.StatusNotFound, dto.NewErrorResponse(
			"TASK_NOT_FOUND",
			"Задача не найдена",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, dto.NewSuccessResponse(h.toTaskResponse(task)))
}

// UpdateTask обновляет задачу
// @Summary Обновить задачу
// @Description Обновляет данные задачи
// @Tags tasks
// @Accept json
// @Produce json
// @Param id path string true "ID задачи"
// @Param request body dto.UpdateTaskRequest true "Данные для обновления"
// @Success 200 {object} dto.BaseResponse{data=dto.TaskResponse}
// @Failure 400 {object} dto.BaseResponse
// @Failure 404 {object} dto.BaseResponse
// @Failure 500 {object} dto.BaseResponse
// @Router /api/tasks/{id} [put]
func (h *TaskHandler) UpdateTask(c *gin.Context) {
	ctx := c.Request.Context()
	taskID := c.Param("id")
	var req dto.UpdateTaskRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn(ctx, "Invalid update task request", "task_id", taskID, "error", err.Error())
		c.JSON(http.StatusBadRequest, dto.NewErrorResponse(
			"INVALID_REQUEST",
			"Неверный формат запроса",
			err.Error(),
		))
		return
	}

	// Преобразуем DTO в портовый запрос
	updateReq := ports.UpdateTaskRequest{
		Subject:     req.Subject,
		Description: req.Description,
		Priority:    req.Priority,
		Category:    req.Category,
		Tags:        req.Tags,
	}

	if req.DueDate != nil {
		dueDateStr := req.DueDate.Format(time.RFC3339)
		updateReq.DueDate = &dueDateStr
	}

	task, err := h.taskService.UpdateTask(ctx, taskID, updateReq)
	if err != nil {
		h.logger.Error(ctx, "Failed to update task", "task_id", taskID, "error", err.Error())
		c.JSON(http.StatusInternalServerError, dto.NewErrorResponse(
			"TASK_UPDATE_FAILED",
			"Не удалось обновить задачу",
			err.Error(),
		))
		return
	}

	h.logger.Info(ctx, "Task updated successfully", "task_id", taskID)
	c.JSON(http.StatusOK, dto.NewSuccessResponse(h.toTaskResponse(task)))
}

// DeleteTask удаляет задачу
// @Summary Удалить задачу
// @Description Удаляет задачу по ID
// @Tags tasks
// @Produce json
// @Param id path string true "ID задачи"
// @Success 204
// @Failure 404 {object} dto.BaseResponse
// @Failure 500 {object} dto.BaseResponse
// @Router /api/tasks/{id} [delete]
func (h *TaskHandler) DeleteTask(c *gin.Context) {
	ctx := c.Request.Context()
	taskID := c.Param("id")

	err := h.taskService.DeleteTask(ctx, taskID)
	if err != nil {
		h.logger.Error(ctx, "Failed to delete task", "task_id", taskID, "error", err.Error())
		c.JSON(http.StatusInternalServerError, dto.NewErrorResponse(
			"TASK_DELETION_FAILED",
			"Не удалось удалить задачу",
			err.Error(),
		))
		return
	}

	h.logger.Info(ctx, "Task deleted successfully", "task_id", taskID)
	c.Status(http.StatusNoContent)
}

// ListTasks возвращает список задач с фильтрацией
// @Summary Список задач
// @Description Возвращает список задач с поддержкой фильтрации и пагинации
// @Tags tasks
// @Produce json
// @Param types query []string false "Типы задач" collectionFormat(multi)
// @Param statuses query []string false "Статусы задач" collectionFormat(multi)
// @Param priorities query []string false "Приоритеты" collectionFormat(multi)
// @Param assignee_id query string false "ID исполнителя"
// @Param customer_id query string false "ID клиента"
// @Param reporter_id query string false "ID автора"
// @Param category query string false "Категория"
// @Param tags query []string false "Теги" collectionFormat(multi)
// @Param search_text query string false "Поисковый запрос"
// @Param page query int false "Номер страницы" default(1) minimum(1)
// @Param page_size query int false "Размер страницы" default(20) minimum(1) maximum(100)
// @Param sort_by query string false "Поле для сортировки"
// @Param sort_order query string false "Порядок сортировки" Enums(asc, desc)
// @Success 200 {object} dto.BaseResponse{data=dto.TaskListResponse}
// @Failure 400 {object} dto.BaseResponse
// @Failure 500 {object} dto.BaseResponse
// @Router /api/tasks [get]
func (h *TaskHandler) ListTasks(c *gin.Context) {
	ctx := c.Request.Context()
	var req dto.TaskSearchRequest

	if err := c.ShouldBindQuery(&req); err != nil {
		h.logger.Warn(ctx, "Invalid task search request", "error", err.Error())
		c.JSON(http.StatusBadRequest, dto.NewErrorResponse(
			"INVALID_SEARCH_REQUEST",
			"Неверные параметры поиска",
			err.Error(),
		))
		return
	}

	// Устанавливаем значения по умолчанию
	if req.Page == 0 {
		req.Page = 1
	}
	if req.PageSize == 0 {
		req.PageSize = 20
	}

	// Преобразуем DTO в портовый запрос
	query := ports.TaskQuery{
		Types:      req.Types,
		Statuses:   req.Statuses,
		Priorities: req.Priorities,
		AssigneeID: req.AssigneeID,
		CustomerID: req.CustomerID,
		ReporterID: req.ReporterID,
		Category:   req.Category,
		Tags:       req.Tags,
		SearchText: req.SearchText,
		Offset:     (req.Page - 1) * req.PageSize,
		Limit:      req.PageSize,
		SortBy:     req.SortBy,
		SortOrder:  req.SortOrder,
	}

	result, err := h.taskService.SearchTasks(ctx, query)
	if err != nil {
		h.logger.Error(ctx, "Failed to search tasks", "error", err.Error())
		c.JSON(http.StatusInternalServerError, dto.NewErrorResponse(
			"SEARCH_FAILED",
			"Не удалось выполнить поиск задач",
			err.Error(),
		))
		return
	}

	// Преобразуем задачи в DTO
	taskResponses := make([]dto.TaskResponse, len(result.Tasks))
	for i, task := range result.Tasks {
		taskResponses[i] = h.toTaskResponse(&task)
	}

	response := dto.TaskListResponse{
		Tasks: taskResponses,
		Pagination: dto.PageInfo{
			Page:       result.Page,
			PageSize:   result.PageSize,
			TotalCount: result.TotalCount,
			TotalPages: result.TotalPages,
		},
	}

	c.JSON(http.StatusOK, dto.NewPaginatedResponse(response, response.Pagination))
}

// ChangeStatus изменяет статус задачи
// @Summary Изменить статус задачи
// @Description Изменяет статус указанной задачи
// @Tags tasks
// @Accept json
// @Produce json
// @Param id path string true "ID задачи"
// @Param request body dto.ChangeStatusRequest true "Новый статус"
// @Success 200 {object} dto.BaseResponse{data=dto.TaskResponse}
// @Failure 400 {object} dto.BaseResponse
// @Failure 404 {object} dto.BaseResponse
// @Failure 500 {object} dto.BaseResponse
// @Router /api/tasks/{id}/status [put]
func (h *TaskHandler) ChangeStatus(c *gin.Context) {
	ctx := c.Request.Context()
	taskID := c.Param("id")
	var req dto.ChangeStatusRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn(ctx, "Invalid change status request", "task_id", taskID, "error", err.Error())
		c.JSON(http.StatusBadRequest, dto.NewErrorResponse(
			"INVALID_REQUEST",
			"Неверный формат запроса",
			err.Error(),
		))
		return
	}

	task, err := h.taskService.ChangeStatus(ctx, taskID, req.Status, "system") // TODO: Заменить на ID пользователя
	if err != nil {
		h.logger.Error(ctx, "Failed to change task status", "task_id", taskID, "error", err.Error())
		c.JSON(http.StatusInternalServerError, dto.NewErrorResponse(
			"STATUS_CHANGE_FAILED",
			"Не удалось изменить статус задачи",
			err.Error(),
		))
		return
	}

	h.logger.Info(ctx, "Task status changed", "task_id", taskID, "status", req.Status)
	c.JSON(http.StatusOK, dto.NewSuccessResponse(h.toTaskResponse(task)))
}

// AssignTask назначает исполнителя задачи
// @Summary Назначить исполнителя
// @Description Назначает исполнителя для задачи
// @Tags tasks
// @Accept json
// @Produce json
// @Param id path string true "ID задачи"
// @Param request body dto.AssignTaskRequest true "ID исполнителя"
// @Success 200 {object} dto.BaseResponse{data=dto.TaskResponse}
// @Failure 400 {object} dto.BaseResponse
// @Failure 404 {object} dto.BaseResponse
// @Failure 500 {object} dto.BaseResponse
// @Router /api/tasks/{id}/assign [put]
func (h *TaskHandler) AssignTask(c *gin.Context) {
	ctx := c.Request.Context()
	taskID := c.Param("id")
	var req dto.AssignTaskRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn(ctx, "Invalid assign task request", "task_id", taskID, "error", err.Error())
		c.JSON(http.StatusBadRequest, dto.NewErrorResponse(
			"INVALID_REQUEST",
			"Неверный формат запроса",
			err.Error(),
		))
		return
	}

	task, err := h.taskService.AssignTask(ctx, taskID, req.AssigneeID, "system") // TODO: Заменить на ID пользователя
	if err != nil {
		h.logger.Error(ctx, "Failed to assign task", "task_id", taskID, "error", err.Error())
		c.JSON(http.StatusInternalServerError, dto.NewErrorResponse(
			"ASSIGNMENT_FAILED",
			"Не удалось назначить исполнителя",
			err.Error(),
		))
		return
	}

	h.logger.Info(ctx, "Task assigned", "task_id", taskID, "assignee_id", req.AssigneeID)
	c.JSON(http.StatusOK, dto.NewSuccessResponse(h.toTaskResponse(task)))
}

// AddMessage добавляет сообщение в задачу
// @Summary Добавить сообщение
// @Description Добавляет сообщение в задачу
// @Tags tasks
// @Accept json
// @Produce json
// @Param id path string true "ID задачи"
// @Param request body dto.AddMessageRequest true "Данные сообщения"
// @Success 201 {object} dto.BaseResponse{data=dto.TaskResponse}
// @Failure 400 {object} dto.BaseResponse
// @Failure 404 {object} dto.BaseResponse
// @Failure 500 {object} dto.BaseResponse
// @Router /api/tasks/{id}/messages [post]
func (h *TaskHandler) AddMessage(c *gin.Context) {
	ctx := c.Request.Context()
	taskID := c.Param("id")
	var req dto.AddMessageRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn(ctx, "Invalid add message request", "task_id", taskID, "error", err.Error())
		c.JSON(http.StatusBadRequest, dto.NewErrorResponse(
			"INVALID_REQUEST",
			"Неверный формат запроса",
			err.Error(),
		))
		return
	}

	messageReq := ports.AddMessageRequest{
		AuthorID:  "system", // TODO: Заменить на ID авторизованного пользователя
		Content:   req.Content,
		Type:      req.Type,
		IsPrivate: req.IsPrivate,
	}

	task, err := h.taskService.AddMessage(ctx, taskID, messageReq)
	if err != nil {
		h.logger.Error(ctx, "Failed to add message to task", "task_id", taskID, "error", err.Error())
		c.JSON(http.StatusInternalServerError, dto.NewErrorResponse(
			"MESSAGE_ADD_FAILED",
			"Не удалось добавить сообщение",
			err.Error(),
		))
		return
	}

	h.logger.Info(ctx, "Message added to task", "task_id", taskID)
	c.JSON(http.StatusCreated, dto.NewSuccessResponse(h.toTaskResponse(task)))
}

// GetTaskMessages возвращает сообщения задачи
// @Summary Получить сообщения задачи
// @Description Возвращает список сообщений указанной задачи
// @Tags tasks
// @Produce json
// @Param id path string true "ID задачи"
// @Success 200 {object} dto.BaseResponse{data=[]dto.MessageResponse}
// @Failure 404 {object} dto.BaseResponse
// @Failure 500 {object} dto.BaseResponse
// @Router /api/tasks/{id}/messages [get]
func (h *TaskHandler) GetTaskMessages(c *gin.Context) {
	ctx := c.Request.Context()
	taskID := c.Param("id")

	task, err := h.taskService.GetTask(ctx, taskID)
	if err != nil {
		h.logger.Error(ctx, "Failed to get task messages", "task_id", taskID, "error", err.Error())
		c.JSON(http.StatusNotFound, dto.NewErrorResponse(
			"TASK_NOT_FOUND",
			"Задача не найдена",
			err.Error(),
		))
		return
	}

	messages := make([]dto.MessageResponse, len(task.Messages))
	for i, msg := range task.Messages {
		messages[i] = dto.MessageResponse{
			ID:        msg.ID,
			Content:   msg.Content,
			AuthorID:  msg.AuthorID,
			Type:      msg.Type,
			CreatedAt: msg.CreatedAt,
		}
	}

	c.JSON(http.StatusOK, dto.NewSuccessResponse(messages))
}

// Вспомогательные методы

func (h *TaskHandler) toTaskResponse(task *domain.Task) dto.TaskResponse {
	response := dto.TaskResponse{
		ID:          task.ID,
		Type:        task.Type,
		Subject:     task.Subject,
		Description: task.Description,
		Status:      task.Status,
		Priority:    task.Priority,
		Category:    task.Category,
		Tags:        task.Tags,
		ParentID:    task.ParentID,
		ProjectID:   task.ProjectID,
		AssigneeID:  task.AssigneeID,
		ReporterID:  task.ReporterID,
		CustomerID:  task.CustomerID,
		Source:      task.Source,
		SourceMeta:  task.SourceMeta,
		CreatedAt:   task.CreatedAt,
		UpdatedAt:   task.UpdatedAt,
		DueDate:     task.DueDate,
		ResolvedAt:  task.ResolvedAt,
		ClosedAt:    task.ClosedAt,
	}

	// Преобразуем участников
	response.Participants = make([]dto.ParticipantResponse, len(task.Participants))
	for i, participant := range task.Participants {
		response.Participants[i] = dto.ParticipantResponse{
			UserID:   participant.UserID,
			Role:     participant.Role,
			JoinedAt: participant.JoinedAt,
		}
	}

	// Преобразуем сообщения (если нужны в ответе)
	response.Messages = make([]dto.MessageResponse, len(task.Messages))
	for i, message := range task.Messages {
		response.Messages[i] = dto.MessageResponse{
			ID:        message.ID,
			Content:   message.Content,
			AuthorID:  message.AuthorID,
			Type:      message.Type,
			CreatedAt: message.CreatedAt,
		}
	}

	// Преобразуем историю (если нужна в ответе)
	response.History = make([]dto.TaskEventResponse, len(task.History))
	for i, event := range task.History {
		response.History[i] = dto.TaskEventResponse{
			ID:        event.ID,
			Type:      event.Type,
			UserID:    event.UserID,
			OldValue:  event.OldValue,
			NewValue:  event.NewValue,
			Timestamp: event.Timestamp,
			Message:   event.Message,
		}
	}

	return response
}
