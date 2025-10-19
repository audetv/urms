// internal/core/domain/types.go
package domain

// TaskSource представляет источник задачи
type TaskSource string

const (
	SourceEmail    TaskSource = "email"    // Email
	SourceTelegram TaskSource = "telegram" // Telegram
	SourceWebForm  TaskSource = "web_form" // Веб-форма
	SourceAPI      TaskSource = "api"      // API
	SourceInternal TaskSource = "internal" // Внутренняя
	SourceGitHub   TaskSource = "github"   // GitHub (для будущего)
)

// Priority представляет приоритет задачи
type Priority string

const (
	PriorityLow      Priority = "low"      // Низкий
	PriorityMedium   Priority = "medium"   // Средний
	PriorityHigh     Priority = "high"     // Высокий
	PriorityCritical Priority = "critical" // Критический
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
