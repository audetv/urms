package email

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/audetv/urms/internal/core/domain"
	"github.com/audetv/urms/internal/core/ports"
	"github.com/audetv/urms/internal/core/services"
	imapclient "github.com/audetv/urms/internal/infrastructure/email/imap"
	"github.com/audetv/urms/internal/infrastructure/logging"
	"github.com/emersion/go-imap"
)

// MessageBodyInfo содержит распарсенную информацию о теле сообщения
type MessageBodyInfo struct {
	Text        string
	HTML        string
	Attachments []domain.Attachment
}

// IMAPAdapter реализует ports.EmailGateway используя существующий IMAP клиент
type IMAPAdapter struct {
	client            *imapclient.Client
	config            *imapclient.Config
	searchConfig      ports.EmailSearchConfigProvider // ✅ Уже добавлено ранее
	searchService     *services.EmailSearchService    // ✅ ДОБАВЛЯЕМ сервис
	mimeParser        *MIMEParser
	addressNormalizer *AddressNormalizer
	retryManager      *RetryManager
	timeoutConfig     TimeoutConfig
	logger            ports.Logger
}

// TimeoutConfig конфигурация таймаутов для IMAP операций
type TimeoutConfig struct {
	ConnectTimeout   time.Duration
	LoginTimeout     time.Duration
	FetchTimeout     time.Duration
	OperationTimeout time.Duration
	PageSize         int
	MaxMessages      int
	MaxRetries       int
	RetryDelay       time.Duration
}

// NewIMAPAdapter создает новый IMAP адаптер с поддержкой таймаутов
func NewIMAPAdapter(
	config *imapclient.Config,
	timeoutConfig TimeoutConfig,
	searchConfig ports.EmailSearchConfigProvider, // ✅ Конфигурационный порт
	logger ports.Logger,
) *IMAPAdapter {
	retryConfig := RetryConfig{
		MaxAttempts:   timeoutConfig.MaxRetries,
		BaseDelay:     timeoutConfig.RetryDelay,
		MaxDelay:      30 * time.Second,
		BackoffFactor: 1.5,
	}

	// ✅ СОЗДАЕМ сервис поиска
	searchService := services.NewEmailSearchService(searchConfig, logger)

	return &IMAPAdapter{
		client:            imapclient.NewClient(config),
		config:            config,
		searchConfig:      searchConfig,
		searchService:     searchService, // ✅ ИНИЦИАЛИЗИРУЕМ сервис
		mimeParser:        NewMIMEParser(logger),
		addressNormalizer: NewAddressNormalizer(),
		retryManager:      NewRetryManager(retryConfig, logger),
		timeoutConfig:     timeoutConfig,
		logger:            logger,
	}
}

// NewIMAPAdapterWithTimeoutsAndConfig создает IMAP адаптер с поддержкой таймаутов и конфигурации поиска
func NewIMAPAdapterWithTimeoutsAndConfig(
	config *imapclient.Config,
	timeoutConfig TimeoutConfig,
	searchConfig ports.EmailSearchConfigProvider, // ✅ ДОБАВЛЯЕМ конфигурацию
	logger ports.Logger,
) *IMAPAdapter {
	retryConfig := RetryConfig{
		MaxAttempts:   timeoutConfig.MaxRetries,
		BaseDelay:     timeoutConfig.RetryDelay,
		MaxDelay:      30 * time.Second,
		BackoffFactor: 1.5,
	}

	// ✅ СОЗДАЕМ сервис поиска
	searchService := services.NewEmailSearchService(searchConfig, logger)

	return &IMAPAdapter{
		client:            imapclient.NewClient(config),
		config:            config,
		searchConfig:      searchConfig,  // ✅ СОХРАНЯЕМ конфигурацию
		searchService:     searchService, // ✅ СОХРАНЯЕМ сервис
		mimeParser:        NewMIMEParser(logger),
		addressNormalizer: NewAddressNormalizer(),
		retryManager:      NewRetryManager(retryConfig, logger),
		timeoutConfig:     timeoutConfig,
		logger:            logger,
	}
}

// NewIMAPAdapterWithTimeouts создает новый IMAP адаптер с поддержкой расширенной конфигурации таймаутов
func NewIMAPAdapterWithTimeouts(config *imapclient.Config, timeoutConfig TimeoutConfig, logger ports.Logger) *IMAPAdapter {
	// Настраиваем retry manager с конфигурацией из timeoutConfig
	retryConfig := RetryConfig{
		MaxAttempts:   timeoutConfig.MaxRetries,
		BaseDelay:     timeoutConfig.RetryDelay,
		MaxDelay:      30 * time.Second,
		BackoffFactor: 1.5,
	}

	return &IMAPAdapter{
		client:            imapclient.NewClient(config),
		config:            config,
		mimeParser:        NewMIMEParser(logger),
		addressNormalizer: NewAddressNormalizer(),
		retryManager:      NewRetryManager(retryConfig, logger),
		timeoutConfig:     timeoutConfig,
		logger:            logger, // ✅ ДОБАВЛЯЕМ logger
	}
}

// ✅ NEW: Метод для обратной совместимости (без timeoutConfig)
func NewIMAPAdapterLegacy(config *imapclient.Config) *IMAPAdapter {
	// Используем дефолтные значения таймаутов
	defaultTimeoutConfig := TimeoutConfig{
		ConnectTimeout:   30 * time.Second,
		LoginTimeout:     15 * time.Second,
		FetchTimeout:     60 * time.Second,
		OperationTimeout: 120 * time.Second,
		PageSize:         100,
		MaxMessages:      500,
		MaxRetries:       3,
		RetryDelay:       10 * time.Second,
	}

	// ✅ СОЗДАЕМ тестовый logger для обратной совместимости
	testLogger := logging.NewTestLogger()

	return NewIMAPAdapter(config, defaultTimeoutConfig, nil, testLogger)
}

// Connect устанавливает соединение с IMAP сервером с таймаутом
func (a *IMAPAdapter) Connect(ctx context.Context) error {
	operation := "IMAP connect"

	// Создаем контекст с таймаутом подключения
	ctx, cancel := context.WithTimeout(ctx, a.timeoutConfig.ConnectTimeout)
	defer cancel()

	a.logger.Info(ctx, "Starting IMAP connection",
		"operation", operation,
		"server", a.config.Server,
		"port", a.config.Port,
		"timeout", a.timeoutConfig.ConnectTimeout.String())

	return a.retryManager.ExecuteWithRetry(ctx, operation, func() error {
		err := a.client.Connect()
		if err != nil {
			a.logger.Error(ctx, "IMAP connection failed",
				"operation", operation,
				"server", a.config.Server,
				"error", err.Error())
			return NewIMAPError("connect", IMAPErrorConnection, "failed to connect to IMAP server", err)
		}

		a.logger.Info(ctx, "IMAP connection successful",
			"operation", operation,
			"server", a.config.Server)
		return nil
	})
}

// Disconnect закрывает соединение
func (a *IMAPAdapter) Disconnect() error {
	return a.client.Logout()
}

// HealthCheck с таймаутом проверяет состояние соединения
func (a *IMAPAdapter) HealthCheck(ctx context.Context) error {
	operation := "IMAP health check"

	// Создаем контекст с operation timeout
	ctx, cancel := context.WithTimeout(ctx, a.timeoutConfig.OperationTimeout)
	defer cancel()

	return a.retryManager.ExecuteWithRetry(ctx, operation, func() error {
		err := a.client.CheckConnection()
		if err != nil {
			return NewIMAPError("health_check", IMAPErrorConnection, "health check failed", err)
		}
		return nil
	})
}

