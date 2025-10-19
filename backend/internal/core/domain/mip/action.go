// backend/internal/core/domain/action.go
package domain

import "time"

// Action - действие в процессе управления
// Действие представляет собой конкретный шаг в тактике управления,
// который может быть выполнен системой или пользователем
type Action struct {
	// ID - уникальный идентификатор действия
	ID string

	// Type - тип действия (создание, изменение, удаление, анализ и т.д.)
	Type ActionType

	// Description - человеко-читаемое описание действия
	Description string

	// Executor - кто выполняет действие (система, пользователь, внешний сервис)
	Executor string

	// Parameters - параметры выполнения действия
	Parameters map[string]interface{}

	// Conditions - условия, при которых действие может быть выполнено
	Conditions []Condition

	// ExpectedOutcome - ожидаемый результат выполнения действия
	ExpectedOutcome *ExpectedResult

	// Timestamps - временные метки выполнения
	StartedAt   *time.Time
	CompletedAt *time.Time
	Deadline    *time.Time

	// Status - текущий статус выполнения действия
	Status ActionStatus

	// Result - фактический результат выполнения (заполняется после завершения)
	Result *ActionResult
}

// ActionType - тип действия
type ActionType string

const (
	ActionTypeCreate   ActionType = "create"
	ActionTypeUpdate   ActionType = "update"
	ActionTypeDelete   ActionType = "delete"
	ActionTypeAnalyze  ActionType = "analyze"
	ActionTypeNotify   ActionType = "notify"
	ActionTypeApprove  ActionType = "approve"
	ActionTypeEscalate ActionType = "escalate"
	ActionTypeResolve  ActionType = "resolve"
)

// ActionStatus - статус выполнения действия
type ActionStatus string

const (
	ActionStatusPending    ActionStatus = "pending"
	ActionStatusInProgress ActionStatus = "in_progress"
	ActionStatusCompleted  ActionStatus = "completed"
	ActionStatusFailed     ActionStatus = "failed"
	ActionStatusCancelled  ActionStatus = "cancelled"
)

// ActionResult - результат выполнения действия
type ActionResult struct {
	// Success - успешно ли выполнено действие
	Success bool

	// Data - данные, полученные в результате выполнения
	Data map[string]interface{}

	// Error - ошибка, если действие завершилось неудачно
	Error error

	// Metrics - метрики выполнения (время, ресурсы и т.д.)
	Metrics map[string]interface{}
}
