package email

import (
	"context"
	"fmt"
	"time"

	"github.com/audetv/urms/internal/core/domain"
	"github.com/audetv/urms/internal/core/ports"
	imapclient "github.com/audetv/urms/internal/infrastructure/email/imap"
	"github.com/emersion/go-imap"
	"github.com/rs/zerolog/log"
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
func NewIMAPAdapter(config *imapclient.Config, timeoutConfig TimeoutConfig) *IMAPAdapter {
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
		retryManager:      NewRetryManager(retryConfig),
		timeoutConfig:     timeoutConfig,
	}
}

// Connect устанавливает соединение с IMAP сервером с таймаутом
func (a *IMAPAdapter) Connect(ctx context.Context) error {
	operation := "IMAP connect"

	// Создаем контекст с таймаутом подключения
	ctx, cancel := context.WithTimeout(ctx, a.timeoutConfig.ConnectTimeout)
	defer cancel()

	return a.retryManager.ExecuteWithRetry(ctx, operation, func() error {
		err := a.client.Connect()
		if err != nil {
			return NewIMAPError("connect", IMAPErrorConnection, "failed to connect to IMAP server", err)
		}
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
	// ВЫБИРАЕМ почтовый ящик перед поиском
	if err := a.SelectMailbox(ctx, criteria.Mailbox); err != nil {
		return nil, NewIMAPError("select_mailbox", IMAPErrorServer, fmt.Sprintf("failed to select mailbox %s", criteria.Mailbox), err)
	}

	// Конвертируем доменные критерии в IMAP-специфичные
	imapCriteria := a.convertToIMAPCriteria(criteria)

	// Ищем сообщения по UID с поддержкой пагинации
	allMessages := []domain.EmailMessage{}
	lastUID := criteria.SinceUID
	processedCount := 0

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err() // Уважаем cancellation
		default:
			// Применяем пагинацию
			if processedCount >= a.timeoutConfig.MaxMessages {
				log.Info().
					Int("processed", processedCount).
					Int("max_messages", a.timeoutConfig.MaxMessages).
					Msg("Reached maximum messages per poll, stopping pagination")
				break
			}

			// Обновляем критерии для следующей страницы
			imapCriteria.Uid = a.createUIDSeqSet(lastUID, a.timeoutConfig.PageSize)

			messageUIDs, err := a.client.SearchMessages(imapCriteria)
			if err != nil {
				return nil, NewIMAPError("search_messages", IMAPErrorProtocol, "failed to search messages", err)
			}

			if len(messageUIDs) == 0 {
				return allMessages, nil // Больше нет сообщений
			}

			// Получаем сообщения текущей страницы
			batchMessages, err := a.fetchMessageBatch(ctx, messageUIDs)
			if err != nil {
				return nil, err
			}

			// Добавляем к общему результату
			allMessages = append(allMessages, batchMessages...)
			processedCount += len(batchMessages)

			// Обновляем lastUID для следующей итерации
			if len(batchMessages) > 0 {
				lastUID = a.extractMaxUID(batchMessages)
			}

			// Логируем прогресс
			log.Info().
				Int("batch_size", len(batchMessages)).
				Int("total_processed", processedCount).
				Uint32("last_uid", lastUID).
				Msg("IMAP pagination progress")

			// Если получили меньше сообщений, чем размер страницы, значит это последняя страница
			if len(batchMessages) < a.timeoutConfig.PageSize {
				break
			}
		}
	}

	return allMessages, nil
}

// fetchMessageBatch получает пачку сообщений по UID
func (a *IMAPAdapter) fetchMessageBatch(ctx context.Context, messageUIDs []uint32) ([]domain.EmailMessage, error) {
	// Создаем SeqSet для получения сообщений
	seqSet := new(imap.SeqSet)
	for _, uid := range messageUIDs {
		seqSet.AddNum(uid)
	}

	// Получаем сообщения с envelope информацией
	fetchItems := imapclient.CreateFetchItems(false) // Без тела для начала
	messagesChan, err := a.client.FetchMessages(seqSet, fetchItems)
	if err != nil {
		return nil, NewIMAPError("fetch_messages", IMAPErrorProtocol, "failed to fetch messages", err)
	}

	// Конвертируем IMAP сообщения в доменные сущности
	var domainMessages []domain.EmailMessage
	for msg := range messagesChan {
		select {
		case <-ctx.Done():
			return nil, ctx.Err() // Уважаем cancellation
		default:
			domainMsg, err := a.convertToDomainMessage(msg)
			if err != nil {
				log.Warn().Err(err).Uint32("uid", msg.Uid).Msg("Failed to convert IMAP message")
				continue
			}
			domainMessages = append(domainMessages, domainMsg)
		}
	}

	return domainMessages, nil
}

// createUIDSeqSet создает SeqSet для пагинации по UID
func (a *IMAPAdapter) createUIDSeqSet(sinceUID uint32, limit int) *imap.SeqSet {
	seqSet := new(imap.SeqSet)

	if sinceUID > 0 {
		// Начинаем со следующего UID после sinceUID
		startUID := sinceUID + 1
		// Ограничиваем количество сообщений
		endUID := startUID + uint32(limit) - 1
		seqSet.AddRange(startUID, endUID)
	} else {
		// Первый запрос - берем последние N сообщений
		// В реальной реализации нужно получить максимальный UID и отнять limit
		seqSet.AddNum(1, uint32(limit)) // Упрощенная реализация
	}

	return seqSet
}

// extractMaxUID извлекает максимальный UID из пачки сообщений
func (a *IMAPAdapter) extractMaxUID(messages []domain.EmailMessage) uint32 {
	// Временная реализация - в реальной реализации нужно извлекать UID из IMAP сообщений
	// Пока возвращаем увеличенный счетчик
	if len(messages) == 0 {
		return 0
	}
	// В Phase 1C.2 добавим реальное извлечение UID из IMAP сообщений
	return uint32(len(messages))
}

// FetchMessagesWithBody получает сообщения с полным телом и вложениями с таймаутом
func (a *IMAPAdapter) FetchMessagesWithBody(ctx context.Context, criteria ports.FetchCriteria) ([]domain.EmailMessage, error) {
	// Создаем контекст с таймаутом
	ctx, cancel := context.WithTimeout(ctx, a.timeoutConfig.FetchTimeout)
	defer cancel()

	if err := a.SelectMailbox(ctx, criteria.Mailbox); err != nil {
		return nil, fmt.Errorf("failed to select mailbox %s: %w", criteria.Mailbox, err)
	}

	imapCriteria := a.convertToIMAPCriteria(criteria)
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
				log.Warn().Err(err).Uint32("uid", msg.Uid).Msg("Failed to convert IMAP message with body")
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
		log.Info().Strs("message_ids", messageIDs).Msg("Marking messages as read")
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
		log.Info().Strs("message_ids", messageIDs).Msg("Marking messages as processed")
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
func (a *IMAPAdapter) convertToIMAPCriteria(criteria ports.FetchCriteria) *imap.SearchCriteria {
	imapCriteria := &imap.SearchCriteria{}

	// Поиск по UID (будет установлен в пагинации)
	// if criteria.SinceUID > 0 {
	//     imapCriteria.Uid = a.createUIDSeqSet(criteria.SinceUID, a.timeoutConfig.PageSize)
	// }

	// Поиск по дате
	if !criteria.Since.IsZero() {
		imapCriteria.Since = criteria.Since
	}

	// Только непрочитанные
	if criteria.UnseenOnly {
		imapCriteria.WithFlags = []string{imap.SeenFlag}
		imapCriteria.WithoutFlags = []string{imap.SeenFlag}
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