// FetchMessages с таймаутом и пагинацией получает сообщения по критериям
func (a *IMAPAdapter) FetchMessages(ctx context.Context, criteria ports.FetchCriteria) ([]domain.EmailMessage, error) {
	operation := "IMAP fetch messages"

	// Создаем контекст с таймаутом получения
	ctx, cancel := context.WithTimeout(ctx, a.timeoutConfig.FetchTimeout)
	defer cancel()

	var messages []domain.EmailMessage

	err := a.retryManager.ExecuteWithRetry(ctx, operation, func() error {
		fetchedMessages, err := a.fetchMessagesWithPagination(ctx, criteria)
		if err != nil {
			return err
		}
		messages = fetchedMessages
		return nil
	})

	return messages, err
}

// fetchMessagesWithPagination - внутренний метод с поддержкой пагинации
func (a *IMAPAdapter) fetchMessagesWithPagination(ctx context.Context, criteria ports.FetchCriteria) ([]domain.EmailMessage, error) {
	a.logger.Info(ctx, "Starting IMAP pagination",
		"operation", "fetch_pagination",
		"mailbox", criteria.Mailbox,
		"since_uid", criteria.SinceUID,
		"since", criteria.Since,
		"page_size", a.timeoutConfig.PageSize)

	// ВЫБИРАЕМ почтовый ящик перед поиском
	if err := a.SelectMailbox(ctx, criteria.Mailbox); err != nil {
		a.logger.Error(ctx, "Failed to select mailbox",
			"operation", "select_mailbox",
			"mailbox", criteria.Mailbox,
			"error", err.Error())
		return nil, NewIMAPError("select_mailbox", IMAPErrorServer, fmt.Sprintf("failed to select mailbox %s", criteria.Mailbox), err)
	}

	// ✅ ИСПРАВЛЕНО: Для первого запуска используем альтернативный подход
	if criteria.SinceUID == 0 {
		return a.fetchInitialMessages(ctx, criteria)
	}

	// Конвертируем доменные критерии в IMAP-специфичные
	imapCriteria, err := a.convertToIMAPCriteria(ctx, criteria)
	if err != nil {
		return nil, fmt.Errorf("failed to convert criteria: %w", err)
	}

	// Ищем сообщения по UID с поддержкой пагинации
	allMessages := []domain.EmailMessage{}
	lastUID := criteria.SinceUID
	processedCount := 0
	hasMoreMessages := true
	pageNumber := 1

	for hasMoreMessages {
		select {
		case <-ctx.Done():
			a.logger.Warn(ctx, "IMAP pagination cancelled by context",
				"operation", "fetch_pagination",
				"processed", processedCount,
				"total_pages", pageNumber-1)
			return nil, ctx.Err()
		default:
			// Проверяем лимит сообщений
			if processedCount >= a.timeoutConfig.MaxMessages {
				a.logger.Info(ctx, "Reached maximum messages per poll, stopping pagination",
					"operation", "fetch_pagination",
					"processed", processedCount,
					"max_messages", a.timeoutConfig.MaxMessages)
				hasMoreMessages = false
				break
			}

			// Обновляем критерии для следующей страницы
			imapCriteria.Uid = a.createUIDSeqSet(ctx, lastUID, a.timeoutConfig.PageSize)

			a.logger.Debug(ctx, "Searching for messages with UID criteria",
				"operation", "fetch_pagination",
				"page", pageNumber,
				"last_uid", lastUID)

			messageUIDs, err := a.client.SearchMessages(imapCriteria)
			if err != nil {
				a.logger.Error(ctx, "Failed to search messages",
					"operation", "fetch_messages",
					"page", pageNumber,
					"error", err.Error())
				return nil, NewIMAPError("search_messages", IMAPErrorProtocol, "failed to search messages", err)
			}

			a.logger.Debug(ctx, "Search results", "operation", "fetch_pagination", "page", pageNumber, "found_uids", len(messageUIDs))

			if len(messageUIDs) == 0 {
				// Больше нет сообщений
				a.logger.Info(ctx, "No more messages found, ending pagination",
					"operation", "fetch_pagination",
					"total_pages", pageNumber,
					"total_messages", len(allMessages))
				hasMoreMessages = false
				break
			}

			// Получаем сообщения текущей страницы
			batchMessages, err := a.fetchMessageBatch(ctx, messageUIDs)
			if err != nil {
				a.logger.Error(ctx, "Failed to fetch message batch",
					"operation", "fetch_pagination",
					"page", pageNumber,
					"error", err.Error())
				return nil, err
			}

			// Добавляем к общему результату
			allMessages = append(allMessages, batchMessages...)
			processedCount += len(batchMessages)

			// Обновляем lastUID для следующей итерации
			if len(batchMessages) > 0 {
				lastUID = a.extractMaxUID(ctx, batchMessages)
			}

			// Логируем прогресс
			a.logger.Info(ctx, "IMAP pagination progress",
				"operation", "fetch_pagination",
				"page", pageNumber,
				"batch_size", len(batchMessages),
				"total_processed", processedCount,
				"last_uid", lastUID)

			// Если получили меньше сообщений, чем размер страницы, значит это последняя страница
			if len(batchMessages) < a.timeoutConfig.PageSize {
				hasMoreMessages = false
				a.logger.Info(ctx, "Reached last page of messages",
					"operation", "fetch_pagination",
					"page", pageNumber,
					"total_messages", len(allMessages))
			}

			pageNumber++
		}
	}

	a.logger.Info(ctx, "IMAP pagination completed",
		"operation", "fetch_pagination",
		"total_messages", len(allMessages),
		"total_pages", pageNumber-1)

	return allMessages, nil
}

// fetchMessageBatch получает пачку сообщений по UID с таймаутом
func (a *IMAPAdapter) fetchMessageBatch(ctx context.Context, messageUIDs []uint32) ([]domain.EmailMessage, error) {
	a.logger.Debug(ctx, "Fetching message batch WITH BODY",
		"operation", "fetch_batch_with_body",
		"message_count", len(messageUIDs))

	// Создаем контекст с таймаутом для batch операций
	batchCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Создаем SeqSet для получения сообщений
	seqSet := new(imap.SeqSet)
	for _, uid := range messageUIDs {
		seqSet.AddNum(uid)
	}

	// ✅ ИСПРАВЛЕНО: Запрашиваем сообщения С ТЕЛОМ
	fetchItems := imapclient.CreateFetchItems(true) // true = ЗАПРАШИВАЕМ ТЕЛО ПИСЬМА
	messagesChan, err := a.client.FetchMessages(seqSet, fetchItems)
	if err != nil {
		a.logger.Error(ctx, "Failed to fetch messages with body",
			"operation", "fetch_batch",
			"error", err.Error())
		return nil, NewIMAPError("fetch_messages", IMAPErrorProtocol, "failed to fetch messages with body", err)
	}

	// Конвертируем IMAP сообщения в доменные сущности
	var domainMessages []domain.EmailMessage
	for msg := range messagesChan {
		select {
		case <-batchCtx.Done():
			a.logger.Warn(ctx, "Batch processing interrupted by timeout",
				"operation", "fetch_batch",
				"processed", len(domainMessages),
				"error", batchCtx.Err().Error())
			return domainMessages, batchCtx.Err()
		default:
			// ✅ ИСПОЛЬЗУЕМ convertToDomainMessageWithBody для полного парсинга
			domainMsg, err := a.convertToDomainMessageWithBody(msg)
			if err != nil {
				a.logger.Warn(ctx, "Failed to convert IMAP message with body",
					"operation", "fetch_batch",
					"uid", msg.Uid,
					"error", err.Error())
				continue
			}
			domainMessages = append(domainMessages, domainMsg)
		}
	}

	a.logger.Debug(ctx, "Message batch with body conversion completed",
		"operation", "fetch_batch",
		"converted_messages", len(domainMessages),
		"first_message_body_length", len(domainMessages[0].BodyText))

	return domainMessages, nil
}

