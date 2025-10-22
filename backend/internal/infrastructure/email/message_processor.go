// internal/infrastructure/email/message_processor.go
package email

import (
	"context"
	"fmt"
	"strings"

	"github.com/audetv/urms/internal/core/domain"
	"github.com/audetv/urms/internal/core/ports"
	"github.com/audetv/urms/internal/core/services"
)

// MessageProcessor реализация с интеграцией Task Management
type MessageProcessor struct {
	taskService     ports.TaskService
	customerService ports.CustomerService
	emailGateway    ports.EmailGateway
	headerFilter    *HeaderFilter
	searchConfig    ports.EmailSearchConfigProvider // ✅ ДОБАВЛЯЕМ конфигурационный порт
	searchService   *services.EmailSearchService    // ✅ ДОБАВЛЯЕМ сервис поиска
	logger          ports.Logger
}

// NewMessageProcessor создает новый экземпляр процессора
func NewMessageProcessor(
	taskService ports.TaskService,
	customerService ports.CustomerService,
	emailGateway ports.EmailGateway,
	searchConfig ports.EmailSearchConfigProvider, // ✅ ДОБАВЛЯЕМ dependency
	logger ports.Logger,
) ports.MessageProcessor {

	// ✅ СОЗДАЕМ сервис поиска
	searchService := services.NewEmailSearchService(searchConfig, logger)

	return &MessageProcessor{
		taskService:     taskService,
		customerService: customerService,
		emailGateway:    emailGateway,
		headerFilter:    NewHeaderFilter(logger),
		searchConfig:    searchConfig,  // ✅ СОХРАНЯЕМ
		searchService:   searchService, // ✅ СОХРАНЯЕМ
		logger:          logger,
	}
}

// ProcessIncomingEmail обрабатывает входящие email сообщения с интеграцией Task Management
func (p *MessageProcessor) ProcessIncomingEmail(ctx context.Context, email domain.EmailMessage) error {
	p.logger.Info(ctx, "Processing incoming email with ENHANCED THREAD SEARCH",
		"message_id", email.MessageID,
		"from", email.From,
		"subject", email.Subject,
		"operation", "enhanced_thread_search_process")

	// 1. Валидация email
	if err := p.validateIncomingEmail(ctx, email); err != nil {
		p.logger.Error(ctx, "Incoming email validation failed",
			"message_id", email.MessageID,
			"error", err.Error())
		return fmt.Errorf("email validation failed: %w", err)
	}

	// 2. ФИЛЬТРАЦИЯ ЗАГОЛОВКОВ
	emailHeaders, err := p.headerFilter.FilterEssentialHeaders(ctx, &email)
	if err != nil {
		p.logger.Error(ctx, "Failed to filter essential headers",
			"message_id", email.MessageID,
			"error", err.Error())
		return fmt.Errorf("headers filtering failed: %w", err)
	}

	// 3. Поиск или создание клиента
	customer, err := p.findOrCreateCustomer(ctx, email)
	if err != nil {
		p.logger.Error(ctx, "Failed to find or create customer",
			"message_id", email.MessageID,
			"from", email.From,
			"error", err.Error())
		return fmt.Errorf("customer management failed: %w", err)
	}

	// 4. ✅ УЛУЧШЕННЫЙ ПОИСК СУЩЕСТВУЮЩЕЙ ЗАДАЧИ
	existingTask, err := p.findExistingTaskByThreadEnhanced(ctx, email, emailHeaders)
	if err != nil {
		p.logger.Error(ctx, "Failed to search for existing task with enhanced search",
			"message_id", email.MessageID,
			"error", err.Error())
		// Продолжаем обработку, создаем новую задачу
	}

	var task *domain.Task
	if existingTask != nil {
		// 5a. Добавление сообщения в существующую задачу
		task, err = p.addMessageToExistingTask(ctx, existingTask, email, customer.ID, emailHeaders)
		if err != nil {
			p.logger.Error(ctx, "Failed to add message to existing task",
				"task_id", existingTask.ID,
				"message_id", email.MessageID,
				"error", err.Error())
			return fmt.Errorf("failed to update existing task: %w", err)
		}
		p.logger.Info(ctx, "Message added to existing task found via ENHANCED search",
			"task_id", existingTask.ID,
			"message_id", email.MessageID)
	} else {
		// 5b. Создание новой задачи
		task, err = p.createNewTaskFromEmail(ctx, email, customer.ID, emailHeaders)
		if err != nil {
			p.logger.Error(ctx, "Failed to create new task from email",
				"message_id", email.MessageID,
				"error", err.Error())
			return fmt.Errorf("failed to create task: %w", err)
		}
		p.logger.Info(ctx, "New task created from email - no existing thread found",
			"task_id", task.ID,
			"message_id", email.MessageID)
	}

	// 6. Автоматическое назначение
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

	p.logger.Info(ctx, "Incoming email processed successfully with ENHANCED THREAD SEARCH",
		"message_id", email.MessageID,
		"task_id", task.ID,
		"customer_id", customer.ID,
		"headers_optimized", true,
		"enhanced_search_used", true,
		"operation", "enhanced_thread_search_complete")

	return nil
}

