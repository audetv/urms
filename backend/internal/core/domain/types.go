// internal/core/domain/types.go
package domain

// TicketStatus представляет статус заявки
type TicketStatus string

const (
	StatusOpen       TicketStatus = "open"        // Открыта
	StatusInProgress TicketStatus = "in_progress" // В работе
	StatusResolved   TicketStatus = "resolved"    // Решена
	StatusClosed     TicketStatus = "closed"      // Закрыта
)

// DisplayName возвращает отображаемое название статуса
func (s TicketStatus) DisplayName() string {
	names := map[TicketStatus]string{
		StatusOpen:       "Открыта",
		StatusInProgress: "В работе",
		StatusResolved:   "Решена",
		StatusClosed:     "Закрыта",
	}
	return names[s]
}

// Priority представляет приоритет заявки
type Priority string

const (
	PriorityLow      Priority = "low"      // Низкий
	PriorityMedium   Priority = "medium"   // Средний
	PriorityHigh     Priority = "high"     // Высокий
	PriorityCritical Priority = "critical" // Критический
)

// DisplayName возвращает отображаемое название приоритета
func (p Priority) DisplayName() string {
	names := map[Priority]string{
		PriorityLow:      "Низкий",
		PriorityMedium:   "Средний",
		PriorityHigh:     "Высокий",
		PriorityCritical: "Критический",
	}
	return names[p]
}

// TicketSource представляет источник заявки
type TicketSource string

const (
	SourceEmail    TicketSource = "email"    // Email
	SourceTelegram TicketSource = "telegram" // Telegram
	SourceWebForm  TicketSource = "web_form" // Веб-форма
	SourceAPI      TicketSource = "api"      // API
	SourceInternal TicketSource = "internal" // Внутренняя
)

// ParticipantRole представляет роль участника
type ParticipantRole string

const (
	RoleReporter    ParticipantRole = "reporter"    // Автор
	RoleAssignee    ParticipantRole = "assignee"    // Исполнитель
	RoleReviewer    ParticipantRole = "reviewer"    // Рецензент
	RoleWatcher     ParticipantRole = "watcher"     // Наблюдатель
	RoleParticipant ParticipantRole = "participant" // Участник
)

// MessageType представляет тип сообщения
type MessageType string

const (
	MessageTypeCustomer MessageType = "customer" // Сообщение от клиента
	MessageTypeInternal MessageType = "internal" // Внутреннее сообщение
	MessageTypeSystem   MessageType = "system"   // Системное сообщение
)

// SubTicketStatus представляет статус подзадачи
type SubTicketStatus string

const (
	SubTicketStatusOpen       SubTicketStatus = "open"
	SubTicketStatusInProgress SubTicketStatus = "in_progress"
	SubTicketStatusCompleted  SubTicketStatus = "completed"
	SubTicketStatusCancelled  SubTicketStatus = "cancelled"
)
