// internal/infrastructure/persistence/inmemory/user_repository.go
package inmemory

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/audetv/urms/internal/core/domain"
	"github.com/audetv/urms/internal/core/ports"
)

type UserRepository struct {
	users  map[string]*domain.User
	mu     sync.RWMutex
	logger ports.Logger
}

func NewUserRepository(logger ports.Logger) *UserRepository {
	// Создаем несколько тестовых пользователей для демонстрации
	repo := &UserRepository{
		users:  make(map[string]*domain.User),
		logger: logger,
	}

	// Добавляем тестовых пользователей
	repo.users["user-1"] = &domain.User{
		ID:    "user-1",
		Email: "admin@company.com",
		Name:  "Admin User",
		Role:  domain.UserRoleAdmin,
	}

	repo.users["user-2"] = &domain.User{
		ID:    "user-2",
		Email: "manager@company.com",
		Name:  "Manager User",
		Role:  domain.UserRoleManager,
	}

	repo.users["user-3"] = &domain.User{
		ID:    "user-3",
		Email: "operator@company.com",
		Name:  "Operator User",
		Role:  domain.UserRoleOperator,
	}

	return repo
}

func (r *UserRepository) FindByID(ctx context.Context, id string) (*domain.User, error) {
	if id == "" {
		return nil, errors.New("user ID cannot be empty")
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	user, exists := r.users[id]
	if !exists {
		return nil, fmt.Errorf("user not found: %s", id)
	}

	return user, nil
}

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	if email == "" {
		return nil, errors.New("email cannot be empty")
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, user := range r.users {
		if user.Email == email {
			return user, nil
		}
	}

	return nil, nil // Не ошибка, просто не найден
}

func (r *UserRepository) FindAssignees(ctx context.Context) ([]domain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var assignees []domain.User
	for _, user := range r.users {
		// Считаем, что операторы и менеджеры могут быть исполнителями
		if user.Role == domain.UserRoleOperator || user.Role == domain.UserRoleManager {
			assignees = append(assignees, *user)
		}
	}

	r.logger.Debug(ctx, "assignees found", "count", len(assignees))
	return assignees, nil
}

func (r *UserRepository) Save(ctx context.Context, user *domain.User) error {
	if user == nil {
		return errors.New("user cannot be nil")
	}
	if user.ID == "" {
		return errors.New("user ID cannot be empty")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.users[user.ID] = user
	r.logger.Info(ctx, "user saved", "user_id", user.ID, "email", user.Email)
	return nil
}

func (r *UserRepository) Update(ctx context.Context, user *domain.User) error {
	if user == nil {
		return errors.New("user cannot be nil")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.users[user.ID]; !exists {
		return fmt.Errorf("user not found: %s", user.ID)
	}

	r.users[user.ID] = user
	r.logger.Info(ctx, "user updated", "user_id", user.ID)
	return nil
}

func (r *UserRepository) Delete(ctx context.Context, id string) error {
	if id == "" {
		return errors.New("user ID cannot be empty")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.users[id]; !exists {
		return fmt.Errorf("user not found: %s", id)
	}

	delete(r.users, id)
	r.logger.Info(ctx, "user deleted", "user_id", id)
	return nil
}