// findExistingTaskByThreadEnhanced - ОБНОВЛЕННАЯ ВЕРСИЯ С КОНФИГУРАЦИЕЙ
func (p *MessageProcessor) findExistingTaskByThreadEnhanced(ctx context.Context, email domain.EmailMessage, headers *domain.EmailHeaders) (*domain.Task, error) {
	if headers == nil {
		p.logger.Debug(ctx, "No headers provided for enhanced thread search")
		return nil, nil
	}

	// ✅ СТРАТЕГИЯ 1: Быстрый поиск по существующим threading данным
	existingTask, err := p.findExistingTaskByThread(ctx, headers)
	if err != nil {
		p.logger.Warn(ctx, "Standard thread search failed, trying enhanced search",
			"message_id", headers.MessageID,
			"error", err.Error())
	} else if existingTask != nil {
		p.logger.Info(ctx, "✅ Found existing task via standard search",
			"message_id", headers.MessageID,
			"task_id", existingTask.ID)
		return existingTask, nil
	}

	// ✅ СТРАТЕГИЯ 2: Enhanced IMAP search с КОНФИГУРАЦИЕЙ
	p.logger.Info(ctx, "🚀 Starting ENHANCED IMAP thread search with CONFIGURABLE parameters",
		"message_id", headers.MessageID,
		"subject", headers.Subject,
		"in_reply_to", headers.InReplyTo,
		"references_count", len(headers.References))

	// ✅ ПОЛУЧАЕМ КОНФИГУРАЦИЮ ДЛЯ ЛОГИРОВАНИЯ
	searchConfig, err := p.searchService.GetThreadSearchConfig(ctx)
	if err != nil {
		p.logger.Warn(ctx, "Failed to get search config, using enhanced search without config",
			"message_id", headers.MessageID,
			"error", err.Error())
	} else {
		p.logger.Info(ctx, "Using CONFIGURABLE search parameters",
			"default_days", searchConfig.DefaultDaysBack(),
			"extended_days", searchConfig.ExtendedDaysBack(),
			"max_days", searchConfig.MaxDaysBack(),
			"search_strategy", "extended_time_range+combined_criteria")
	}

	// Создаем критерии для thread-aware поиска
	threadCriteria := ports.ThreadSearchCriteria{
		MessageID:  headers.MessageID,
		InReplyTo:  headers.InReplyTo,
		References: headers.References,
		Subject:    p.normalizeSubject(headers.Subject),
		Mailbox:    "INBOX",
	}

	// Выполняем ENHANCED поиск через EmailGateway
	threadMessages, err := p.emailGateway.SearchThreadMessages(ctx, threadCriteria)
	if err != nil {
		p.logger.Warn(ctx, "Enhanced IMAP thread search failed",
			"message_id", headers.MessageID,
			"error", err.Error())
		return nil, nil // Продолжаем с созданием новой задачи
	}

	p.logger.Info(ctx, "Enhanced IMAP thread search completed",
		"message_id", headers.MessageID,
		"found_messages", len(threadMessages),
		"search_criteria", fmt.Sprintf("%+v", threadCriteria))

	// Если нашли сообщения в цепочке, ищем соответствующую задачу
	if len(threadMessages) > 0 {
		task := p.findTaskForThreadMessages(ctx, threadMessages)
		if task != nil {
			p.logger.Info(ctx, "✅ SUCCESS: Found existing task via ENHANCED IMAP search",
				"message_id", headers.MessageID,
				"task_id", task.ID,
				"thread_messages_found", len(threadMessages),
				"search_improvement", "configurable_extended_time_range")
			return task, nil
		}

		p.logger.Warn(ctx, "Found thread messages but no existing task - creating new task",
			"message_id", headers.MessageID,
			"thread_messages_count", len(threadMessages),
			"first_thread_message_id", safeGetMessageID(threadMessages))
	}

	p.logger.Info(ctx, "Enhanced thread search completed - creating new task",
		"message_id", headers.MessageID,
		"reason", "no_existing_task_found_with_enhanced_search")

	return nil, nil
}

