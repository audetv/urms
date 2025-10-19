// internal/core/domain/business_rules.go
package domain

import (
	"fmt"
	"time"
)

// BusinessRules содержит бизнес-правила для тикетов
type BusinessRules struct {
	MaxOpenTicketsPerAssignee int
	AutoCloseAfterResolution  time.Duration
	EscalationRules           []EscalationRule
}

// EscalationRule правило эскалации
type EscalationRule struct {
	Condition EscalationCondition
	Action    EscalationAction
	Priority  Priority
}

// EscalationCondition условие эскалации
type EscalationCondition struct {
	Field    string
	Operator string
	Value    interface{}
}

// EscalationAction действие при эскалации
type EscalationAction struct {
	Type        string
	AssigneeID  string
	NotifyUsers []string
}

// CanAssign проверяет можно ли назначить тикет на исполнителя
func (r *BusinessRules) CanAssign(assigneeID string, currentAssignments int) error {
	if currentAssignments >= r.MaxOpenTicketsPerAssignee {
		return fmt.Errorf("assignee %s has too many open tickets (%d/%d)",
			assigneeID, currentAssignments, r.MaxOpenTicketsPerAssignee)
	}
	return nil
}

// ShouldAutoClose проверяет нужно ли автоматически закрыть тикет
func (r *BusinessRules) ShouldAutoClose(resolvedAt time.Time) bool {
	if resolvedAt.IsZero() {
		return false
	}
	return time.Since(resolvedAt) > r.AutoCloseAfterResolution
}

// CheckEscalation проверяет условия эскалации
func (r *BusinessRules) CheckEscalation(ticket *Ticket) *EscalationAction {
	for _, rule := range r.EscalationRules {
		if r.matchesCondition(ticket, rule.Condition) {
			return &rule.Action
		}
	}
	return nil
}

func (r *BusinessRules) matchesCondition(ticket *Ticket, condition EscalationCondition) bool {
	switch condition.Field {
	case "priority":
		return ticket.Priority == condition.Value.(Priority)
	case "status_age":
		age := time.Since(ticket.UpdatedAt)
		return age > condition.Value.(time.Duration)
	case "customer_priority":
		// Здесь можно добавить логику проверки приоритета клиента
		return false
	}
	return false
}
