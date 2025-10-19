// internal/core/services/customer_service.go
package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/audetv/urms/internal/core/domain"
	"github.com/audetv/urms/internal/core/ports"
)

type CustomerService struct {
	customerRepo ports.CustomerRepository
	taskRepo     ports.TaskRepository
	logger       ports.Logger
}

func NewCustomerService(
	customerRepo ports.CustomerRepository,
	taskRepo ports.TaskRepository,
	logger ports.Logger,
) *CustomerService {
	return &CustomerService{
		customerRepo: customerRepo,
		taskRepo:     taskRepo,
		logger:       logger,
	}
}

// CreateCustomer создает нового клиента
func (s *CustomerService) CreateCustomer(ctx context.Context, req ports.CreateCustomerRequest) (*domain.Customer, error) {
	if err := s.validateCreateCustomerRequest(req); err != nil {
		return nil, err
	}

	// Проверяем, существует ли клиент с таким email
	existing, _ := s.customerRepo.FindByEmail(ctx, req.Email)
	if existing != nil {
		return nil, fmt.Errorf("customer with email %s already exists", req.Email)
	}

	customer := &domain.Customer{
		ID:        s.generateCustomerID(),
		Name:      req.Name,
		Email:     req.Email,
		Phone:     req.Phone,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if req.Organization != "" {
		customer.Organization = &domain.Organization{
			ID:   s.generateOrganizationID(),
			Name: req.Organization,
		}
	}

	if err := s.customerRepo.Save(ctx, customer); err != nil {
		return nil, fmt.Errorf("failed to save customer: %w", err)
	}

	s.logger.Info(ctx, "customer created",
		"customer_id", customer.ID,
		"email", customer.Email,
	)

	return customer, nil
}

// FindOrCreateByEmail находит или создает клиента по email
func (s *CustomerService) FindOrCreateByEmail(ctx context.Context, email, name string) (*domain.Customer, error) {
	if email == "" {
		return nil, errors.New("email is required")
	}

	// Ищем существующего клиента
	customer, err := s.customerRepo.FindByEmail(ctx, email)
	if err == nil && customer != nil {
		return customer, nil
	}

	// Создаем нового клиента
	req := ports.CreateCustomerRequest{
		Name:  name,
		Email: email,
	}

	return s.CreateCustomer(ctx, req)
}

// GetCustomerProfile возвращает профиль клиента с задачами и статистикой
func (s *CustomerService) GetCustomerProfile(ctx context.Context, id string) (*ports.CustomerProfile, error) {
	if id == "" {
		return nil, errors.New("customer ID is required")
	}

	customer, err := s.customerRepo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to find customer: %w", err)
	}

	// Получаем задачи клиента
	tasks, err := s.taskRepo.FindByCustomerID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get customer tasks: %w", err)
	}

	// Вычисляем статистику
	stats := s.calculateCustomerStats(tasks)

	// Получаем последние задачи
	recentTasks := s.getRecentTasks(tasks, 5)

	profile := &ports.CustomerProfile{
		Customer:     customer,
		Tasks:        tasks,
		Stats:        stats,
		RecentTasks:  recentTasks,
		Organization: customer.Organization,
	}

	return profile, nil
}

// UpdateCustomer обновляет данные клиента
func (s *CustomerService) UpdateCustomer(ctx context.Context, id string, req ports.UpdateCustomerRequest) (*domain.Customer, error) {
	customer, err := s.customerRepo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to find customer: %w", err)
	}

	// Обновляем поля если они предоставлены
	if req.Name != nil {
		customer.Name = *req.Name
	}
	if req.Email != nil {
		// Проверяем уникальность email
		if *req.Email != customer.Email {
			existing, _ := s.customerRepo.FindByEmail(ctx, *req.Email)
			if existing != nil {
				return nil, fmt.Errorf("customer with email %s already exists", *req.Email)
			}
			customer.Email = *req.Email
		}
	}
	if req.Phone != nil {
		customer.Phone = *req.Phone
	}
	if req.Organization != nil {
		if customer.Organization == nil {
			customer.Organization = &domain.Organization{
				ID: s.generateOrganizationID(),
			}
		}
		customer.Organization.Name = *req.Organization
	}

	customer.UpdatedAt = time.Now()

	if err := s.customerRepo.Update(ctx, customer); err != nil {
		return nil, fmt.Errorf("failed to update customer: %w", err)
	}

	s.logger.Info(ctx, "customer updated", "customer_id", customer.ID)
	return customer, nil
}

