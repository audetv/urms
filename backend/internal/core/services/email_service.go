package services

import (
	"context"
	"fmt"
	"time"

	"github.com/audetv/urms/internal/core/domain"
	"github.com/audetv/urms/internal/core/ports"
)

// EmailService реализует бизнес-логику работы с email
type EmailService struct {
	gateway     ports.EmailGateway
	repo        ports.EmailRepository
	processor   ports.MessageProcessor
	idGenerator domain.IDGenerator
	policy      domain.EmailProcessingPolicy
	logger      ports.Logger
}

// NewEmailService создает новый экземпляр EmailService
func NewEmailService(
	gateway ports.EmailGateway,
	repo ports.EmailRepository,
	processor ports.MessageProcessor,
	idGenerator domain.IDGenerator,
	policy domain.EmailProcessingPolicy,
	logger ports.Logger,
) *EmailService {
	return &EmailService{
		gateway:     gateway,
		repo:        repo,
		processor:   processor,
		idGenerator: idGenerator,
		policy:      policy,
		logger:      logger,
	}
}

// ProcessIncomingEmails обрабатывает входящие email сообщения
func (s *EmailService) ProcessIncomingEmails(ctx context.Context) error {
	s.logger.Info(ctx, "Starting incoming email processing")

	// Проверяем соединение
	if err := s.gateway.HealthCheck(ctx); err != nil {
		return fmt.Errorf("email gateway health check failed: %w", err)
	}

	// Критерии для выборки сообщений
	criteria := ports.FetchCriteria{
		Since:      s.getLastPollTime(),
		Mailbox:    "INBOX",
		Limit:      100,
		UnseenOnly: true,
	}

	// Получаем сообщения из email провайдера
	messages, err := s.gateway.FetchMessages(ctx, criteria)
	if err != nil {
		return fmt.Errorf("failed to fetch messages: %w", err)
	}

	s.logger.Info(ctx, "Fetched messages for processing", "count", len(messages))

	// Обрабатываем каждое сообщение
	processedCount := 0
	for _, msg := range messages {
		if err := s.processSingleEmail(ctx, msg); err != nil {
			s.logger.Error(ctx, "Failed to process email message",
				"message_id", msg.MessageID, "error", err)
			continue
		}
		processedCount++
	}

	s.logger.Info(ctx, "Completed email processing",
		"total", len(messages), "processed", processedCount)

	return nil
}

// SendEmail отправляет исходящее email сообщение
func (s *EmailService) SendEmail(ctx context.Context, msg domain.EmailMessage) error {
	s.logger.Info(ctx, "Sending email message",
		"to", msg.To, "subject", msg.Subject)

	// Валидируем сообщение
	if err := msg.Validate(); err != nil {
		return fmt.Errorf("email validation failed: %w", err)
	}

	// Применяем бизнес-правила
	if s.policy.ReadOnlyMode {
		s.logger.Warn(ctx, "Read-only mode enabled, skipping actual send")
		return nil
	}

	// Проверяем спам (для исходящих - проверяем получателей)
	if s.isSpamRecipient(msg) {
		s.logger.Warn(ctx, "Email to blocked recipient detected as spam",
			"message_id", msg.MessageID)
		return domain.NewEmailDomainError("email to blocked recipient", "SPAM_RECIPIENT", nil)
	}

	// Создаем исходящее сообщение с помощью IDGenerator
	outgoingMsg, err := domain.NewOutgoingEmail(
		msg.From,
		msg.To,
		msg.Subject,
		s.idGenerator,
	)
	if err != nil {
		return err
	}

	// Копируем остальные поля
	outgoingMsg.BodyHTML = msg.BodyHTML
	outgoingMsg.BodyText = msg.BodyText
	outgoingMsg.CC = msg.CC
	outgoingMsg.BCC = msg.BCC
	outgoingMsg.Attachments = msg.Attachments

	// Сохраняем в репозиторий перед отправкой
	if err := s.repo.Save(ctx, outgoingMsg); err != nil {
		return fmt.Errorf("failed to save outgoing email: %w", err)
	}

	// Отправляем через gateway
	if err := s.gateway.SendMessage(ctx, *outgoingMsg); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	// Обрабатываем исходящее сообщение
	if s.processor != nil {
		if err := s.processor.ProcessOutgoingEmail(ctx, *outgoingMsg); err != nil {
			s.logger.Error(ctx, "Failed to process outgoing email",
				"message_id", outgoingMsg.MessageID, "error", err)
			// Не прерываем выполнение, т.к. сообщение уже отправлено
		}
	}

	s.logger.Info(ctx, "Email sent successfully",
		"message_id", outgoingMsg.MessageID, "to", outgoingMsg.To)

	return nil
}

// TestConnection тестирует соединение с email сервером
func (s *EmailService) TestConnection(ctx context.Context) error {
	s.logger.Info(ctx, "Testing email connection")

	if err := s.gateway.Connect(ctx); err != nil {
		return fmt.Errorf("connection test failed: %w", err)
	}

	// Получаем информацию о почтовых ящиках
	mailboxes, err := s.gateway.ListMailboxes(ctx)
	if err != nil {
		return fmt.Errorf("failed to list mailboxes: %w", err)
	}

	s.logger.Info(ctx, "Connection test successful",
		"mailboxes_count", len(mailboxes))

	return nil
}