// ✅ ВСПОМОГАТЕЛЬНЫЙ МЕТОД: findTaskForThreadMessages
func (p *MessageProcessor) findTaskForThreadMessages(ctx context.Context, messages []domain.EmailMessage) *domain.Task {
	// Ищем задачу по Message-ID первого найденного сообщения в цепочке
	for _, msg := range messages {
		if msg.MessageID == "" {
			continue
		}

		searchMeta := map[string]interface{}{
			"message_id": msg.MessageID,
		}

		tasks, err := p.taskService.FindBySourceMeta(ctx, searchMeta)
		if err != nil {
			p.logger.Warn(ctx, "Failed to search task for thread message",
				"message_id", msg.MessageID,
				"error", err.Error())
			continue
		}

		if len(tasks) > 0 {
			return &tasks[0]
		}
	}

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

// findExistingTaskByThread ищет существующую задачу по Thread-ID с использованием EmailHeaders
func (p *MessageProcessor) findExistingTaskByThread(ctx context.Context, headers *domain.EmailHeaders) (*domain.Task, error) {
	if headers == nil {
		p.logger.Debug(ctx, "No headers provided for thread search")
		return nil, nil
	}

	// Создаем критерии поиска по Thread-ID из EmailHeaders
	searchMeta := make(map[string]interface{})

	if headers.MessageID != "" {
		searchMeta["message_id"] = headers.MessageID
	}
	if headers.InReplyTo != "" {
		searchMeta["in_reply_to"] = headers.InReplyTo
	}
	if len(headers.References) > 0 {
		searchMeta["references"] = headers.References
	}

	// ✅ ЛОГИРУЕМ КРИТЕРИИ ПОИСКА С НОВОЙ АРХИТЕКТУРОЙ
	p.logger.Debug(ctx, "Email threading search with OPTIMIZED headers",
		"message_id", headers.MessageID,
		"in_reply_to", headers.InReplyTo,
		"references_count", len(headers.References),
		"search_meta", searchMeta)

	// Если нет критериев для поиска - создаем новую задачу
	if len(searchMeta) == 0 {
		p.logger.Debug(ctx, "No thread criteria found for email",
			"message_id", headers.MessageID)
		return nil, nil
	}

	// Ищем существующие задачи по Thread-ID
	tasks, err := p.taskService.FindBySourceMeta(ctx, searchMeta)
	if err != nil {
		p.logger.Warn(ctx, "Failed to search tasks by source meta",
			"message_id", headers.MessageID,
			"error", err.Error())
		return nil, err
	}

	// ✅ ЛОГИРУЕМ РЕЗУЛЬТАТЫ ПОИСКА С НОВОЙ АРХИТЕКТУРОЙ
	p.logger.Debug(ctx, "Email threading search results with OPTIMIZED headers",
		"message_id", headers.MessageID,
		"tasks_found", len(tasks),
		"search_criteria", searchMeta)

	// Возвращаем самую релевантную задачу (первую найденную)
	if len(tasks) > 0 {
		p.logger.Info(ctx, "Found existing task for email thread with OPTIMIZED headers",
			"message_id", headers.MessageID,
			"task_id", tasks[0].ID,
			"matches_count", len(tasks),
			"search_criteria", searchMeta)
		return &tasks[0], nil
	}

	p.logger.Debug(ctx, "No existing task found for email thread with OPTIMIZED headers",
		"message_id", headers.MessageID,
		"in_reply_to", headers.InReplyTo,
		"references_count", len(headers.References))
	return nil, nil
}

// createNewTaskFromEmail создает новую задачу из email с использованием EmailHeaders
func (p *MessageProcessor) createNewTaskFromEmail(ctx context.Context, email domain.EmailMessage, customerID string, headers *domain.EmailHeaders) (*domain.Task, error) {
	// Определяем приоритет на основе содержимого
	priority := p.determinePriority(ctx, email)

	// Определяем категорию
	category := p.determineCategory(ctx, email)

	// ✅ ИСПОЛЬЗУЕМ НОВУЮ АРХИТЕКТУРУ ДЛЯ SOURCE META
	sourceMeta := p.buildSourceMeta(headers, email)

	req := ports.CreateSupportTaskRequest{
		Subject:     p.normalizeSubject(headers.Subject),
		Description: p.buildTaskDescription(email, headers),
		CustomerID:  customerID,
		ReporterID:  "system",
		Source:      domain.SourceEmail,
		SourceMeta:  sourceMeta,
		Priority:    priority,
		Category:    category,
		Tags:        p.extractTags(ctx, email),
	}

	// ✅ ЛОГИРУЕМ СОЗДАНИЕ ЗАДАЧИ С OPTIMIZED SOURCE META
	p.logger.Info(ctx, "Creating new task with OPTIMIZED source meta",
		"message_id", headers.MessageID,
		"in_reply_to", headers.InReplyTo,
		"references_count", len(headers.References),
		"source_meta_size", fmt.Sprintf("%d keys", len(sourceMeta)),
		"headers_optimized", true)

	task, err := p.taskService.CreateSupportTask(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create support task: %w", err)
	}

	return task, nil
}

// addMessageToExistingTask добавляет сообщение в существующую задачу
func (p *MessageProcessor) addMessageToExistingTask(ctx context.Context, task *domain.Task, email domain.EmailMessage, customerID string, headers *domain.EmailHeaders) (*domain.Task, error) {
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

// buildTaskDescription создает описание задачи из email с использованием EmailHeaders
func (p *MessageProcessor) buildTaskDescription(email domain.EmailMessage, headers *domain.EmailHeaders) string {
	var description strings.Builder

	description.WriteString("Заявка создана автоматически из входящего email.\n\n")
	description.WriteString("От: " + string(headers.From) + "\n")
	description.WriteString("Тема: " + headers.Subject + "\n")
	description.WriteString("Дата: " + headers.Date.Format("2006-01-02 15:04:05") + "\n\n")

	// ✅ ИСПОЛЬЗУЕМ РЕАЛЬНОЕ СОДЕРЖАНИЕ для описания задачи
	if email.BodyText != "" {
		description.WriteString("Содержимое сообщения:\n")
		// Обрезаем слишком длинные сообщения для описания
		if len(email.BodyText) > 500 {
			description.WriteString(email.BodyText[:500] + "...")
		} else {
			description.WriteString(email.BodyText)
		}
	} else if email.BodyHTML != "" {
		description.WriteString("Содержимое сообщения (HTML):\n")
		description.WriteString("[HTML content - see messages for full text]")
	} else {
		description.WriteString("Сообщение не содержит текста.")
	}

	return description.String()
}

// buildMessageContent создает содержимое сообщения из email
func (p *MessageProcessor) buildMessageContent(email domain.EmailMessage) string {
	var content strings.Builder

	// ✅ ИСПОЛЬЗУЕМ РЕАЛЬНОЕ СОДЕРЖАНИЕ ПИСЬМА вместо заглушки
	if email.BodyText != "" {
		content.WriteString(email.BodyText)
	} else if email.BodyHTML != "" {
		// TODO: Конвертировать HTML в текст
		content.WriteString("[HTML content - needs conversion]")
	} else {
		content.WriteString("[No message content]")
	}

	// Добавляем информацию о вложениях
	if len(email.Attachments) > 0 {
		content.WriteString(fmt.Sprintf("\n\n📎 Вложения: %d файл(ов)", len(email.Attachments)))
		for _, att := range email.Attachments {
			content.WriteString(fmt.Sprintf("\n- %s (%s, %d bytes)",
				att.Name, att.ContentType, att.Size))
		}
	}

	return content.String()
}

// buildSourceMeta - ОБНОВЛЕННАЯ ВЕРСИЯ С КОНФИГУРАЦИОННЫМИ ТЕГАМИ
func (p *MessageProcessor) buildSourceMeta(headers *domain.EmailHeaders, email domain.EmailMessage) map[string]interface{} {
	// Используем EmailHeaders value object для создания source_meta
	sourceMeta := headers.ToSourceMeta()

	// Добавляем информацию о вложениях
	if len(email.Attachments) > 0 {
		attachments := make([]map[string]interface{}, len(email.Attachments))
		for i, att := range email.Attachments {
			attachments[i] = map[string]interface{}{
				"name":         att.Name,
				"content_type": att.ContentType,
				"size":         att.Size,
			}
		}
		sourceMeta["attachments"] = attachments
	}

	// ✅ ДОБАВЛЯЕМ ИНФОРМАЦИЮ О КОНФИГУРАЦИИ ПОИСКА
	ctx := context.Background()
	searchConfig, err := p.searchService.GetThreadSearchConfig(ctx)
	if err == nil {
		sourceMeta["search_config"] = map[string]interface{}{
			"default_days_back":  searchConfig.DefaultDaysBack(),
			"extended_days_back": searchConfig.ExtendedDaysBack(),
			"max_days_back":      searchConfig.MaxDaysBack(),
			"config_version":     "phase3c_enhanced",
		}
	}

	// ✅ ЛОГИРУЕМ РЕЗУЛЬТАТ ОПТИМИЗАЦИИ С КОНФИГУРАЦИЕЙ
	p.logger.Debug(ctx, "Built OPTIMIZED source meta with CONFIGURABLE search",
		"message_id", headers.MessageID,
		"source_meta_keys", len(sourceMeta),
		"headers_optimized", true,
		"threading_data_preserved", headers.HasThreadingData(),
		"search_config_included", err == nil)

	return sourceMeta
}

// extractTags - ОБНОВЛЕННАЯ ВЕРСИЯ С КОНФИГУРАЦИОННЫМИ ТЕГАМИ
func (p *MessageProcessor) extractTags(ctx context.Context, email domain.EmailMessage) []string {
	tags := []string{
		"email",
		"auto-created",
		"headers-optimized",
		"phase3c-enhanced", // ✅ ДОБАВЛЯЕМ ТЕГ НОВОЙ ВЕРСИИ
	}

	// ✅ ДОБАВЛЯЕМ ТЕГ КОНФИГУРАЦИИ ПОИСКА
	searchConfig, err := p.searchService.GetThreadSearchConfig(ctx)
	if err == nil {
		tags = append(tags, fmt.Sprintf("search-%ddays", searchConfig.ExtendedDaysBack()))
	}

	// Добавляем теги на основе содержимого
	content := strings.ToLower(email.Subject + " " + email.BodyText)

	if strings.Contains(content, "срочно") || strings.Contains(content, "urgent") {
		tags = append(tags, "urgent")
	}

	if len(email.Attachments) > 0 {
		tags = append(tags, "has-attachments")
	}

	p.logger.Debug(ctx, "Generated tags for email",
		"message_id", email.MessageID,
		"tags_count", len(tags),
		"tags", tags)

	return tags
}

// NormalizeSubject - публичный метод для тестирования нормализации subject
func (p *MessageProcessor) NormalizeSubject(subject string) string {
	return p.normalizeSubject(subject)
}

// FindExistingTaskByThreadEnhanced - публичный метод для тестирования enhanced search
func (p *MessageProcessor) FindExistingTaskByThreadEnhanced(ctx context.Context, email domain.EmailMessage, headers *domain.EmailHeaders) *domain.Task {
	task, _ := p.findExistingTaskByThreadEnhanced(ctx, email, headers)
	return task
}
