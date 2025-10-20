// internal/infrastructure/http/handlers/customer_handler.go
package handlers

import (
	"net/http"

	"github.com/audetv/urms/internal/core/domain"
	"github.com/audetv/urms/internal/core/ports"
	"github.com/audetv/urms/internal/infrastructure/http/dto"
	"github.com/gin-gonic/gin"
)

type CustomerHandler struct {
	customerService ports.CustomerService
	taskService     ports.TaskService
	logger          ports.Logger
}

func NewCustomerHandler(
	customerService ports.CustomerService,
	taskService ports.TaskService,
	logger ports.Logger,
) *CustomerHandler {
	return &CustomerHandler{
		customerService: customerService,
		taskService:     taskService,
		logger:          logger,
	}
}

// CreateCustomer создает нового клиента
// @Summary Создать клиента
// @Description Создает нового клиента в системе
// @Tags customers
// @Accept json
// @Produce json
// @Param request body dto.CreateCustomerRequest true "Данные клиента"
// @Success 201 {object} dto.BaseResponse{data=dto.CustomerResponse}
// @Failure 400 {object} dto.BaseResponse
// @Failure 500 {object} dto.BaseResponse
// @Router /api/customers [post]
func (h *CustomerHandler) CreateCustomer(c *gin.Context) {
	ctx := c.Request.Context()
	var req dto.CreateCustomerRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn(ctx, "Invalid create customer request", "error", err.Error())
		c.JSON(http.StatusBadRequest, dto.NewErrorResponse(
			"INVALID_REQUEST",
			"Неверный формат запроса",
			err.Error(),
		))
		return
	}

	createReq := ports.CreateCustomerRequest{
		Name:         req.Name,
		Email:        req.Email,
		Phone:        req.Phone,
		Organization: req.Organization,
	}

	customer, err := h.customerService.CreateCustomer(ctx, createReq)
	if err != nil {
		h.logger.Error(ctx, "Failed to create customer", "error", err.Error())
		c.JSON(http.StatusInternalServerError, dto.NewErrorResponse(
			"CUSTOMER_CREATION_FAILED",
			"Не удалось создать клиента",
			err.Error(),
		))
		return
	}

	h.logger.Info(ctx, "Customer created successfully", "customer_id", customer.ID)
	c.JSON(http.StatusCreated, dto.NewSuccessResponse(h.toCustomerResponse(customer)))
}