// GetEmailStatistics возвращает статистику по email сообщениям
func (s *EmailService) GetEmailStatistics(ctx context.Context) (*EmailStatistics, error) {
	// Получаем непрочитанные сообщения
	unprocessed, err := s.repo.FindUnprocessed(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get unprocessed emails: %w", err)
	}

	// Получаем сообщения за последние 24 часа
	last24h := time.Now().Add(-24 * time.Hour)
	recent, err := s.repo.FindByPeriod(ctx, last24h, time.Now())
	if err != nil {
		return nil, fmt.Errorf("failed to get recent emails: %w", err)
	}

	stats := &EmailStatistics{
		UnprocessedCount: len(unprocessed),
		RecentCount:      len(recent),
		LastProcessed:    s.getLastProcessedTime(unprocessed),
	}

	return stats, nil
}

// ProcessSingleEmail обрабатывает одно email сообщение (для тестирования)
func (s *EmailService) ProcessSingleEmail(ctx context.Context, msg domain.EmailMessage) error {
	return s.processSingleEmail(ctx, msg)
}

// Private methods

// processSingleEmail обрабатывает одно email сообщение
func (s *EmailService) processSingleEmail(ctx context.Context, msg domain.EmailMessage) error {
	s.logger.Debug(ctx, "Processing single email",
		"message_id", msg.MessageID, "subject", msg.Subject)

	// Проверяем, не было ли сообщение уже обработано
	existing, err := s.repo.FindByMessageID(ctx, msg.MessageID)
	if err == nil && existing != nil && existing.Processed {
		s.logger.Debug(ctx, "Email already processed, skipping",
			"message_id", msg.MessageID)
		return nil
	}

	// Проверяем спам-фильтр
	if msg.IsSpam(s.policy) {
		s.logger.Info(ctx, "Skipping spam email",
			"message_id", msg.MessageID, "subject", msg.Subject)
		msg.Processed = true
		msg.ProcessedAt = time.Now()
		if err := s.repo.Save(ctx, &msg); err != nil {
			s.logger.Error(ctx, "Failed to save spam email",
				"message_id", msg.MessageID, "error", err)
		}
		return nil
	}

	// Проверяем разрешенных отправителей
	if !msg.IsFromAllowedSender(s.policy) {
		s.logger.Warn(ctx, "Email from blocked sender",
			"message_id", msg.MessageID, "from", msg.From)
		msg.Processed = true
		msg.ProcessedAt = time.Now()
		if err := s.repo.Save(ctx, &msg); err != nil {
			s.logger.Error(ctx, "Failed to save blocked sender email",
				"message_id", msg.MessageID, "error", err)
		}
		return nil
	}

	// Сохраняем сообщение
	msg.Direction = domain.DirectionIncoming
	msg.Processed = false
	msg.CreatedAt = time.Now()
	msg.UpdatedAt = time.Now()

	if err := s.repo.Save(ctx, &msg); err != nil {
		return fmt.Errorf("failed to save incoming email: %w", err)
	}

	// Обрабатываем через процессор
	if s.processor != nil {
		if err := s.processor.ProcessIncomingEmail(ctx, msg); err != nil {
			return fmt.Errorf("failed to process incoming email: %w", err)
		}
	}

	// Помечаем как обработанное
	msg.Processed = true
	msg.ProcessedAt = time.Now()
	if err := s.repo.Update(ctx, &msg); err != nil {
		s.logger.Error(ctx, "Failed to mark email as processed",
			"message_id", msg.MessageID, "error", err)
	}

	// Помечаем как прочитанное на сервере (если не в read-only режиме)
	if !s.policy.ReadOnlyMode {
		if err := s.gateway.MarkAsRead(ctx, []string{msg.MessageID}); err != nil {
			s.logger.Warn(ctx, "Failed to mark email as read on server",
				"message_id", msg.MessageID, "error", err)
		}
	}

	s.logger.Info(ctx, "Email processed successfully",
		"message_id", msg.MessageID, "subject", msg.Subject)

	return nil
}

// isSpamRecipient проверяет получателей на спам (для исходящих)
func (s *EmailService) isSpamRecipient(msg domain.EmailMessage) bool {
	// Проверяем всех получателей на наличие в заблокированных
	for _, recipient := range msg.To {
		for _, blocked := range s.policy.BlockedSenders {
			if recipient == blocked {
				return true
			}
		}
	}
	return false
}

// getLastPollTime возвращает время последнего успешного опроса
func (s *EmailService) getLastPollTime() time.Time {
	// TODO: Реализовать сохранение времени последнего опроса
	// Пока возвращаем время 1 час назад для первого запуска
	return time.Now().Add(-1 * time.Hour)
}

// getLastProcessedTime возвращает время последнего обработанного сообщения
func (s *EmailService) getLastProcessedTime(messages []domain.EmailMessage) *time.Time {
	if len(messages) == 0 {
		return nil
	}

	var lastTime time.Time
	for _, msg := range messages {
		if msg.ProcessedAt.After(lastTime) {
			lastTime = msg.ProcessedAt
		}
	}

	return &lastTime
}

// EmailStatistics статистика по email сообщениям
type EmailStatistics struct {
	UnprocessedCount int        `json:"unprocessed_count"`
	RecentCount      int        `json:"recent_count"`
	LastProcessed    *time.Time `json:"last_processed"`
}
