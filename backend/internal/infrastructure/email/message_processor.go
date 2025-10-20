// internal/infrastructure/email/advanced_message_processor.go
package email

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/audetv/urms/internal/core/domain"
	"github.com/audetv/urms/internal/core/ports"
)

// MessageProcessor реализация с интеграцией Task Management
type MessageProcessor struct {
	taskService     ports.TaskService
	customerService ports.CustomerService
	logger          ports.Logger
}

// NewMessageProcessor создает новый экземпляр процессора
func NewMessageProcessor(
	taskService ports.TaskService,
	customerService ports.CustomerService,
	logger ports.Logger,
) ports.MessageProcessor {
	return &MessageProcessor{
		taskService:     taskService,
		customerService: customerService,
		logger:          logger,
	}
}

// ProcessIncomingEmail обрабатывает входящие email сообщения с интеграцией Task Management
func (p *MessageProcessor) ProcessIncomingEmail(ctx context.Context, email domain.EmailMessage) error {
	p.logger.Info(ctx, "Processing incoming email with task integration",
		"message_id", email.MessageID,
		"from", email.From,
		"subject", email.Subject,
		"in_reply_to", email.InReplyTo,
		"operation", "advanced_process_incoming_email")

	// 1. Валидация email
	if err := p.validateIncomingEmail(ctx, email); err != nil {
		p.logger.Error(ctx, "Incoming email validation failed",
			"message_id", email.MessageID,
			"error", err.Error())
		return fmt.Errorf("email validation failed: %w", err)
	}

	// 2. Поиск или создание клиента
	customer, err := p.findOrCreateCustomer(ctx, email)
	if err != nil {
		p.logger.Error(ctx, "Failed to find or create customer",
			"message_id", email.MessageID,
			"from", email.From,
			"error", err.Error())
		return fmt.Errorf("customer management failed: %w", err)
	}

	// 3. Поиск существующей задачи по Thread-ID
	existingTask, err := p.findExistingTaskByThread(ctx, email)
	if err != nil {
		p.logger.Error(ctx, "Failed to search for existing task",
			"message_id", email.MessageID,
			"error", err.Error())
		// Продолжаем обработку, создаем новую задачу
	}

	var task *domain.Task
	if existingTask != nil {
		// 4a. Добавление сообщения в существующую задачу
		task, err = p.addMessageToExistingTask(ctx, existingTask, email, customer.ID)
		if err != nil {
			p.logger.Error(ctx, "Failed to add message to existing task",
				"task_id", existingTask.ID,
				"message_id", email.MessageID,
				"error", err.Error())
			return fmt.Errorf("failed to update existing task: %w", err)
		}
		p.logger.Info(ctx, "Message added to existing task",
			"task_id", existingTask.ID,
			"message_id", email.MessageID)
	} else {
		// 4b. Создание новой задачи
		task, err = p.createNewTaskFromEmail(ctx, email, customer.ID)
		if err != nil {
			p.logger.Error(ctx, "Failed to create new task from email",
				"message_id", email.MessageID,
				"error", err.Error())
			return fmt.Errorf("failed to create task: %w", err)
		}
		p.logger.Info(ctx, "New task created from email",
			"task_id", task.ID,
			"message_id", email.MessageID)
	}

	// 5. Автоматическое назначение (базовая логика)
	if task.AssigneeID == "" {
		task, err = p.autoAssignTask(ctx, task)
		if err != nil {
			p.logger.Warn(ctx, "Auto-assignment failed, task remains unassigned",
				"task_id", task.ID,
				"error", err.Error())
		} else {
			p.logger.Info(ctx, "Task auto-assigned",
				"task_id", task.ID,
				"assignee_id", task.AssigneeID)
		}
	}

	p.logger.Info(ctx, "Incoming email processed successfully with task integration",
		"message_id", email.MessageID,
		"task_id", task.ID,
		"customer_id", customer.ID,
		"operation", "email_task_integration_complete")

	return nil
}