// fetchInitialMessages получает сообщения для первого запуска (без UID)
func (a *IMAPAdapter) fetchInitialMessages(ctx context.Context, criteria ports.FetchCriteria) ([]domain.EmailMessage, error) {

	a.logger.Info(ctx, "Performing initial message fetch without UID tracking", "operation", "fetch_initial")

	// Используем поиск по дате и статусу
	imapCriteria := &imap.SearchCriteria{}

	// Непрочитанные сообщения
	if criteria.UnseenOnly {
		imapCriteria.WithoutFlags = []string{imap.SeenFlag}
	}

	// Ограничение по дате - только за последние 7 дней
	imapCriteria.Since = time.Now().Add(-3 * 24 * time.Hour)

	messageUIDs, err := a.client.SearchMessages(imapCriteria)
	if err != nil {
		a.logger.Error(ctx, "Failed to search initial messages", "operation", "fetch_initial", "error", err.Error())
		return nil, NewIMAPError("search_messages", IMAPErrorProtocol, "failed to search initial messages", err)
	}

	a.logger.Info(ctx, "Initial message fetch completed",
		"operation", "fetch_initial",
		"message_count", len(messageUIDs),
		"since", imapCriteria.Since)

	if len(messageUIDs) == 0 {
		return []domain.EmailMessage{}, nil
	}

	// ✅ УВЕЛИЧИВАЕМ ОГРАНИЧЕНИЕ: Берем только первые 50 сообщений для тестирования
	if len(messageUIDs) > 100 {
		messageUIDs = messageUIDs[:100]
		a.logger.Info(ctx, "Limited initial message fetch for testing",
			"operation", "fetch_initial",
			"limited_to", 100)
	}

	// Получаем сообщения
	return a.fetchMessageBatch(ctx, messageUIDs)
}

// createUIDSeqSet создает SeqSet для пагинации по UID
func (a *IMAPAdapter) createUIDSeqSet(ctx context.Context, sinceUID uint32, limit int) *imap.SeqSet {
	seqSet := new(imap.SeqSet)

	if sinceUID > 0 {
		// Начинаем со следующего UID после sinceUID
		startUID := sinceUID + 1
		// Ограничиваем количество сообщений
		endUID := startUID + uint32(limit) - 1
		seqSet.AddRange(startUID, endUID)

		a.logger.Debug(ctx, "Creating UID range for pagination",
			"start_uid", startUID,
			"end_uid", endUID,
			"limit", limit)
	} else {
		// ✅ ИСПРАВЛЕНО: Для первого запроса используем ALL вместо конкретного диапазона
		// UID будет установлен в convertToIMAPCriteria через дату
		a.logger.Debug(ctx, "Using date-based criteria for initial search, no UID range")
		// Не устанавливаем UID - используем критерии по дате
	}

	return seqSet
}

// extractMaxUID извлекает максимальный UID из пачки сообщений
func (a *IMAPAdapter) extractMaxUID(ctx context.Context, messages []domain.EmailMessage) uint32 {
	if len(messages) == 0 {
		return 0
	}

	maxUID := uint32(0)
	for _, msg := range messages {
		// ✅ ИСПРАВЛЕНО: Извлекаем реальный UID из headers
		if uidHeaders, exists := msg.Headers["X-IMAP-UID"]; exists && len(uidHeaders) > 0 {
			if uid, err := strconv.ParseUint(uidHeaders[0], 10, 32); err == nil {
				if uint32(uid) > maxUID {
					maxUID = uint32(uid)
				}
			}
		}
	}

	if maxUID == 0 {
		// Fallback: используем временную логику если UID не найден
		a.logger.Warn(ctx, "No IMAP UID found in messages, using fallback logic")
		for _, msg := range messages {
			uidHash := uint32(0)
			for _, char := range msg.MessageID {
				uidHash = uidHash*31 + uint32(char)
			}
			if uidHash > maxUID {
				maxUID = uidHash
			}
		}
	}

	return maxUID
}

// FetchMessagesWithBody получает сообщения с полным телом и вложениями с таймаутом
func (a *IMAPAdapter) FetchMessagesWithBody(ctx context.Context, criteria ports.FetchCriteria) ([]domain.EmailMessage, error) {
	// Создаем контекст с таймаутом
	ctx, cancel := context.WithTimeout(ctx, a.timeoutConfig.FetchTimeout)
	defer cancel()

	if err := a.SelectMailbox(ctx, criteria.Mailbox); err != nil {
		return nil, fmt.Errorf("failed to select mailbox %s: %w", criteria.Mailbox, err)
	}

	imapCriteria, err := a.convertToIMAPCriteria(ctx, criteria)
	if err != nil {
		return nil, fmt.Errorf("failed to convert criteria: %w", err)
	}

	messageUIDs, err := a.client.SearchMessages(imapCriteria)
	if err != nil {
		return nil, fmt.Errorf("failed to search messages: %w", err)
	}

	if len(messageUIDs) == 0 {
		return []domain.EmailMessage{}, nil
	}

	seqSet := new(imap.SeqSet)
	for _, uid := range messageUIDs {
		seqSet.AddNum(uid)
	}

	// Получаем полные сообщения с телом и вложениями
	fetchItems := imapclient.CreateFetchItems(true) // С телом сообщения
	messagesChan, err := a.client.FetchMessages(seqSet, fetchItems)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch messages: %w", err)
	}

	var domainMessages []domain.EmailMessage
	for msg := range messagesChan {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			domainMsg, err := a.convertToDomainMessageWithBody(msg)
			if err != nil {
				a.logger.Warn(ctx, "Failed to convert IMAP message with body", "uid", msg.Uid, "error", err.Error())
				continue
			}
			domainMessages = append(domainMessages, domainMsg)
		}
	}

	return domainMessages, nil
}

// SendMessage отправляет сообщение (заглушка - будет реализовано в SMTP адаптере)
func (a *IMAPAdapter) SendMessage(ctx context.Context, msg domain.EmailMessage) error {
	return fmt.Errorf("IMAP adapter does not support sending messages. Use SMTP adapter instead")
}

// MarkAsRead помечает сообщения как прочитанные с таймаутом
func (a *IMAPAdapter) MarkAsRead(ctx context.Context, messageIDs []string) error {
	// Создаем контекст с таймаутом
	ctx, cancel := context.WithTimeout(ctx, a.timeoutConfig.OperationTimeout)
	defer cancel()

	operation := "IMAP mark as read"

	return a.retryManager.ExecuteWithRetry(ctx, operation, func() error {
		// TODO: Реализовать пометку сообщений как прочитанных
		a.logger.Info(ctx, "Marking messages as read", "message_ids", messageIDs)
		return nil
	})
}