// GetCustomer возвращает клиента по ID
// @Summary Получить клиента
// @Description Возвращает информацию о клиенте по ID
// @Tags customers
// @Produce json
// @Param id path string true "ID клиента"
// @Success 200 {object} dto.BaseResponse{data=dto.CustomerResponse}
// @Failure 404 {object} dto.BaseResponse
// @Failure 500 {object} dto.BaseResponse
// @Router /api/customers/{id} [get]
func (h *CustomerHandler) GetCustomer(c *gin.Context) {
	ctx := c.Request.Context()
	customerID := c.Param("id")

	customer, err := h.customerService.GetCustomerProfile(ctx, customerID)
	if err != nil {
		h.logger.Error(ctx, "Failed to get customer", "customer_id", customerID, "error", err.Error())
		c.JSON(http.StatusNotFound, dto.NewErrorResponse(
			"CUSTOMER_NOT_FOUND",
			"Клиент не найден",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, dto.NewSuccessResponse(h.toCustomerResponse(customer.Customer)))
}

// GetCustomerProfile возвращает полный профиль клиента с задачами
// @Summary Получить профиль клиента
// @Description Возвращает полный профиль клиента включая задачи и статистику
// @Tags customers
// @Produce json
// @Param id path string true "ID клиента"
// @Success 200 {object} dto.BaseResponse{data=dto.CustomerProfileResponse}
// @Failure 404 {object} dto.BaseResponse
// @Failure 500 {object} dto.BaseResponse
// @Router /api/customers/{id}/profile [get]
func (h *CustomerHandler) GetCustomerProfile(c *gin.Context) {
	ctx := c.Request.Context()
	customerID := c.Param("id")

	profile, err := h.customerService.GetCustomerProfile(ctx, customerID)
	if err != nil {
		h.logger.Error(ctx, "Failed to get customer profile", "customer_id", customerID, "error", err.Error())
		c.JSON(http.StatusNotFound, dto.NewErrorResponse(
			"CUSTOMER_NOT_FOUND",
			"Клиент не найден",
			err.Error(),
		))
		return
	}

	response := h.toCustomerProfileResponse(profile)
	c.JSON(http.StatusOK, dto.NewSuccessResponse(response))
}

// UpdateCustomer обновляет данные клиента
// @Summary Обновить клиента
// @Description Обновляет данные клиента
// @Tags customers
// @Accept json
// @Produce json
// @Param id path string true "ID клиента"
// @Param request body dto.UpdateCustomerRequest true "Данные для обновления"
// @Success 200 {object} dto.BaseResponse{data=dto.CustomerResponse}
// @Failure 400 {object} dto.BaseResponse
// @Failure 404 {object} dto.BaseResponse
// @Failure 500 {object} dto.BaseResponse
// @Router /api/customers/{id} [put]
func (h *CustomerHandler) UpdateCustomer(c *gin.Context) {
	ctx := c.Request.Context()
	customerID := c.Param("id")
	var req dto.UpdateCustomerRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn(ctx, "Invalid update customer request", "customer_id", customerID, "error", err.Error())
		c.JSON(http.StatusBadRequest, dto.NewErrorResponse(
			"INVALID_REQUEST",
			"Неверный формат запроса",
			err.Error(),
		))
		return
	}

	updateReq := ports.UpdateCustomerRequest{
		Name:         req.Name,
		Email:        req.Email,
		Phone:        req.Phone,
		Organization: req.Organization,
	}

	customer, err := h.customerService.UpdateCustomer(ctx, customerID, updateReq)
	if err != nil {
		h.logger.Error(ctx, "Failed to update customer", "customer_id", customerID, "error", err.Error())
		c.JSON(http.StatusInternalServerError, dto.NewErrorResponse(
			"CUSTOMER_UPDATE_FAILED",
			"Не удалось обновить клиента",
			err.Error(),
		))
		return
	}

	h.logger.Info(ctx, "Customer updated successfully", "customer_id", customerID)
	c.JSON(http.StatusOK, dto.NewSuccessResponse(h.toCustomerResponse(customer)))
}

// DeleteCustomer удаляет клиента
// @Summary Удалить клиента
// @Description Удаляет клиента по ID (только если нет активных задач)
// @Tags customers
// @Produce json
// @Param id path string true "ID клиента"
// @Success 204
// @Failure 400 {object} dto.BaseResponse
// @Failure 404 {object} dto.BaseResponse
// @Failure 500 {object} dto.BaseResponse
// @Router /api/customers/{id} [delete]
func (h *CustomerHandler) DeleteCustomer(c *gin.Context) {
	ctx := c.Request.Context()
	customerID := c.Param("id")

	err := h.customerService.DeleteCustomer(ctx, customerID)
	if err != nil {
		h.logger.Error(ctx, "Failed to delete customer", "customer_id", customerID, "error", err.Error())

		// Определяем тип ошибки для соответствующего HTTP статуса
		errorCode := "CUSTOMER_DELETION_FAILED"
		statusCode := http.StatusInternalServerError

		if err.Error() == "customer not found" {
			statusCode = http.StatusNotFound
			errorCode = "CUSTOMER_NOT_FOUND"
		} else if err.Error() == "cannot delete customer with open tasks" {
			statusCode = http.StatusBadRequest
			errorCode = "CUSTOMER_HAS_OPEN_TASKS"
		}

		c.JSON(statusCode, dto.NewErrorResponse(
			errorCode,
			"Не удалось удалить клиента",
			err.Error(),
		))
		return
	}

	h.logger.Info(ctx, "Customer deleted successfully", "customer_id", customerID)
	c.Status(http.StatusNoContent)
}

// ListCustomers возвращает список клиентов
// @Summary Список клиентов
// @Description Возвращает список клиентов с поддержкой поиска и пагинации
// @Tags customers
// @Produce json
// @Param search_text query string false "Поисковый запрос"
// @Param organization query string false "Организация"
// @Param email query string false "Email"
// @Param page query int false "Номер страницы" default(1) minimum(1)
// @Param page_size query int false "Размер страницы" default(20) minimum(1) maximum(100)
// @Success 200 {object} dto.BaseResponse{data=dto.CustomerListResponse}
// @Failure 400 {object} dto.BaseResponse
// @Failure 500 {object} dto.BaseResponse
// @Router /api/customers [get]
func (h *CustomerHandler) ListCustomers(c *gin.Context) {
	ctx := c.Request.Context()
	var req dto.CustomerSearchRequest

	if err := c.ShouldBindQuery(&req); err != nil {
		h.logger.Warn(ctx, "Invalid customer search request", "error", err.Error())
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

	query := ports.CustomerQuery{
		SearchText:   req.SearchText,
		Organization: req.Organization,
		Email:        req.Email,
		Offset:       (req.Page - 1) * req.PageSize,
		Limit:        req.PageSize,
	}

	result, err := h.customerService.ListCustomers(ctx, query)
	if err != nil {
		h.logger.Error(ctx, "Failed to list customers", "error", err.Error())
		c.JSON(http.StatusInternalServerError, dto.NewErrorResponse(
			"LIST_CUSTOMERS_FAILED",
			"Не удалось получить список клиентов",
			err.Error(),
		))
		return
	}

	// Преобразуем клиентов в DTO
	customerResponses := make([]dto.CustomerResponse, len(result.Customers))
	for i, customer := range result.Customers {
		customerResponses[i] = h.toCustomerResponse(&customer)
	}

	response := dto.CustomerListResponse{
		Customers: customerResponses,
		Pagination: dto.PageInfo{
			Page:       result.Page,
			PageSize:   result.PageSize,
			TotalCount: result.TotalCount,
			TotalPages: result.TotalPages,
		},
	}

	c.JSON(http.StatusOK, dto.NewPaginatedResponse(response, response.Pagination))
}

// GetCustomerTasks возвращает задачи клиента
// @Summary Получить задачи клиента
// @Description Возвращает список задач указанного клиента
// @Tags customers
// @Produce json
// @Param id path string true "ID клиента"
// @Success 200 {object} dto.BaseResponse{data=[]dto.TaskResponse}
// @Failure 404 {object} dto.BaseResponse
// @Failure 500 {object} dto.BaseResponse
// @Router /api/customers/{id}/tasks [get]
func (h *CustomerHandler) GetCustomerTasks(c *gin.Context) {
	ctx := c.Request.Context()
	customerID := c.Param("id")

	tasks, err := h.taskService.GetCustomerTasks(ctx, customerID)
	if err != nil {
		h.logger.Error(ctx, "Failed to get customer tasks", "customer_id", customerID, "error", err.Error())
		c.JSON(http.StatusInternalServerError, dto.NewErrorResponse(
			"GET_CUSTOMER_TASKS_FAILED",
			"Не удалось получить задачи клиента",
			err.Error(),
		))
		return
	}

	taskResponses := make([]dto.TaskResponse, len(tasks))
	for i, task := range tasks {
		taskResponses[i] = h.toTaskResponse(&task)
	}

	c.JSON(http.StatusOK, dto.NewSuccessResponse(taskResponses))
}

// FindOrCreateCustomer находит или создает клиента по email
// @Summary Найти или создать клиента
// @Description Находит клиента по email или создает нового если не найден
// @Tags customers
// @Produce json
// @Param email query string true "Email клиента"
// @Param name query string false "Имя клиента (если создается новый)"
// @Success 200 {object} dto.BaseResponse{data=dto.CustomerResponse}
// @Failure 400 {object} dto.BaseResponse
// @Failure 500 {object} dto.BaseResponse
// @Router /api/customers/find-or-create [get]
func (h *CustomerHandler) FindOrCreateCustomer(c *gin.Context) {
	ctx := c.Request.Context()
	email := c.Query("email")
	name := c.Query("name")

	if email == "" {
		c.JSON(http.StatusBadRequest, dto.NewErrorResponse(
			"MISSING_EMAIL",
			"Email обязателен для поиска",
			"",
		))
		return
	}

	if name == "" {
		// Генерируем имя из email если не указано
		name = "Customer" // TODO: Извлечь имя из email как в MessageProcessor
	}

	customer, err := h.customerService.FindOrCreateByEmail(ctx, email, name)
	if err != nil {
		h.logger.Error(ctx, "Failed to find or create customer", "email", email, "error", err.Error())
		c.JSON(http.StatusInternalServerError, dto.NewErrorResponse(
			"FIND_OR_CREATE_FAILED",
			"Не удалось найти или создать клиента",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, dto.NewSuccessResponse(h.toCustomerResponse(customer)))
}

// Вспомогательные методы

func (h *CustomerHandler) toCustomerResponse(customer *domain.Customer) dto.CustomerResponse {
	response := dto.CustomerResponse{
		ID:        customer.ID,
		Name:      customer.Name,
		Email:     customer.Email,
		Phone:     customer.Phone,
		CreatedAt: customer.CreatedAt,
		UpdatedAt: customer.UpdatedAt,
	}

	if customer.Organization != nil {
		response.Organization = &dto.OrganizationResponse{
			ID:   customer.Organization.ID,
			Name: customer.Organization.Name,
		}
	}

	return response
}

func (h *CustomerHandler) toCustomerProfileResponse(profile *ports.CustomerProfile) dto.CustomerProfileResponse {
	response := dto.CustomerProfileResponse{
		Customer: h.toCustomerResponse(profile.Customer),
		Stats: dto.CustomerStats{
			TotalTasks:      profile.Stats.TotalTasks,
			OpenTasks:       profile.Stats.OpenTasks,
			AvgResponseTime: profile.Stats.AvgResponseTime,
			Satisfaction:    profile.Stats.Satisfaction,
			ByPriority:      profile.Stats.ByPriority,
			ByCategory:      profile.Stats.ByCategory,
		},
	}

	// Преобразуем задачи
	response.Tasks = make([]dto.TaskResponse, len(profile.Tasks))
	for i, task := range profile.Tasks {
		response.Tasks[i] = h.toTaskResponse(&task)
	}

	// Преобразуем недавние задачи если есть
	if profile.RecentTasks != nil {
		response.RecentTasks = make([]dto.TaskResponse, len(profile.RecentTasks))
		for i, task := range profile.RecentTasks {
			response.RecentTasks[i] = h.toTaskResponse(&task)
		}
	}

	// Организация
	if profile.Organization != nil {
		response.Organization = &dto.OrganizationResponse{
			ID:   profile.Organization.ID,
			Name: profile.Organization.Name,
		}
	}

	return response
}

// Временная реализация - нужно вынести в общий mapper
func (h *CustomerHandler) toTaskResponse(task *domain.Task) dto.TaskResponse {
	// Упрощенная версия - в production нужно вынести в общий mapper
	return dto.TaskResponse{
		ID:          task.ID,
		Type:        task.Type,
		Subject:     task.Subject,
		Description: task.Description,
		Status:      task.Status,
		Priority:    task.Priority,
		Category:    task.Category,
		Tags:        task.Tags,
		AssigneeID:  task.AssigneeID,
		ReporterID:  task.ReporterID,
		CustomerID:  task.CustomerID,
		Source:      task.Source,
		CreatedAt:   task.CreatedAt,
		UpdatedAt:   task.UpdatedAt,
		DueDate:     task.DueDate,
		ResolvedAt:  task.ResolvedAt,
		ClosedAt:    task.ClosedAt,
	}
}