// ProcessOutgoingEmail обрабатывает исходящие email сообщения
func (p *MessageProcessor) ProcessOutgoingEmail(ctx context.Context, email domain.EmailMessage) error {
	p.logger.Info(ctx, "Processing outgoing email with task integration",
		"message_id", email.MessageID,
		"to", email.To,
		"subject", email.Subject,
		"operation", "advanced_process_outgoing_email")

	// 1. Валидация исходящего сообщения
	if err := p.validateOutgoingEmail(ctx, email); err != nil {
		p.logger.Error(ctx, "Outgoing email validation failed",
			"message_id", email.MessageID,
			"error", err.Error())
		return fmt.Errorf("outgoing email validation failed: %w", err)
	}

	// 2. Если email связан с задачей, добавляем сообщение
	if email.RelatedTicketID != nil {
		task, err := p.taskService.GetTask(ctx, *email.RelatedTicketID)
		if err != nil {
			p.logger.Error(ctx, "Failed to get related task for outgoing email",
				"task_id", *email.RelatedTicketID,
				"message_id", email.MessageID,
				"error", err.Error())
		} else {
			// Добавляем внутреннее сообщение в задачу
			messageReq := ports.AddMessageRequest{
				AuthorID:  "system", // TODO: Заменить на реального пользователя
				Content:   fmt.Sprintf("Отправлен ответ по email: %s", email.Subject),
				Type:      domain.MessageTypeInternal,
				IsPrivate: true,
			}
			_, err = p.taskService.AddMessage(ctx, task.ID, messageReq)
			if err != nil {
				p.logger.Warn(ctx, "Failed to add outgoing message to task",
					"task_id", task.ID,
					"message_id", email.MessageID,
					"error", err.Error())
			} else {
				p.logger.Info(ctx, "Outgoing email logged in task",
					"task_id", task.ID,
					"message_id", email.MessageID)
			}
		}
	}

	p.logger.Info(ctx, "Outgoing email processed successfully",
		"message_id", email.MessageID,
		"operation", "outgoing_email_processed")

	return nil
}

// findOrCreateCustomer находит или создает клиента по email
func (p *MessageProcessor) findOrCreateCustomer(ctx context.Context, email domain.EmailMessage) (*domain.Customer, error) {
	customerName := p.extractNameFromEmail(string(email.From))

	customer, err := p.customerService.FindOrCreateByEmail(ctx, string(email.From), customerName)
	if err != nil {
		return nil, fmt.Errorf("failed to find or create customer: %w", err)
	}

	return customer, nil
}

// findExistingTaskByThread ищет существующую задачу по Thread-ID
func (p *MessageProcessor) findExistingTaskByThread(ctx context.Context, email domain.EmailMessage) (*domain.Task, error) {
	// ВРЕМЕННАЯ РЕАЛИЗАЦИЯ: Поиск по Thread-ID будет реализован в Phase 3
	// когда добавим полноценную поддержку email threading

	p.logger.Debug(ctx, "Thread-based task search not yet implemented - creating new task",
		"message_id", email.MessageID,
		"in_reply_to", email.InReplyTo,
		"references_count", len(email.References))

	// TODO: Реализовать когда добавим поиск по SourceMeta в TaskRepository
	// План реализации:
	// 1. Добавить поле SourceMeta в TaskQuery
	// 2. Реализовать поиск по message_id, in_reply_to в репозиториях
	// 3. Добавить индексацию для производительности

	return nil, nil
}

// createNewTaskFromEmail создает новую задачу из email
func (p *MessageProcessor) createNewTaskFromEmail(ctx context.Context, email domain.EmailMessage, customerID string) (*domain.Task, error) {
	// Определяем приоритет на основе содержимого
	priority := p.determinePriority(ctx, email)

	// Определяем категорию
	category := p.determineCategory(ctx, email)

	req := ports.CreateSupportTaskRequest{
		Subject:     p.normalizeSubject(email.Subject),
		Description: p.buildTaskDescription(email),
		CustomerID:  customerID,
		ReporterID:  "system", // TODO: Заменить на реального пользователя
		Source:      domain.SourceEmail,
		SourceMeta:  p.buildSourceMeta(email),
		Priority:    priority,
		Category:    category,
		Tags:        p.extractTags(ctx, email),
	}

	task, err := p.taskService.CreateSupportTask(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create support task: %w", err)
	}

	return task, nil
}

// addMessageToExistingTask добавляет сообщение в существующую задачу
func (p *MessageProcessor) addMessageToExistingTask(ctx context.Context, task *domain.Task, email domain.EmailMessage, customerID string) (*domain.Task, error) {
	messageReq := ports.AddMessageRequest{
		AuthorID:  customerID,
		Content:   p.buildMessageContent(email),
		Type:      domain.MessageTypeCustomer,
		IsPrivate: false,
	}

	updatedTask, err := p.taskService.AddMessage(ctx, task.ID, messageReq)
	if err != nil {
		return nil, fmt.Errorf("failed to add message to task: %w", err)
	}

	return updatedTask, nil
}

// autoAssignTask автоматически назначает задачу
func (p *MessageProcessor) autoAssignTask(ctx context.Context, task *domain.Task) (*domain.Task, error) {
	// Базовая логика назначения - по категории
	// TODO: Реализовать более сложную логику назначения

	// Временно возвращаем задачу без изменений
	// Реальная логика назначения будет в Phase 4 с AI
	return task, nil
}

// Вспомогательные методы

func (p *MessageProcessor) validateIncomingEmail(ctx context.Context, email domain.EmailMessage) error {
	if email.MessageID == "" {
		return fmt.Errorf("message ID is required")
	}
	if email.From == "" {
		return fmt.Errorf("sender address is required")
	}
	if len(email.To) == 0 && len(email.CC) == 0 {
		return fmt.Errorf("no recipients found")
	}
	return nil
}

