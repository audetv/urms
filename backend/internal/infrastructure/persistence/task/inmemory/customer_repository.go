// internal/infrastructure/persistence/inmemory/customer_repository.go
package inmemory

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/audetv/urms/internal/core/domain"
	"github.com/audetv/urms/internal/core/ports"
)

type CustomerRepository struct {
	customers map[string]*domain.Customer
	mu        sync.RWMutex
	logger    ports.Logger
}

func NewCustomerRepository(logger ports.Logger) *CustomerRepository {
	return &CustomerRepository{
		customers: make(map[string]*domain.Customer),
		logger:    logger,
	}
}

func (r *CustomerRepository) Save(ctx context.Context, customer *domain.Customer) error {
	if customer == nil {
		return errors.New("customer cannot be nil")
	}
	if customer.ID == "" {
		return errors.New("customer ID cannot be empty")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.customers[customer.ID] = customer
	r.logger.Info(ctx, "customer saved", "customer_id", customer.ID, "email", customer.Email)
	return nil
}

func (r *CustomerRepository) FindByID(ctx context.Context, id string) (*domain.Customer, error) {
	if id == "" {
		return nil, errors.New("customer ID cannot be empty")
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	customer, exists := r.customers[id]
	if !exists {
		return nil, fmt.Errorf("customer not found: %s", id)
	}

	return customer, nil
}

func (r *CustomerRepository) FindByEmail(ctx context.Context, email string) (*domain.Customer, error) {
	if email == "" {
		return nil, errors.New("email cannot be empty")
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, customer := range r.customers {
		if customer.Email == email {
			return customer, nil
		}
	}

	return nil, nil // Не ошибка, просто не найден
}

func (r *CustomerRepository) FindByOrganization(ctx context.Context, orgID string) ([]domain.Customer, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var customers []domain.Customer
	for _, customer := range r.customers {
		if customer.Organization != nil && customer.Organization.ID == orgID {
			customers = append(customers, *customer)
		}
	}

	r.logger.Debug(ctx, "customers found by organization", "org_id", orgID, "count", len(customers))
	return customers, nil
}

func (r *CustomerRepository) Update(ctx context.Context, customer *domain.Customer) error {
	if customer == nil {
		return errors.New("customer cannot be nil")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.customers[customer.ID]; !exists {
		return fmt.Errorf("customer not found: %s", customer.ID)
	}

	r.customers[customer.ID] = customer
	r.logger.Info(ctx, "customer updated", "customer_id", customer.ID)
	return nil
}

func (r *CustomerRepository) Delete(ctx context.Context, id string) error {
	if id == "" {
		return errors.New("customer ID cannot be empty")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.customers[id]; !exists {
		return fmt.Errorf("customer not found: %s", id)
	}

	delete(r.customers, id)
	r.logger.Info(ctx, "customer deleted", "customer_id", id)
	return nil
}