// MarkAsProcessed помечает сообщения как обработанные с таймаутом
func (a *IMAPAdapter) MarkAsProcessed(ctx context.Context, messageIDs []string) error {
	// Создаем контекст с таймаутом
	ctx, cancel := context.WithTimeout(ctx, a.timeoutConfig.OperationTimeout)
	defer cancel()

	operation := "IMAP mark as processed"

	return a.retryManager.ExecuteWithRetry(ctx, operation, func() error {
		// IMAP не поддерживает эту операцию напрямую
		// Можно реализовать через перемещение в другую папку
		a.logger.Info(ctx, "Marking messages as processed", "message_ids", messageIDs)
		return nil
	})
}

// ListMailboxes с таймаутом возвращает список почтовых ящиков
func (a *IMAPAdapter) ListMailboxes(ctx context.Context) ([]ports.MailboxInfo, error) {
	operation := "IMAP list mailboxes"

	// Создаем контекст с таймаутом
	ctx, cancel := context.WithTimeout(ctx, a.timeoutConfig.OperationTimeout)
	defer cancel()

	var mailboxes []ports.MailboxInfo

	err := a.retryManager.ExecuteWithRetry(ctx, operation, func() error {
		// TODO: Реализовать получение списка почтовых ящиков
		// Пока возвращаем только INBOX
		mailboxes = []ports.MailboxInfo{
			{
				Name:     "INBOX",
				Messages: 0,
				Unseen:   0,
				Recent:   0,
			},
		}
		return nil
	})

	return mailboxes, err
}

// SelectMailbox с таймаутом выбирает почтовый ящик
func (a *IMAPAdapter) SelectMailbox(ctx context.Context, name string) error {
	operation := fmt.Sprintf("IMAP select mailbox %s", name)

	// Создаем контекст с таймаутом
	ctx, cancel := context.WithTimeout(ctx, a.timeoutConfig.OperationTimeout)
	defer cancel()

	return a.retryManager.ExecuteWithRetry(ctx, operation, func() error {
		_, err := a.client.SelectMailbox(name, a.config.ReadOnly)
		if err != nil {
			return NewIMAPError("select_mailbox", IMAPErrorServer, fmt.Sprintf("failed to select mailbox %s", name), err)
		}
		return nil
	})
}

// GetMailboxInfo возвращает информацию о почтовом ящике с таймаутом
func (a *IMAPAdapter) GetMailboxInfo(ctx context.Context, name string) (*ports.MailboxInfo, error) {
	// Создаем контекст с таймаутом
	ctx, cancel := context.WithTimeout(ctx, a.timeoutConfig.OperationTimeout)
	defer cancel()

	operation := "IMAP get mailbox info"

	var mailboxInfo *ports.MailboxInfo

	err := a.retryManager.ExecuteWithRetry(ctx, operation, func() error {
		mailbox, err := a.client.GetMailboxInfo(name)
		if err != nil {
			return fmt.Errorf("failed to get mailbox info: %w", err)
		}

		mailboxInfo = &ports.MailboxInfo{
			Name:     mailbox.Name,
			Messages: int(mailbox.Messages),
			Unseen:   int(mailbox.Unseen),
			Recent:   int(mailbox.Recent),
		}
		return nil
	})

	return mailboxInfo, err
}

// convertToIMAPCriteria конвертирует доменные критерии в IMAP-специфичные
// convertToIMAPCriteria - ОБНОВЛЯЕМ для возврата error
func (a *IMAPAdapter) convertToIMAPCriteria(ctx context.Context, criteria ports.FetchCriteria) (*imap.SearchCriteria, error) {
	imapCriteria := &imap.SearchCriteria{}

	// ✅ ПОЛУЧАЕМ КОНФИГУРАЦИЮ ЧЕРЕЗ СЕРВИС
	searchConfig, err := a.searchService.GetThreadSearchConfig(ctx)
	if err != nil {
		a.logger.Error(ctx, "Failed to get search configuration for criteria",
			"error", err.Error())
		return nil, fmt.Errorf("failed to get search configuration: %w", err)
	}

	// ✅ ИСПОЛЬЗУЕМ КОНФИГУРИРУЕМЫЕ ЗНАЧЕНИЯ
	if !criteria.Since.IsZero() {
		imapCriteria.Since = criteria.Since
	} else {
		imapCriteria.Since = searchConfig.GetSearchSince("standard")
	}

	if criteria.Subject != "" {
		if imapCriteria.Header == nil {
			imapCriteria.Header = make(map[string][]string)
		}
		imapCriteria.Header["Subject"] = []string{criteria.Subject}

		a.logger.Debug(ctx, "Added subject-based search to criteria",
			"subject", criteria.Subject,
			"since", imapCriteria.Since.Format("2006-01-02"))
	}

	if criteria.SinceUID == 0 && criteria.UnseenOnly {
		imapCriteria.WithoutFlags = []string{imap.SeenFlag}
	}

	a.logger.Info(ctx, "Using CONFIGURABLE search criteria",
		"since", imapCriteria.Since.Format("2006-01-02"),
		"days_back", searchConfig.DefaultDaysBack(),
		"has_subject", criteria.Subject != "",
		"unseen_only", criteria.UnseenOnly,
		"config_source", "EmailSearchConfig")

	return imapCriteria, nil // ✅ ВОЗВРАЩАЕМ error
}

// SearchThreadMessages ищет все сообщения в цепочке по threading данным
// SearchThreadMessages - ОБНОВЛЯЕМ для использования createEnhancedThreadSearchCriteria
func (a *IMAPAdapter) SearchThreadMessages(ctx context.Context, threadData ports.ThreadSearchCriteria) ([]domain.EmailMessage, error) {
	operation := "IMAP enhanced thread search"

	providerConfig, err := a.searchService.GetProviderSearchConfig(ctx, "imap")
	if err != nil {
		a.logger.Warn(ctx, "Failed to get provider config, using default timeout",
			"provider", "imap", "error", err.Error())
		providerConfig = &ports.ProviderSearchConfig{
			SearchTimeout: a.timeoutConfig.FetchTimeout,
		}
	}

	ctx, cancel := context.WithTimeout(ctx, providerConfig.SearchTimeout*2)
	defer cancel()

	a.logger.Info(ctx, "🚀 Starting ENHANCED thread-aware message search",
		"operation", operation,
		"message_id", threadData.MessageID,
		"in_reply_to", threadData.InReplyTo,
		"references_count", len(threadData.References),
		"subject", threadData.Subject,
		"mailbox", threadData.Mailbox,
		"timeout", providerConfig.SearchTimeout*2)

	var messages []domain.EmailMessage

	err = a.retryManager.ExecuteWithRetry(ctx, operation, func() error {
		if err := a.SelectMailbox(ctx, threadData.Mailbox); err != nil {
			return fmt.Errorf("failed to select mailbox: %w", err)
		}

		// ✅ ИСПОЛЬЗУЕМ НОВЫЙ МЕТОД createEnhancedThreadSearchCriteria
		imapCriteria, err := a.createEnhancedThreadSearchCriteria(threadData)
		if err != nil {
			return fmt.Errorf("failed to create enhanced search criteria: %w", err)
		}

		messageUIDs, err := a.client.SearchMessages(imapCriteria)
		if err != nil {
			return fmt.Errorf("failed to search thread messages: %w", err)
		}

		a.logger.Info(ctx, "Enhanced thread search completed",
			"message_id", threadData.MessageID,
			"found_messages", len(messageUIDs),
			"search_criteria", a.describeEnhancedSearchCriteria(imapCriteria))

		if len(messageUIDs) == 0 {
			messages = []domain.EmailMessage{}

			a.logger.Warn(ctx, "No messages found with enhanced search criteria",
				"original_message_id", threadData.MessageID,
				"criteria_used", a.describeEnhancedSearchCriteria(imapCriteria))
			return nil
		}

		fetchedMessages, err := a.fetchMessageBatch(ctx, messageUIDs)
		if err != nil {
			return fmt.Errorf("failed to fetch thread messages: %w", err)
		}

		messages = fetchedMessages

		a.logger.Info(ctx, "✅ ENHANCED thread search SUCCESS",
			"original_message_id", threadData.MessageID,
			"found_thread_messages", len(messages),
			"first_found_message_id", safeGetMessageID(messages),
			"search_strategy", "extended_time+combined_criteria+configurable")

		return nil
	})

	return messages, err
}

