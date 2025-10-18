package email

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/audetv/urms/internal/core/domain"
	"github.com/audetv/urms/internal/core/ports"
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
	mimeParser        *MIMEParser
	addressNormalizer *AddressNormalizer
	retryManager      *RetryManager
	timeoutConfig     TimeoutConfig
	logger            ports.Logger // ✅ ДОБАВЛЯЕМ ports.Logger
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
func NewIMAPAdapter(config *imapclient.Config, timeoutConfig TimeoutConfig, logger ports.Logger) *IMAPAdapter {
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
		mimeParser:        NewMIMEParser(),
		addressNormalizer: NewAddressNormalizer(),
		retryManager:      NewRetryManager(retryConfig, logger),
		timeoutConfig:     timeoutConfig,
		logger:            logger, // ✅ ДОБАВЛЯЕМ logger
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
		mimeParser:        NewMIMEParser(),
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

	return NewIMAPAdapter(config, defaultTimeoutConfig, testLogger)
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

// ✅ NEW: Вспомогательный метод для получения логгера с context
// func (a *IMAPAdapter) getLogger(ctx context.Context) *zerolog.Logger {
// 	// Временная реализация - в реальной реализации нужно интегрировать ports.Logger
// 	logger := zerolog.Ctx(ctx)
// 	if logger.GetLevel() == zerolog.Disabled {
// 		consoleWriter := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}
// 		defaultLogger := zerolog.New(consoleWriter).With().Timestamp().Logger()
// 		return &defaultLogger
// 	}
// 	return logger
// }

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
	imapCriteria := a.convertToIMAPCriteria(ctx, criteria)

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
	a.logger.Debug(ctx, "Fetching message batch",
		"operation", "fetch_batch",
		"message_count", len(messageUIDs))

	// ✅ ДОБАВЛЕНО: Создаем контекст с таймаутом для batch операций
	batchCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Создаем SeqSet для получения сообщений
	seqSet := new(imap.SeqSet)
	for _, uid := range messageUIDs {
		seqSet.AddNum(uid)
	}

	// ✅ ИСПРАВЛЕНО: Используем контекст с таймаутом
	fetchItems := imapclient.CreateFetchItems(false) // Без тела для начала
	messagesChan, err := a.client.FetchMessages(seqSet, fetchItems)
	if err != nil {
		a.logger.Error(ctx, "Failed to fetch messages", "operation", "fetch_batch", "error", err.Error())
		return nil, NewIMAPError("fetch_messages", IMAPErrorProtocol, "failed to fetch messages", err)
	}

	// Конвертируем IMAP сообщения в доменные сущности
	var domainMessages []domain.EmailMessage
	for msg := range messagesChan {
		select {
		case <-batchCtx.Done():
			// ✅ ДОБАВЛЕНО: Прерываем обработку при таймауте
			a.logger.Warn(ctx, "Batch processing interrupted by timeout",
				"operation", "fetch_batch",
				"processed", len(domainMessages),
				"error", batchCtx.Err().Error())
			return domainMessages, batchCtx.Err()
		default:
			domainMsg, err := a.convertToDomainMessage(msg)
			if err != nil {
				a.logger.Warn(ctx, "Failed to convert IMAP message",
					"operation", "fetch_batch",
					"uid", msg.Uid,
					"error", err.Error())
				continue
			}
			domainMessages = append(domainMessages, domainMsg)
		}
	}

	a.logger.Debug(ctx, "Message batch conversion completed",
		"operation", "fetch_batch",
		"converted_messages", len(domainMessages))

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

	// Ограничение по дате - только последний час для тестирования
	imapCriteria.Since = time.Now().Add(-1 * time.Hour)

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

	// ✅ УВЕЛИЧИВАЕМ ОГРАНИЧЕНИЕ: Берем только первые 10 сообщений для тестирования
	if len(messageUIDs) > 10 {
		messageUIDs = messageUIDs[:10]
		a.logger.Info(ctx, "Limited initial message fetch for testing",
			"operation", "fetch_initial",
			"limited_to", 10)
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

	imapCriteria := a.convertToIMAPCriteria(ctx, criteria)
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
func (a *IMAPAdapter) convertToIMAPCriteria(ctx context.Context, criteria ports.FetchCriteria) *imap.SearchCriteria {
	imapCriteria := &imap.SearchCriteria{}

	// ✅ ИСПРАВЛЕНО: Для первого запроса (sinceUID=0) ищем все непрочитанные сообщения
	if criteria.SinceUID == 0 {
		// Ищем непрочитанные сообщения за последние 24 часа
		if criteria.UnseenOnly {
			imapCriteria.WithoutFlags = []string{imap.SeenFlag}
		}

		// Ограничиваем поиск по дате если указано
		if !criteria.Since.IsZero() {
			imapCriteria.Since = criteria.Since
		} else {
			// По умолчанию ищем за последние 7 дней
			imapCriteria.Since = time.Now().Add(-7 * 24 * time.Hour)
		}

		a.logger.Info(ctx, "Using date-based search for initial polling",
			"since", imapCriteria.Since,
			"unseen_only", criteria.UnseenOnly)
	} else {
		// Для последующих запросов используем UID-based поиск
		// UID будет установлен в пагинации
		a.logger.Debug(ctx, "Using UID-based search for pagination", "since_uid", criteria.SinceUID)
	}

	return imapCriteria
}

// convertToDomainMessage конвертирует IMAP сообщение в доменную сущность
func (a *IMAPAdapter) convertToDomainMessage(imapMsg *imap.Message) (domain.EmailMessage, error) {
	if imapMsg.Envelope == nil {
		return domain.EmailMessage{}, fmt.Errorf("IMAP message has no envelope")
	}

	// Извлекаем базовую информацию
	envelopeInfo := imapclient.GetMessageEnvelopeInfo(imapMsg)
	if envelopeInfo == nil {
		return domain.EmailMessage{}, fmt.Errorf("failed to extract envelope info")
	}

	// Конвертируем адреса
	from := a.extractPrimaryAddress(envelopeInfo.From)
	to := a.extractAddresses(envelopeInfo.To)

	// Создаем доменное сообщение
	domainMsg := domain.EmailMessage{
		MessageID: envelopeInfo.MessageID,
		InReplyTo: envelopeInfo.InReplyTo,
		From:      domain.EmailAddress(from),
		To:        a.convertToDomainAddresses(to),
		Subject:   envelopeInfo.Subject,
		Direction: domain.DirectionIncoming,
		Source:    "imap",
		CreatedAt: envelopeInfo.Date,
		UpdatedAt: time.Now(),
		Headers:   make(map[string][]string),
	}

	// ✅ NEW: Сохраняем IMAP UID в headers для отслеживания
	if imapMsg.Uid > 0 {
		domainMsg.Headers["X-IMAP-UID"] = []string{fmt.Sprintf("%d", imapMsg.Uid)}
	}

	// Добавляем References если есть
	if len(imapMsg.Envelope.InReplyTo) > 0 {
		for _, ref := range imapMsg.Envelope.InReplyTo {
			domainMsg.References = append(domainMsg.References, string(ref))
		}
	}

	return domainMsg, nil
}

// convertToDomainMessageWithBody конвертирует IMAP сообщение с полным парсингом
func (a *IMAPAdapter) convertToDomainMessageWithBody(imapMsg *imap.Message) (domain.EmailMessage, error) {
	if imapMsg.Envelope == nil {
		return domain.EmailMessage{}, fmt.Errorf("IMAP message has no envelope")
	}

	// Извлекаем базовую информацию
	envelopeInfo := imapclient.GetMessageEnvelopeInfo(imapMsg)
	if envelopeInfo == nil {
		return domain.EmailMessage{}, fmt.Errorf("failed to extract envelope info")
	}

	// Парсим тело сообщения и вложения
	bodyInfo, err := a.parseMessageBody(imapMsg)
	if err != nil {
		return domain.EmailMessage{}, fmt.Errorf("failed to parse message body: %w", err)
	}

	// Извлекаем все RFC заголовки
	headers := a.extractAllHeaders(imapMsg)

	// Нормализуем адреса
	fromAddr, err := a.addressNormalizer.NormalizeEmailAddress(envelopeInfo.From[0])
	if err != nil {
		return domain.EmailMessage{}, fmt.Errorf("failed to normalize from address: %w", err)
	}

	toAddrs := a.addressNormalizer.ConvertToDomainAddresses(envelopeInfo.To)
	ccAddrs := a.addressNormalizer.ConvertToDomainAddresses(envelopeInfo.CC)

	// Создаем доменное сообщение с полной информацией
	domainMsg := domain.EmailMessage{
		MessageID:   envelopeInfo.MessageID,
		InReplyTo:   envelopeInfo.InReplyTo,
		References:  envelopeInfo.References,
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

// parseMessageBody парсит тело сообщения и вложения
func (a *IMAPAdapter) parseMessageBody(imapMsg *imap.Message) (*MessageBodyInfo, error) {
	bodyInfo := &MessageBodyInfo{
		Text:        "",
		HTML:        "",
		Attachments: []domain.Attachment{},
	}

	if imapMsg.Body == nil {
		return bodyInfo, nil
	}

	// Для полного парсинга нам нужно получить сырое сообщение
	// Временно возвращаем базовую информацию
	// Полный MIME парсинг будет реализован в отдельном методе
	return bodyInfo, nil
}

// extractAllHeaders извлекает все RFC заголовки из IMAP сообщения
func (a *IMAPAdapter) extractAllHeaders(imapMsg *imap.Message) map[string][]string {
	headers := make(map[string][]string)

	if imapMsg.Body == nil {
		return headers
	}

	// Временная реализация - извлекаем базовые заголовки из envelope
	// Полный парсинг заголовков будет в MIME парсере
	envelopeInfo := imapclient.GetMessageEnvelopeInfo(imapMsg)
	if envelopeInfo != nil {
		headers["From"] = envelopeInfo.From
		headers["To"] = envelopeInfo.To
		headers["Cc"] = envelopeInfo.CC
		headers["Subject"] = []string{envelopeInfo.Subject}
		headers["Message-ID"] = []string{envelopeInfo.MessageID}
		headers["Date"] = []string{envelopeInfo.Date.Format(time.RFC1123Z)}

		if envelopeInfo.InReplyTo != "" {
			headers["In-Reply-To"] = []string{envelopeInfo.InReplyTo}
		}
	}

	return headers
}

// parseMultipartMessage парсит multipart сообщение
func (a *IMAPAdapter) parseMultipartMessage(imapMsg *imap.Message, structure *imap.BodyStructure) (*MessageBodyInfo, error) {
	// Временная заглушка - полная реализация будет в MIME парсере
	return &MessageBodyInfo{
		Text:        "",
		HTML:        "",
		Attachments: []domain.Attachment{},
	}, nil
}

// parseSimpleMessage парсит простое сообщение
func (a *IMAPAdapter) parseSimpleMessage(imapMsg *imap.Message, structure *imap.BodyStructure) (*MessageBodyInfo, error) {
	// Временная заглушка - полная реализация будет в MIME парсере
	return &MessageBodyInfo{
		Text:        "",
		HTML:        "",
		Attachments: []domain.Attachment{},
	}, nil
}

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
