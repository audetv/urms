// internal/core/domain/task.go
package domain

import (
	"errors"
	"fmt"
	"time"
)

// TaskType тип задачи
type TaskType string

const (
	TaskTypeSupport  TaskType = "support"  // Заявка поддержки
	TaskTypeInternal TaskType = "internal" // Внутренняя задача
	TaskTypeSubTask  TaskType = "subtask"  // Подзадача
)

// TaskStatus статус задачи
type TaskStatus string

const (
	TaskStatusOpen       TaskStatus = "open"        // Открыта
	TaskStatusInProgress TaskStatus = "in_progress" // В работе
	TaskStatusReview     TaskStatus = "review"      // На проверке
	TaskStatusResolved   TaskStatus = "resolved"    // Решена
	TaskStatusClosed     TaskStatus = "closed"      // Закрыта
	TaskStatusCancelled  TaskStatus = "cancelled"   // Отменена
)

// Task представляет универсальную задачу
type Task struct {
	ID          string
	Type        TaskType
	Subject     string
	Description string
	Status      TaskStatus
	Priority    Priority
	Category    string
	Tags        []string

	// Связи
	ParentID    *string // Для подзадач
	ProjectID   *string // Привязка к проекту (заглушка для будущего)
	MilestoneID *string // Привязка к этапу (заглушка для будущего)

	// Участники
	AssigneeID   string
	ReporterID   string
	CustomerID   *string // Для задач поддержки
	Participants []Participant

	// Источник
	Source     TaskSource
	SourceMeta map[string]interface{}

	// Сообщения и история
	Messages []Message
	History  []TaskEvent

	// Временные метки
	CreatedAt  time.Time
	UpdatedAt  time.Time
	DueDate    *time.Time
	ResolvedAt *time.Time
	ClosedAt   *time.Time
}

// TaskEvent событие в истории задачи
type TaskEvent struct {
	ID        string
	Type      string
	UserID    string
	OldValue  interface{}
	NewValue  interface{}
	Timestamp time.Time
	Message   string
}

// NewTask создает новую задачу
func NewTask(
	taskType TaskType,
	subject string,
	description string,
	reporterID string,
	sourceMeta map[string]interface{}, // ✅ ДОБАВЛЯЕМ sourceMeta параметр
) (*Task, error) {

	if subject == "" {
		return nil, errors.New("subject is required")
	}
	if description == "" {
		return nil, errors.New("description is required")
	}

	// ✅ ОБРАБАТЫВАЕМ nil SourceMeta
	if sourceMeta == nil {
		sourceMeta = make(map[string]interface{})
	}

	now := time.Now()
	task := &Task{
		ID:          GenerateTaskID(),
		Type:        taskType,
		Subject:     subject,
		Description: description,
		Status:      TaskStatusOpen,
		Priority:    PriorityMedium,
		ReporterID:  reporterID,
		Source:      SourceInternal,
		SourceMeta:  sourceMeta, // ✅ ИСПОЛЬЗУЕМ sourceMeta вместо пустого map
		Tags:        []string{},
		Participants: []Participant{
			{
				UserID:   reporterID,
				Role:     RoleReporter,
				JoinedAt: now,
			},
		},
		Messages:  []Message{},
		History:   []TaskEvent{},
		CreatedAt: now,
		UpdatedAt: now,
	}

	task.addHistoryEvent("created", reporterID, nil, nil, "Задача создана")

	return task, nil
}

// NewSupportTask создает задачу поддержки
func NewSupportTask(
	subject string,
	description string,
	customerID string,
	reporterID string,
	source TaskSource,
	sourceMeta map[string]interface{}, // ✅ ДОБАВЛЯЕМ sourceMeta параметр
) (*Task, error) {

	task, err := NewTask(TaskTypeSupport, subject, description, reporterID, sourceMeta) // ✅ ПЕРЕДАЕМ sourceMeta
	if err != nil {
		return nil, err
	}

	task.CustomerID = &customerID
	task.Source = source

	return task, nil
}

// NewSubTask создает подзадачу
func NewSubTask(
	parentID string,
	subject string,
	description string,
	reporterID string,
	sourceMeta map[string]interface{}, // ✅ ДОБАВЛЯЕМ sourceMeta параметр
) (*Task, error) {

	task, err := NewTask(TaskTypeSubTask, subject, description, reporterID, sourceMeta) // ✅ ПЕРЕДАЕМ sourceMeta
	if err != nil {
		return nil, err
	}

	task.ParentID = &parentID

	return task, nil
}