// Вспомогательная функция для безопасного получения Message-ID
func safeGetMessageID(messages []domain.EmailMessage) string {
	if len(messages) == 0 {
		return "none"
	}
	return messages[0].MessageID
}

// describeEnhancedSearchCriteria - улучшенное описание критериев
func (a *IMAPAdapter) describeEnhancedSearchCriteria(criteria *imap.SearchCriteria) string {
	description := []string{}

	if criteria.Since != (time.Time{}) {
		days := int(time.Since(criteria.Since).Hours() / 24)
		description = append(description, fmt.Sprintf("since:%s(%d days)",
			criteria.Since.Format("2006-01-02"), days))
	}

	if criteria.Header != nil {
		for key, values := range criteria.Header {
			if key == "Subject" && len(values) > 3 {
				description = append(description,
					fmt.Sprintf("subject:%d variants", len(values)))
			} else {
				description = append(description,
					fmt.Sprintf("%s:%d values", key, len(values)))
			}
		}
	}

	if len(description) == 0 {
		return "default_criteria"
	}

	return strings.Join(description, " | ")
}

// internal/infrastructure/email/imap_adapter.go
// createEnhancedThreadSearchCriteria - НОВЫЙ МЕТОД с конфигурацией
func (a *IMAPAdapter) createEnhancedThreadSearchCriteria(threadData ports.ThreadSearchCriteria) (*imap.SearchCriteria, error) {
	ctx := context.Background()
	criteria := &imap.SearchCriteria{}

	// ✅ СТРАТЕГИЯ 1: КОМБИНИРОВАННЫЕ MESSAGE-ID КРИТЕРИИ
	var allMessageIDs []string

	if threadData.MessageID != "" {
		allMessageIDs = append(allMessageIDs, threadData.MessageID)
	}
	if threadData.InReplyTo != "" {
		allMessageIDs = append(allMessageIDs, threadData.InReplyTo)
	}
	if len(threadData.References) > 0 {
		allMessageIDs = append(allMessageIDs, threadData.References...)
	}

	allMessageIDs = a.removeDuplicateMessageIDs(allMessageIDs)

	if len(allMessageIDs) > 0 {
		criteria.Header = map[string][]string{
			"Message-ID":  allMessageIDs,
			"In-Reply-To": allMessageIDs,
		}

		if len(threadData.References) > 0 {
			criteria.Header["References"] = threadData.References
		}
	}

	// ✅ СТРАТЕГИЯ 2: SUBJECT-BASED ПОИСК С ПРЕФИКСАМИ ИЗ КОНФИГУРАЦИИ
	if threadData.Subject != "" {
		subjectVariants, err := a.searchService.GenerateSearchSubjectVariants(ctx, threadData.Subject)
		if err != nil {
			a.logger.Warn(ctx, "Failed to generate subject variants, using basic subject",
				"subject", threadData.Subject, "error", err.Error())
			subjectVariants = []string{threadData.Subject}
		}

		if criteria.Header == nil {
			criteria.Header = make(map[string][]string)
		}

		criteria.Header["Subject"] = subjectVariants
	}

	// ✅ СТРАТЕГИЯ 3: РАСШИРЕННЫЙ ВРЕМЕННОЙ ДИАПАЗОН ИЗ КОНФИГУРАЦИИ
	searchConfig, err := a.searchService.GetThreadSearchConfig(ctx)
	if err != nil {
		a.logger.Warn(ctx, "Failed to get search config, using default 90 days",
			"error", err.Error())
		criteria.Since = time.Now().Add(-90 * 24 * time.Hour)
	} else {
		criteria.Since = searchConfig.GetSearchSince("extended")
	}

	a.logger.Debug(ctx, "Enhanced search criteria created", // ✅ DEBUG уровень
		"message_ids_count", len(allMessageIDs),
		"subject_preview", a.getPreview(threadData.Subject, 30))
	// ✅ УБИРАЕМ: since, search_strategies - избыточно для каждого вызова

	return criteria, nil // ✅ ВОЗВРАЩАЕМ error
}

// removeDuplicateMessageIDs - удаляем дубликаты Message-ID
func (a *IMAPAdapter) removeDuplicateMessageIDs(ids []string) []string {
	seen := make(map[string]bool)
	var result []string

	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		// Нормализуем Message-ID (убираем < > если есть)
		normalizedID := strings.Trim(id, "<>")
		if !seen[normalizedID] {
			seen[normalizedID] = true
			result = append(result, normalizedID)
		}
	}

	return result
}

// createThreadSearchCriteria создает комбинированные критерии для поиска цепочек
func (a *IMAPAdapter) createThreadSearchCriteria(threadData ports.ThreadSearchCriteria) *imap.SearchCriteria {
	criteria := &imap.SearchCriteria{}

	// ✅ СТРАТЕГИЯ 1: Поиск по Message-ID цепочки
	var messageIDs []string
	if threadData.MessageID != "" {
		messageIDs = append(messageIDs, threadData.MessageID)
	}
	if threadData.InReplyTo != "" {
		messageIDs = append(messageIDs, threadData.InReplyTo)
	}
	if len(threadData.References) > 0 {
		messageIDs = append(messageIDs, threadData.References...)
	}

	if len(messageIDs) > 0 {
		criteria.Header = map[string][]string{
			"Message-ID": messageIDs,
		}
	}

	// ✅ СТРАТЕГИЯ 2: Поиск по Subject (нормализованному)
	if threadData.Subject != "" {
		normalizedSubject := a.normalizeThreadSubject(threadData.Subject)
		if criteria.Header == nil {
			criteria.Header = make(map[string][]string)
		}
		criteria.Header["Subject"] = []string{normalizedSubject}
	}

	// ✅ СТРАТЕГИЯ 3: Расширенный временной диапазон
	criteria.Since = time.Now().Add(-90 * 24 * time.Hour) // 90 дней

	// ✅ СТРАТЕГИЯ 4: Включаем прочитанные сообщения для полных цепочек
	// (не устанавливаем WithoutFlags для Seen)

	a.logger.Debug(context.Background(), "Created thread search criteria",
		"message_ids_count", len(messageIDs),
		"subject", threadData.Subject,
		"since_days", 90)

	return criteria
}

// normalizeThreadSubject нормализует subject для поиска цепочек
func (a *IMAPAdapter) normalizeThreadSubject(subject string) string {
	// Убираем префиксы ответов
	prefixes := []string{"Re:", "Fwd:", "FW:", "RE:", "Ответ:", "FWD:"}
	result := subject

	for _, prefix := range prefixes {
		if strings.HasPrefix(strings.ToUpper(result), strings.ToUpper(prefix)) {
			result = strings.TrimSpace(result[len(prefix):])
		}
	}

	return result
}

