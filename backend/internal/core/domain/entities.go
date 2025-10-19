// internal/core/domain/entities.go
package domain

import "time"

// Participant представляет участника заявки
type Participant struct {
	UserID   string
	Role     ParticipantRole
	JoinedAt time.Time
}

// Message представляет сообщение в заявке
type Message struct {
	ID        string
	Content   string
	AuthorID  string
	Type      MessageType
	CreatedAt time.Time
}

// SubTicket представляет подзадачу
type SubTicket struct {
	ID          string
	ParentID    string
	Subject     string
	Description string
	AssigneeID  string
	Status      SubTicketStatus
	CreatedAt   time.Time
	UpdatedAt   time.Time
	CompletedAt *time.Time
}

// Customer представляет клиента/организацию
type Customer struct {
	ID           string
	Name         string
	Email        string
	Phone        string
	Organization *Organization
	Projects     []ProjectMembership
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// Organization представляет организацию
type Organization struct {
	ID   string
	Name string
}

// ProjectMembership представляет принадлежность к проекту
type ProjectMembership struct {
	ProjectID string
	Role      string
}