// AddMessage добавляет сообщение в задачу
func (t *Task) AddMessage(authorID, content string, messageType MessageType) error {
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
	if messageType != MessageTypeSystem {
		t.addParticipantIfNotExists(authorID, RoleParticipant)
	}

	return nil
}

// ChangeStatus изменяет статус задачи
func (t *Task) ChangeStatus(newStatus TaskStatus, userID string) error {
	if !t.isValidStatusTransition(newStatus) {
		return fmt.Errorf("invalid status transition from %s to %s", t.Status, newStatus)
	}

	oldStatus := t.Status
	t.Status = newStatus
	t.UpdatedAt = time.Now()

	// Обновляем временные метки
	now := time.Now()
	switch newStatus {
	case TaskStatusResolved:
		t.ResolvedAt = &now
	case TaskStatusClosed:
		t.ClosedAt = &now
	case TaskStatusOpen:
		// Сброс при возврате к открытому
		t.ResolvedAt = nil
		t.ClosedAt = nil
	}

	// Записываем в историю (только коды статусов, без локализации)
	message := fmt.Sprintf("Статус изменен: %s → %s", oldStatus, newStatus)
	t.addHistoryEvent("status_changed", userID, oldStatus, newStatus, message)

	return nil
}

// Assign назначает исполнителя
func (t *Task) Assign(assigneeID string, userID string) error {
	if assigneeID == "" {
		return errors.New("assignee ID is required")
	}

	oldAssignee := t.AssigneeID
	t.AssigneeID = assigneeID
	t.UpdatedAt = time.Now()

	// Добавляем исполнителя в участники
	t.addParticipantIfNotExists(assigneeID, RoleAssignee)

	// Записываем в историю
	message := fmt.Sprintf("Исполнитель назначен: %s", assigneeID)
	t.addHistoryEvent("assignee_changed", userID, oldAssignee, assigneeID, message)

	return nil
}

// AddTag добавляет тег к задаче
func (t *Task) AddTag(tag string) {
	for _, existingTag := range t.Tags {
		if existingTag == tag {
			return
		}
	}
	t.Tags = append(t.Tags, tag)
	t.UpdatedAt = time.Now()
}

// Вспомогательные методы
func (t *Task) addParticipantIfNotExists(userID string, role ParticipantRole) {
	for _, participant := range t.Participants {
		if participant.UserID == userID {
			return
		}
	}

	t.Participants = append(t.Participants, Participant{
		UserID:   userID,
		Role:     role,
		JoinedAt: time.Now(),
	})
}

func (t *Task) addHistoryEvent(eventType string, userID string, oldVal, newVal interface{}, message string) {
	event := TaskEvent{
		ID:        GenerateEventID(),
		Type:      eventType,
		UserID:    userID,
		OldValue:  oldVal,
		NewValue:  newVal,
		Timestamp: time.Now(),
		Message:   message,
	}
	t.History = append(t.History, event)
}

func (t *Task) isValidStatusTransition(newStatus TaskStatus) bool {
	validTransitions := map[TaskStatus][]TaskStatus{
		TaskStatusOpen:       {TaskStatusInProgress, TaskStatusResolved, TaskStatusClosed, TaskStatusCancelled},
		TaskStatusInProgress: {TaskStatusOpen, TaskStatusReview, TaskStatusResolved, TaskStatusClosed, TaskStatusCancelled},
		TaskStatusReview:     {TaskStatusInProgress, TaskStatusResolved, TaskStatusClosed},
		TaskStatusResolved:   {TaskStatusOpen, TaskStatusInProgress, TaskStatusClosed},
		TaskStatusClosed:     {TaskStatusOpen},
		TaskStatusCancelled:  {TaskStatusOpen},
	}

	for _, validStatus := range validTransitions[t.Status] {
		if validStatus == newStatus {
			return true
		}
	}
	return false
}

// DisplayName возвращает отображаемое название статуса
func (s TaskStatus) DisplayName() string {
	names := map[TaskStatus]string{
		TaskStatusOpen:       "Открыта",
		TaskStatusInProgress: "В работе",
		TaskStatusReview:     "На проверке",
		TaskStatusResolved:   "Решена",
		TaskStatusClosed:     "Закрыта",
		TaskStatusCancelled:  "Отменена",
	}
	return names[s]
}

// GenerateTaskID генерирует ID для задачи
func GenerateTaskID() string {
	return fmt.Sprintf("TASK-%d", time.Now().UnixNano())
}

// GenerateMessageID генерирует ID для сообщения
func GenerateMessageID() string {
	return fmt.Sprintf("MSG-%d", time.Now().UnixNano())
}

// GenerateEventID генерирует ID для события
func GenerateEventID() string {
	return fmt.Sprintf("EVT-%d", time.Now().UnixNano())
}