// describeSearchCriteria создает описание критериев для логирования
func (a *IMAPAdapter) describeSearchCriteria(criteria *imap.SearchCriteria) string {
	description := []string{}

	if criteria.Since != (time.Time{}) {
		description = append(description, fmt.Sprintf("since:%s", criteria.Since.Format("2006-01-02")))
	}

	if criteria.Header != nil {
		for key, values := range criteria.Header {
			description = append(description, fmt.Sprintf("%s:%v", key, values))
		}
	}

	if len(criteria.WithoutFlags) > 0 {
		description = append(description, fmt.Sprintf("without_flags:%v", criteria.WithoutFlags))
	}

	return strings.Join(description, ", ")
}

// convertToDomainMessage конвертирует IMAP сообщение в доменную сущность
func (a *IMAPAdapter) convertToDomainMessage(imapMsg *imap.Message) (domain.EmailMessage, error) {
	var email domain.EmailMessage

	if imapMsg.Envelope == nil {
		return domain.EmailMessage{}, fmt.Errorf("IMAP message has no envelope")
	}

	// Получаем базовую информацию из envelope
	envelopeInfo := imapclient.GetMessageEnvelopeInfo(imapMsg)
	if envelopeInfo != nil {
		email.MessageID = envelopeInfo.MessageID
		email.Subject = envelopeInfo.Subject
		email.From = domain.EmailAddress(strings.Join(envelopeInfo.From, ", "))
		email.InReplyTo = envelopeInfo.InReplyTo
	}

	// Парсим References из заголовков
	headers := a.extractAllHeaders(imapMsg)
	if refs, exists := headers["References"]; exists && len(refs) > 0 {
		email.References = strings.Fields(refs[0])
	}

	// Дополняем: Если In-Reply-To пустой в envelope, берем из заголовков
	if email.InReplyTo == "" {
		if inReplyTos, exists := headers["In-Reply-To"]; exists && len(inReplyTos) > 0 {
			email.InReplyTo = inReplyTos[0]
		}
	}

	// ✅ ИСПОЛЬЗУЕМ УЛУЧШЕННЫЙ parseMessageBody
	bodyInfo, err := a.parseMessageBody(imapMsg)
	if err != nil {
		a.logger.Warn(context.Background(), "Failed to parse message body",
			"message_id", email.MessageID,
			"error", err.Error())
	}

	// Конвертируем адреса
	from := a.extractPrimaryAddress(envelopeInfo.From)
	to := a.extractAddresses(envelopeInfo.To)

	// ✅ СОЗДАЕМ domainMsg С РАСПАРСЕННЫМ ТЕЛОМ
	domainMsg := domain.EmailMessage{
		MessageID:   email.MessageID,
		InReplyTo:   email.InReplyTo,
		References:  email.References,
		From:        domain.EmailAddress(from),
		To:          a.convertToDomainAddresses(to),
		Subject:     email.Subject,
		Direction:   domain.DirectionIncoming,
		Source:      "imap",
		BodyText:    bodyInfo.Text,        // ✅ ТЕКСТОВОЕ ТЕЛО
		BodyHTML:    bodyInfo.HTML,        // ✅ HTML ТЕЛО
		Attachments: bodyInfo.Attachments, // ✅ ВЛОЖЕНИЯ
		CreatedAt:   envelopeInfo.Date,
		UpdatedAt:   time.Now(),
		Headers:     make(map[string][]string),
	}

	// Сохраняем IMAP UID в headers для отслеживания
	if imapMsg.Uid > 0 {
		domainMsg.Headers["X-IMAP-UID"] = []string{fmt.Sprintf("%d", imapMsg.Uid)}
	}

	// ✅ ЛОГИРУЕМ РЕЗУЛЬТАТ С ТЕЛОМ ПИСЬМА
	a.logger.Info(context.Background(), "Domain message converted with body content",
		"message_id", domainMsg.MessageID,
		"body_text_length", len(domainMsg.BodyText),
		"body_html_length", len(domainMsg.BodyHTML),
		"attachments_count", len(domainMsg.Attachments),
		"references_count", len(domainMsg.References))

	return domainMsg, nil
}

// convertToDomainMessageWithBody - ФИНАЛЬНАЯ ИСПРАВЛЕННАЯ ВЕРСИЯ
func (a *IMAPAdapter) convertToDomainMessageWithBody(imapMsg *imap.Message) (domain.EmailMessage, error) {
	if imapMsg.Envelope == nil {
		return domain.EmailMessage{}, fmt.Errorf("IMAP message has no envelope")
	}

	// ✅ КРИТИЧЕСКОЕ: Сохраняем данные ПЕРВЫМ действием
	rawData, err := a.preserveMessageData(imapMsg)
	if err != nil {
		return domain.EmailMessage{}, fmt.Errorf("failed to preserve message data: %w", err)
	}

	// ✅ ИСПОЛЬЗУЕМ сохраненные данные для ВСЕХ операций
	headers := a.extractHeadersFromPreservedData(rawData)
	bodyInfo := a.parseBodyFromPreservedData(rawData)

	// ✅ ИСПРАВЛЕНО: используем правильный тип EnvelopeInfo
	envelopeInfo := imapclient.GetMessageEnvelopeInfo(imapMsg)

	// ✅ ВОССТАНАВЛИВАЕМ THREADING ДАННЫЕ
	allReferences := a.extractThreadingData(headers, envelopeInfo)
	finalInReplyTo := a.determineInReplyTo(headers, envelopeInfo)

	// ✅ СОЗДАЕМ ДОМЕННОЕ СООБЩЕНИЕ
	domainMsg, err := a.buildDomainMessage(
		envelopeInfo,
		headers,
		bodyInfo,
		allReferences,
		finalInReplyTo,
	)
	if err != nil {
		return domain.EmailMessage{}, err
	}

	// ✅ ДЕТАЛЬНАЯ ВАЛИДАЦИЯ РЕЗУЛЬТАТА
	a.validateMessageConversion(domainMsg, rawData)

	return domainMsg, nil
}

func (a *IMAPAdapter) preserveMessageData(imapMsg *imap.Message) ([]byte, error) {
	a.logger.Debug(context.Background(), "Preserving message data", // ✅ DEBUG уровень
		"available_sections", len(imapMsg.Body))

	// ✅ УБИРАЕМ цикл логирования каждой секции - слишком детально

	// Ищем подходящую секцию для чтения
	for sectionName, literal := range imapMsg.Body {
		if literal == nil {
			continue
		}

		if a.isReadableSection(sectionName) {
			data, err := io.ReadAll(literal)
			if err != nil {
				a.logger.Debug(context.Background(), "Failed to read section, trying next", // ✅ DEBUG уровень
					"section", sectionName.Specifier, "error", err.Error())
				continue
			}

			a.logger.Debug(context.Background(), "Message data preserved", // ✅ DEBUG уровень
				"section", sectionName.Specifier,
				"data_length", len(data))
			// ✅ УБИРАЕМ data_preview и детальные проверки - слишком много шума

			return data, nil
		}
	}

	return nil, fmt.Errorf("no readable body sections found among %d available sections", len(imapMsg.Body))
}

// isReadableSection - исправленная версия для работы с указателем
func (a *IMAPAdapter) isReadableSection(sectionName *imap.BodySectionName) bool {
	if sectionName == nil {
		return false
	}

	// ✅ Читаем все секции, которые могут содержать полное сообщение
	readable := sectionName.Specifier == imap.EntireSpecifier || // BODY[]
		sectionName.Specifier == imap.TextSpecifier || // BODY[TEXT]
		sectionName.Specifier == "" || // RFC822
		len(sectionName.Path) == 0 // корневая секция

	a.logger.Debug(context.Background(), "Section readability check",
		"specifier", sectionName.Specifier,
		"path", sectionName.Path,
		"is_readable", readable)

	return readable
}

