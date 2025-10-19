// internal/core/domain/entities.go
package domain

import "time"

// Participant представляет участника задачи
type Participant struct {
	UserID   string
	Role     ParticipantRole
	JoinedAt time.Time
}

// Message представляет сообщение в задаче
type Message struct {
	ID        string
	Content   string
	AuthorID  string
	Type      MessageType
	CreatedAt time.Time
}

// Customer представляет клиента/организацию
type Customer struct {
	ID           string
	Name         string
	Email        string
	Phone        string
	Organization *Organization
	Projects     []ProjectMembership // Заглушка для будущего
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// Organization представляет организацию
type Organization struct {
	ID   string
	Name string
}

// ProjectMembership представляет принадлежность к проекту (заглушка)
type ProjectMembership struct {
	ProjectID string
	Role      string
}

// User представляет пользователя системы (заглушка для RBAC)
type User struct {
	ID    string
	Email string
	Name  string
	Role  UserRole // Заглушка для RBAC
}

// UserRole роль пользователя (заглушка для RBAC)
type UserRole string

const (
	UserRoleAdmin    UserRole = "admin"
	UserRoleManager  UserRole = "manager"
	UserRoleOperator UserRole = "operator"
	UserRoleViewer   UserRole = "viewer"
)
