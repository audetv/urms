// internal/core/domain/ticket.go
package domain

import (
	"errors"
	"fmt"
	"time"
)

// Ticket представляет основную сущность - заявку/обращение
type Ticket struct {
	ID           string
	Subject      string
	Description  string
	Status       TicketStatus
	Priority     Priority
	Category     string
	Tags         []string
	AssigneeID   string
	ReporterID   string
	CustomerID   string
	Source       TicketSource
	SourceMeta   map[string]interface{}
	Participants []Participant
	Messages     []Message
	SubTickets   []SubTicket
	CreatedAt    time.Time
	UpdatedAt    time.Time
	ResolvedAt   *time.Time
	ClosedAt     *time.Time
}

// NewTicket создает новую заявку с валидацией
func NewTicket(
	subject string,
	description string,
	source TicketSource,
	customerID string,
	reporterID string,
) (*Ticket, error) {

	if subject == "" {
		return nil, errors.New("subject is required")
	}
	if description == "" {
		return nil, errors.New("description is required")
	}
	if customerID == "" {
		return nil, errors.New("customer ID is required")
	}

	now := time.Now()
	ticket := &Ticket{
		ID:          GenerateTicketID(),
		Subject:     subject,
		Description: description,
		Status:      StatusOpen,
		Priority:    PriorityMedium,
		Source:      source,
		CustomerID:  customerID,
		ReporterID:  reporterID,
		SourceMeta:  make(map[string]interface{}),
		Tags:        []string{},
		Participants: []Participant{
			{
				UserID:   reporterID,
				Role:     RoleReporter,
				JoinedAt: now,
			},
		},
		Messages:   []Message{},
		SubTickets: []SubTicket{},
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	return ticket, nil
}

// AddMessage добавляет сообщение в тикет
func (t *Ticket) AddMessage(authorID, content string, messageType MessageType) error {
	if content == "" {
		return errors.New("message content is required")
	}

	message := Message{
		ID:        GenerateMessageID(),
		Content:   content,
		AuthorID:  authorID,
		Type:      messageType,
		CreatedAt: time.Now(),
	}

	t.Messages = append(t.Messages, message)
	t.UpdatedAt = time.Now()

	// Автоматически добавляем автора в участники если его еще нет
	t.addParticipantIfNotExists(authorID, RoleParticipant)

	return nil
}

// ChangeStatus изменяет статус тикета с валидацией переходов
func (t *Ticket) ChangeStatus(newStatus TicketStatus) error {
	if !t.isValidStatusTransition(newStatus) {
		return fmt.Errorf("invalid status transition from %s to %s", t.Status, newStatus)
	}

	oldStatus := t.Status
	t.Status = newStatus
	t.UpdatedAt = time.Now()

	// Обновляем временные метки при определенных статусах
	now := time.Now()
	switch newStatus {
	case StatusResolved:
		t.ResolvedAt = &now
	case StatusClosed:
		t.ClosedAt = &now
	case StatusOpen:
		// Сброс временных меток при возврате к открытому статусу
		t.ResolvedAt = nil
		t.ClosedAt = nil
	}

	// Добавляем системное сообщение о смене статуса
	systemMessage := fmt.Sprintf("Статус изменен: %s → %s", oldStatus.Russian(), newStatus.Russian())
	t.AddMessage("system", systemMessage, MessageTypeSystem)

	return nil
}

// Assign назначает исполнителя
func (t *Ticket) Assign(assigneeID string) error {
	if assigneeID == "" {
		return errors.New("assignee ID is required")
	}

	t.AssigneeID = assigneeID
	t.UpdatedAt = time.Now()

	// Добавляем исполнителя в участники
	t.addParticipantIfNotExists(assigneeID, RoleAssignee)

	// Системное сообщение о назначении
	message := fmt.Sprintf("Исполнитель назначен: %s", assigneeID)
	t.AddMessage("system", message, MessageTypeSystem)

	return nil
}

// AddTag добавляет тег к тикету
func (t *Ticket) AddTag(tag string) {
	for _, existingTag := range t.Tags {
		if existingTag == tag {
			return // Тег уже существует
		}
	}
	t.Tags = append(t.Tags, tag)
	t.UpdatedAt = time.Now()
}

// CreateSubTicket создает подзадачу
func (t *Ticket) CreateSubTicket(subject, description, assigneeID string) (*SubTicket, error) {
	subTicket := SubTicket{
		ID:          GenerateSubTicketID(t.ID),
		ParentID:    t.ID,
		Subject:     subject,
		Description: description,
		AssigneeID:  assigneeID,
		Status:      SubTicketStatusOpen,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	t.SubTickets = append(t.SubTickets, subTicket)
	t.UpdatedAt = time.Now()

	return &subTicket, nil
}

// Вспомогательные методы

func (t *Ticket) addParticipantIfNotExists(userID string, role ParticipantRole) {
	for _, participant := range t.Participants {
		if participant.UserID == userID {
			return // Участник уже существует
		}
	}

	t.Participants = append(t.Participants, Participant{
		UserID:   userID,
		Role:     role,
		JoinedAt: time.Now(),
	})
}

func (t *Ticket) isValidStatusTransition(newStatus TicketStatus) bool {
	validTransitions := map[TicketStatus][]TicketStatus{
		StatusOpen:       {StatusInProgress, StatusResolved, StatusClosed},
		StatusInProgress: {StatusOpen, StatusResolved, StatusClosed},
		StatusResolved:   {StatusOpen, StatusInProgress, StatusClosed},
		StatusClosed:     {StatusOpen}, // Закрытые тикеты можно только заново открыть
	}

	for _, validStatus := range validTransitions[t.Status] {
		if validStatus == newStatus {
			return true
		}
	}
	return false
}

// GenerateTicketID генерирует ID для тикета
func GenerateTicketID() string {
	return fmt.Sprintf("TKT-%d", time.Now().UnixNano())
}

// GenerateMessageID генерирует ID для сообщения
func GenerateMessageID() string {
	return fmt.Sprintf("MSG-%d", time.Now().UnixNano())
}

// GenerateSubTicketID генерирует ID для подзадачи
func GenerateSubTicketID(parentID string) string {
	return fmt.Sprintf("%s-SUB-%d", parentID, time.Now().UnixNano())
}