// extractHeadersFromPreservedData - УЛУЧШЕННАЯ ВЕРСИЯ extractAllHeaders
func (a *IMAPAdapter) extractHeadersFromPreservedData(rawData []byte) map[string][]string {
	headers := make(map[string][]string)

	reader := bytes.NewReader(rawData)
	scanner := bufio.NewScanner(reader)

	var currentHeader string
	var currentValue strings.Builder

	for scanner.Scan() {
		line := scanner.Text()

		// Конец заголовков
		if line == "" {
			break
		}

		// ✅ СОХРАНЯЕМ ПРОВЕРЕННУЮ ЛОГИКУ ДЛЯ МНОГОСТРОЧНЫХ REFERENCES
		if strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
			// Продолжение предыдущего заголовка
			if currentHeader != "" {
				currentValue.WriteString(" ")
				currentValue.WriteString(strings.TrimSpace(line))
			}
		} else {
			// Сохраняем предыдущий заголовок если есть
			if currentHeader != "" {
				headers[currentHeader] = append(headers[currentHeader], currentValue.String())
				currentValue.Reset()
			}

			// Новый заголовок
			parts := strings.SplitN(line, ":", 2)
			if len(parts) != 2 {
				continue
			}

			currentHeader = strings.TrimSpace(parts[0])
			currentValue.WriteString(strings.TrimSpace(parts[1]))
		}
	}

	// Сохраняем последний заголовок
	if currentHeader != "" {
		headers[currentHeader] = append(headers[currentHeader], currentValue.String())
	}

	// ✅ ОПТИМИЗИРУЕМ: только ключевая информация о References
	if refs, exists := headers["References"]; exists && len(refs) > 0 {
		referencesList := strings.Fields(refs[0])
		a.logger.Debug(context.Background(), "References header parsed", // ✅ DEBUG уровень
			"references_count", len(referencesList))
		// ✅ УБИРАЕМ raw_value, parsed_references - слишком много шума
	}

	a.logger.Debug(context.Background(), "Headers extraction completed", // ✅ DEBUG уровень
		"total_headers", len(headers))
	// ✅ УБИРАЕМ critical_headers - избыточно

	return headers
}

// parseBodyFromPreservedData - парсим тело из сохраненных данных
func (a *IMAPAdapter) parseBodyFromPreservedData(rawData []byte) *MessageBodyInfo {
	if len(rawData) == 0 {
		a.logger.Warn(context.Background(), "Raw data is empty, cannot parse body")
		return &MessageBodyInfo{}
	}

	mimeParser := NewMIMEParser(a.logger)
	parsed, err := mimeParser.ParseMessage(rawData)
	if err != nil {
		a.logger.Error(context.Background(), "MIME parsing failed from preserved data",
			"error", err.Error(),
			"raw_data_length", len(rawData))
		return &MessageBodyInfo{}
	}

	result := &MessageBodyInfo{
		Text:        parsed.Text,
		HTML:        parsed.HTML,
		Attachments: parsed.Attachments,
	}

	a.logger.Info(context.Background(), "✅ Body parsed from preserved data",
		"text_length", len(result.Text),
		"html_length", len(result.HTML),
		"attachments_count", len(result.Attachments),
		"text_preview", a.getPreview(result.Text, 100))

	return result
}

func (a *IMAPAdapter) extractThreadingData(headers map[string][]string, envelopeInfo *imapclient.EnvelopeInfo) []string {
	var allReferences []string

	// ✅ ОПТИМИЗИРУЕМ: только счетчики, без деталей
	if refs, exists := headers["References"]; exists && len(refs) > 0 {
		extracted := strings.Fields(refs[0])
		allReferences = append(allReferences, extracted...)
		// ✅ УБИРАЕМ детальное логирование extracted_refs
	}

	if envelopeInfo != nil && len(envelopeInfo.References) > 0 {
		allReferences = append(allReferences, envelopeInfo.References...)
		// ✅ УБИРАЕМ детальное логирование envelope_refs
	}

	allReferences = a.removeDuplicateReferences(allReferences)

	a.logger.Debug(context.Background(), "Threading data extracted", // ✅ DEBUG уровень
		"total_references", len(allReferences))
	// ✅ УБИРАЕМ список всех references и source - избыточно

	return allReferences
}

// determineInReplyTo - исправленная версия
func (a *IMAPAdapter) determineInReplyTo(headers map[string][]string, envelopeInfo *imapclient.EnvelopeInfo) string {
	if envelopeInfo != nil && envelopeInfo.InReplyTo != "" {
		return envelopeInfo.InReplyTo
	}

	if inReplyTos, exists := headers["In-Reply-To"]; exists && len(inReplyTos) > 0 {
		return inReplyTos[0]
	}

	return ""
}

// buildDomainMessage - создаем финальное доменное сообщение
// buildDomainMessage - исправленная версия
func (a *IMAPAdapter) buildDomainMessage(
	envelopeInfo *imapclient.EnvelopeInfo,
	headers map[string][]string,
	bodyInfo *MessageBodyInfo,
	references []string,
	inReplyTo string,
) (domain.EmailMessage, error) {

	if envelopeInfo == nil {
		return domain.EmailMessage{}, fmt.Errorf("envelope info is required")
	}

	// Нормализуем адреса
	fromAddr, err := a.addressNormalizer.NormalizeEmailAddress(envelopeInfo.From[0])
	if err != nil {
		return domain.EmailMessage{}, fmt.Errorf("failed to normalize from address: %w", err)
	}

	toAddrs := a.addressNormalizer.ConvertToDomainAddresses(envelopeInfo.To)
	ccAddrs := a.addressNormalizer.ConvertToDomainAddresses(envelopeInfo.CC)

	domainMsg := domain.EmailMessage{
		MessageID:   envelopeInfo.MessageID,
		InReplyTo:   inReplyTo,
		References:  references,
		From:        domain.EmailAddress(fromAddr),
		To:          toAddrs,
		CC:          ccAddrs,
		Subject:     envelopeInfo.Subject,
		BodyText:    bodyInfo.Text,
		BodyHTML:    bodyInfo.HTML,
		Attachments: bodyInfo.Attachments,
		Direction:   domain.DirectionIncoming,
		Source:      "imap",
		Headers:     headers,
		CreatedAt:   envelopeInfo.Date,
		UpdatedAt:   time.Now(),
	}

	return domainMsg, nil
}

func (a *IMAPAdapter) validateMessageConversion(domainMsg domain.EmailMessage, rawData []byte) {
	a.logger.Debug(context.Background(), "Message conversion validated", // ✅ DEBUG уровень
		"message_id", domainMsg.MessageID,
		"body_text_length", len(domainMsg.BodyText),
		"references_count", len(domainMsg.References),
		"attachments_count", len(domainMsg.Attachments))
	// ✅ УБИРАЕМ: raw_data_length, body_html_length, has_threading_data - избыточно

	// ✅ ОПТИМИЗИРУЕМ: только критические предупреждения
	if len(domainMsg.BodyText) == 0 && len(domainMsg.BodyHTML) == 0 {
		a.logger.Warn(context.Background(), "No message content extracted",
			"message_id", domainMsg.MessageID)
		// ✅ УБИРАЕМ raw_data_sample - слишком много шума
	}
	// ✅ УБИРАЕМ дополнительное логирование успеха - избыточно
}