func (p *MessageProcessor) validateOutgoingEmail(ctx context.Context, email domain.EmailMessage) error {
	if email.MessageID == "" {
		return fmt.Errorf("message ID is required")
	}
	if len(email.To) == 0 {
		return fmt.Errorf("at least one recipient is required")
	}
	if email.Subject == "" {
		return fmt.Errorf("subject is required for outgoing emails")
	}
	return nil
}

func (p *MessageProcessor) extractNameFromEmail(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) > 0 {
		namePart := strings.ReplaceAll(parts[0], ".", " ")
		return strings.Title(namePart) // Простая эвристика
	}
	return "Customer"
}

func (p *MessageProcessor) normalizeSubject(subject string) string {
	// Убираем префиксы типа "Re:", "Fwd:" и т.д.
	prefixes := []string{"Re:", "Fwd:", "FW:", "RE:", "Ответ:", "FWD:"}
	result := subject

	for _, prefix := range prefixes {
		if strings.HasPrefix(strings.ToUpper(result), strings.ToUpper(prefix)) {
			result = strings.TrimSpace(result[len(prefix):])
		}
	}

	if result == "" {
		return "Без темы"
	}

	return result
}

func (p *MessageProcessor) determinePriority(ctx context.Context, email domain.EmailMessage) domain.Priority {
	// Базовая логика определения приоритета
	content := strings.ToLower(email.Subject + " " + email.BodyText)

	urgencyKeywords := []string{"срочно", "urgent", "critical", "важно", "error", "ошибка"}
	for _, keyword := range urgencyKeywords {
		if strings.Contains(content, keyword) {
			return domain.PriorityHigh
		}
	}

	return domain.PriorityMedium
}

func (p *MessageProcessor) determineCategory(ctx context.Context, email domain.EmailMessage) string {
	// Базовая логика категоризации
	content := strings.ToLower(email.Subject + " " + email.BodyText)

	categories := map[string][]string{
		"technical": {"ошибка", "error", "bug", "сломал", "не работает"},
		"billing":   {"оплата", "payment", "счет", "invoice", "bill"},
		"general":   {"вопрос", "question", "помощь", "help"},
	}

	for category, keywords := range categories {
		for _, keyword := range keywords {
			if strings.Contains(content, keyword) {
				return category
			}
		}
	}

	return "general"
}

func (p *MessageProcessor) buildTaskDescription(email domain.EmailMessage) string {
	var description strings.Builder

	description.WriteString("Заявка создана автоматически из входящего email.\n\n")
	description.WriteString("От: " + string(email.From) + "\n")
	description.WriteString("Тема: " + email.Subject + "\n")
	description.WriteString("Дата: " + time.Now().Format("2006-01-02 15:04:05") + "\n\n")

	if email.BodyText != "" {
		description.WriteString("Содержимое сообщения:\n")
		description.WriteString(email.BodyText)
	} else if email.BodyHTML != "" {
		description.WriteString("Содержимое сообщения (HTML):\n")
		// TODO: Конвертировать HTML в текст
		description.WriteString("[HTML content]")
	}

	return description.String()
}

func (p *MessageProcessor) buildMessageContent(email domain.EmailMessage) string {
	var content strings.Builder

	content.WriteString("Новое сообщение от клиента:\n\n")

	if email.BodyText != "" {
		content.WriteString(email.BodyText)
	} else if email.BodyHTML != "" {
		content.WriteString("[HTML content]")
	}

	if len(email.Attachments) > 0 {
		content.WriteString(fmt.Sprintf("\n\nВложения: %d файл(ов)", len(email.Attachments)))
	}

	return content.String()
}

func (p *MessageProcessor) buildSourceMeta(email domain.EmailMessage) map[string]interface{} {
	meta := map[string]interface{}{
		"message_id":  email.MessageID,
		"in_reply_to": email.InReplyTo,
		"references":  email.References,
		"headers":     email.Headers,
	}

	if len(email.Attachments) > 0 {
		attachments := make([]map[string]interface{}, len(email.Attachments))
		for i, att := range email.Attachments {
			attachments[i] = map[string]interface{}{
				"name":         att.Name,
				"content_type": att.ContentType,
				"size":         att.Size,
			}
		}
		meta["attachments"] = attachments
	}

	return meta
}

func (p *MessageProcessor) extractTags(ctx context.Context, email domain.EmailMessage) []string {
	tags := []string{"email", "auto-created"}

	// Добавляем теги на основе содержимого
	content := strings.ToLower(email.Subject + " " + email.BodyText)

	if strings.Contains(content, "срочно") || strings.Contains(content, "urgent") {
		tags = append(tags, "urgent")
	}

	if len(email.Attachments) > 0 {
		tags = append(tags, "has-attachments")
	}

	return tags
}