// DeleteCustomer удаляет клиента
func (s *CustomerService) DeleteCustomer(ctx context.Context, id string) error {
	if id == "" {
		return errors.New("customer ID is required")
	}

	// Проверяем существование клиента
	customer, err := s.customerRepo.FindByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to find customer: %w", err)
	}

	// Проверяем, есть ли у клиента активные задачи
	tasks, err := s.taskRepo.FindByCustomerID(ctx, id)
	if err == nil && len(tasks) > 0 {
		openTasks := 0
		for _, task := range tasks {
			if task.Status == domain.TaskStatusOpen || task.Status == domain.TaskStatusInProgress {
				openTasks++
			}
		}
		if openTasks > 0 {
			return fmt.Errorf("cannot delete customer with %d open tasks", openTasks)
		}
	}

	if err := s.customerRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete customer: %w", err)
	}

	s.logger.Info(ctx, "customer deleted",
		"customer_id", id,
		"customer_name", customer.Name,
	)

	return nil
}

// ListCustomers возвращает список клиентов с пагинацией
func (s *CustomerService) ListCustomers(ctx context.Context, query ports.CustomerQuery) (*ports.CustomerSearchResult, error) {
	// TODO: Реализовать полнофункциональный поиск клиентов
	// Временная реализация - возвращаем всех клиентов

	// Для демонстрации создаем пустой результат
	result := &ports.CustomerSearchResult{
		Customers:  []domain.Customer{},
		TotalCount: 0,
		Page:       1,
		PageSize:   query.Limit,
		TotalPages: 0,
	}

	return result, nil
}

// Вспомогательные методы

func (s *CustomerService) validateCreateCustomerRequest(req ports.CreateCustomerRequest) error {
	if req.Name == "" {
		return errors.New("name is required")
	}
	if req.Email == "" {
		return errors.New("email is required")
	}
	// TODO: Добавить валидацию email формата
	return nil
}

func (s *CustomerService) generateCustomerID() string {
	return fmt.Sprintf("CUST-%d", time.Now().UnixNano())
}

func (s *CustomerService) generateOrganizationID() string {
	return fmt.Sprintf("ORG-%d", time.Now().UnixNano())
}

func (s *CustomerService) calculateCustomerStats(tasks []domain.Task) *ports.CustomerStats {
	stats := &ports.CustomerStats{
		TotalTasks: len(tasks),
		ByPriority: make(map[domain.Priority]int),
		ByCategory: make(map[string]int),
	}

	for _, task := range tasks {
		// Считаем открытые задачи
		if task.Status == domain.TaskStatusOpen || task.Status == domain.TaskStatusInProgress {
			stats.OpenTasks++
		}

		// Распределение по приоритетам
		stats.ByPriority[task.Priority]++

		// Распределение по категориям
		if task.Category != "" {
			stats.ByCategory[task.Category]++
		}
	}

	// TODO: Реализовать расчет времени ответа и удовлетворенности
	stats.AvgResponseTime = 0
	stats.Satisfaction = 0

	return stats
}

func (s *CustomerService) getRecentTasks(tasks []domain.Task, limit int) []domain.Task {
	if len(tasks) <= limit {
		return tasks
	}

	// Сортируем задачи по дате создания (новые сначала)
	// и возвращаем последние limit задач
	return tasks[len(tasks)-limit:]
}