// removeDuplicateReferences - удаляем дубликаты (сохраняем порядок)
func (a *IMAPAdapter) removeDuplicateReferences(refs []string) []string {
	seen := make(map[string]bool)
	var result []string

	for _, ref := range refs {
		ref = strings.TrimSpace(ref)
		if ref == "" {
			continue
		}
		if !seen[ref] {
			seen[ref] = true
			result = append(result, ref)
		}
	}

	return result
}

// parseMessageBody парсит тело сообщения из RFC822 секции
func (a *IMAPAdapter) parseMessageBody(imapMsg *imap.Message) (*MessageBodyInfo, error) {
	bodyInfo := &MessageBodyInfo{
		Text:        "",
		HTML:        "",
		Attachments: []domain.Attachment{},
	}

	if imapMsg.Body == nil {
		a.logger.Debug(context.Background(), "IMAP message has no body sections")
		return bodyInfo, nil
	}

	// ✅ ПРАВИЛЬНО: Ищем RFC822 секции по BodySectionName
	for sectionName, literal := range imapMsg.Body {
		a.logger.Debug(context.Background(), "Checking IMAP body section",
			"section_specifier", sectionName.Specifier,
			"section_path", sectionName.Path,
			"has_literal", literal != nil)

		// ✅ ПРАВИЛЬНО: Сравниваем по Specifier, а не по строке
		if sectionName.Specifier == imap.EntireSpecifier || // BODY[]
			sectionName.Specifier == imap.TextSpecifier || // BODY[TEXT]
			(sectionName.Specifier == "" && len(sectionName.Path) == 0) { // RFC822

			if literal == nil {
				a.logger.Debug(context.Background(), "Section has no literal",
					"specifier", sectionName.Specifier)
				continue
			}

			// ✅ КРИТИЧЕСКОЕ ИСПРАВЛЕНИЕ: Сохраняем данные при первом чтении
			data, err := io.ReadAll(literal)
			if err != nil {
				a.logger.Warn(context.Background(), "Failed to read body section",
					"specifier", sectionName.Specifier,
					"error", err.Error())
				continue
			}

			// ✅ ДИАГНОСТИКА: Логируем реальные данные
			a.logger.Info(context.Background(), "Found body section with data",
				"specifier", sectionName.Specifier,
				"data_length", len(data),
				"data_preview", string(data[:min(200, len(data))]))

			// ✅ ПЕРЕДАЕМ СОХРАНЕННЫЕ ДАННЫЕ В MIME ПАРСЕР
			mimeParser := NewMIMEParser(a.logger)
			parsed, err := mimeParser.ParseMessage(data) // ← Используем СОХРАНЕННЫЕ данные
			if err != nil {
				a.logger.Warn(context.Background(), "MIME parsing failed",
					"specifier", sectionName.Specifier,
					"error", err.Error())
				continue
			}

			bodyInfo.Text = parsed.Text
			bodyInfo.HTML = parsed.HTML
			bodyInfo.Attachments = parsed.Attachments

			a.logger.Info(context.Background(), "MIME parsing completed with content",
				"specifier", sectionName.Specifier,
				"text_length", len(bodyInfo.Text),
				"html_length", len(bodyInfo.HTML),
				"attachments_count", len(bodyInfo.Attachments),
				"text_preview", a.getPreview(bodyInfo.Text, 100))

			return bodyInfo, nil
		}
	}

	a.logger.Warn(context.Background(), "No suitable body sections found")
	return bodyInfo, nil
}

// extractAllHeaders извлекает все RFC заголовки из IMAP сообщения
func (a *IMAPAdapter) extractAllHeaders(imapMsg *imap.Message) map[string][]string {
	headers := make(map[string][]string)

	if len(imapMsg.Body) == 0 {
		return headers
	}

	for _, body := range imapMsg.Body {
		if body == nil {
			continue
		}

		reader, ok := body.(io.Reader)
		if !ok {
			continue
		}

		scanner := bufio.NewScanner(reader)
		var currentHeader string
		var currentValue strings.Builder

		for scanner.Scan() {
			line := scanner.Text()

			// Конец заголовков
			if line == "" {
				break
			}

			// ✅ ИСПРАВЛЯЕМ: Обработка многострочных заголовков
			if strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
				// Продолжение предыдущего заголовка
				if currentHeader != "" {
					currentValue.WriteString(" ")
					currentValue.WriteString(strings.TrimSpace(line))
				}
			} else {
				// Сохраняем предыдущий заголовок если есть
				if currentHeader != "" {
					headers[currentHeader] = append(headers[currentHeader], currentValue.String())
					currentValue.Reset()
				}

				// Новый заголовок
				parts := strings.SplitN(line, ":", 2)
				if len(parts) != 2 {
					continue
				}

				currentHeader = strings.TrimSpace(parts[0])
				currentValue.WriteString(strings.TrimSpace(parts[1]))
			}
		}

		// Сохраняем последний заголовок
		if currentHeader != "" {
			headers[currentHeader] = append(headers[currentHeader], currentValue.String())
		}

		break // Только первый body part содержит заголовки
	}

	// ✅ ДОБАВЛЯЕМ ЛОГИРОВАНИЕ ДЛЯ ПРОВЕРКИ REFERENCES
	if refs, exists := headers["References"]; exists {
		a.logger.Debug(context.Background(), "Extracted References header",
			"raw_references", refs,
			"references_count", len(refs),
			"first_reference_length", len(refs[0]))
	}

	return headers
}

// parseMultipartMessage парсит multipart сообщение
// func (a *IMAPAdapter) parseMultipartMessage(imapMsg *imap.Message, structure *imap.BodyStructure) (*MessageBodyInfo, error) {
// 	// Временная заглушка - полная реализация будет в MIME парсере
// 	return &MessageBodyInfo{
// 		Text:        "",
// 		HTML:        "",
// 		Attachments: []domain.Attachment{},
// 	}, nil
// }

// // parseSimpleMessage парсит простое сообщение
// func (a *IMAPAdapter) parseSimpleMessage(imapMsg *imap.Message, structure *imap.BodyStructure) (*MessageBodyInfo, error) {
// 	// Временная заглушка - полная реализация будет в MIME парсере
// 	return &MessageBodyInfo{
// 		Text:        "",
// 		HTML:        "",
// 		Attachments: []domain.Attachment{},
// 	}, nil
// }

// extractPrimaryAddress извлекает основной адрес из списка
func (a *IMAPAdapter) extractPrimaryAddress(addresses []string) string {
	if len(addresses) == 0 {
		return ""
	}
	return addresses[0]
}

// extractAddresses извлекает адреса
func (a *IMAPAdapter) extractAddresses(addresses []string) []string {
	return addresses
}

// convertToDomainAddresses конвертирует строки в domain.EmailAddress
func (a *IMAPAdapter) convertToDomainAddresses(addresses []string) []domain.EmailAddress {
	result := make([]domain.EmailAddress, len(addresses))
	for i, addr := range addresses {
		result[i] = domain.EmailAddress(addr)
	}
	return result
}

// getDataPreview - вспомогательный метод для preview данных
func (a *IMAPAdapter) getDataPreview(data []byte, length int) string {
	if len(data) == 0 {
		return "[empty]"
	}
	if len(data) <= length {
		return string(data)
	}
	return string(data[:length]) + "..."
}

// getPreview - для текстового preview (из mime_parser.go, но как метод)
func (a *IMAPAdapter) getPreview(text string, length int) string {
	if text == "" {
		return "[empty]"
	}
	if len(text) <= length {
		return text
	}
	return text[:length] + "..."
}
