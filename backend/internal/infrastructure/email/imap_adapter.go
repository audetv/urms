package email

import (
	"context"
	"fmt"
	"time"

	"github.com/audetv/urms/internal/core/domain"
	"github.com/audetv/urms/internal/core/ports"
	"github.com/audetv/urms/internal/email/imapclient"
	"github.com/emersion/go-imap"
	"github.com/rs/zerolog/log"
)

// IMAPAdapter реализует ports.EmailGateway используя существующий IMAP клиент
type IMAPAdapter struct {
	client *imapclient.Client
	config *imapclient.Config
}

// NewIMAPAdapter создает новый IMAP адаптер
func NewIMAPAdapter(config *imapclient.Config) *IMAPAdapter {
	return &IMAPAdapter{
		client: imapclient.NewClient(config),
		config: config,
	}
}

// Connect устанавливает соединение с IMAP сервером
func (a *IMAPAdapter) Connect(ctx context.Context) error {
	return a.client.Connect()
}

// Disconnect закрывает соединение
func (a *IMAPAdapter) Disconnect() error {
	return a.client.Logout()
}

// HealthCheck проверяет состояние соединения
func (a *IMAPAdapter) HealthCheck(ctx context.Context) error {
	return a.client.CheckConnection()
}

// FetchMessages получает сообщения по критериям
func (a *IMAPAdapter) FetchMessages(ctx context.Context, criteria ports.FetchCriteria) ([]domain.EmailMessage, error) {
	// ВЫБИРАЕМ почтовый ящик перед поиском
	if err := a.SelectMailbox(ctx, criteria.Mailbox); err != nil {
		return nil, fmt.Errorf("failed to select mailbox %s: %w", criteria.Mailbox, err)
	}

	// Конвертируем доменные критерии в IMAP-специфичные
	imapCriteria := a.convertToIMAPCriteria(criteria)

	// Ищем сообщения по UID
	messageUIDs, err := a.client.SearchMessages(imapCriteria)
	if err != nil {
		return nil, fmt.Errorf("failed to search messages: %w", err)
	}

	if len(messageUIDs) == 0 {
		return []domain.EmailMessage{}, nil
	}

	// Создаем SeqSet для получения сообщений
	seqSet := new(imap.SeqSet)
	for _, uid := range messageUIDs {
		seqSet.AddNum(uid)
	}

	// Получаем сообщения с envelope информацией
	fetchItems := imapclient.CreateFetchItems(false) // Без тела для начала
	messagesChan, err := a.client.FetchMessages(seqSet, fetchItems)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch messages: %w", err)
	}

	// Конвертируем IMAP сообщения в доменные сущности
	var domainMessages []domain.EmailMessage
	for msg := range messagesChan {
		domainMsg, err := a.convertToDomainMessage(msg)
		if err != nil {
			log.Warn().Err(err).Uint32("uid", msg.Uid).Msg("Failed to convert IMAP message")
			continue
		}
		domainMessages = append(domainMessages, domainMsg)
	}

	return domainMessages, nil
}

// SendMessage отправляет сообщение (заглушка - будет реализовано в SMTP адаптере)
func (a *IMAPAdapter) SendMessage(ctx context.Context, msg domain.EmailMessage) error {
	return fmt.Errorf("IMAP adapter does not support sending messages. Use SMTP adapter instead")
}

// MarkAsRead помечает сообщения как прочитанные
func (a *IMAPAdapter) MarkAsRead(ctx context.Context, messageIDs []string) error {
	// TODO: Реализовать пометку сообщений как прочитанных
	// Пока возвращаем заглушку
	log.Info().Strs("message_ids", messageIDs).Msg("Marking messages as read")
	return nil
}

// MarkAsProcessed помечает сообщения как обработанные
func (a *IMAPAdapter) MarkAsProcessed(ctx context.Context, messageIDs []string) error {
	// IMAP не поддерживает эту операцию напрямую
	// Можно реализовать через перемещение в другую папку
	log.Info().Strs("message_ids", messageIDs).Msg("Marking messages as processed")
	return nil
}

// ListMailboxes возвращает список почтовых ящиков
func (a *IMAPAdapter) ListMailboxes(ctx context.Context) ([]ports.MailboxInfo, error) {
	// TODO: Реализовать получение списка почтовых ящиков
	// Пока возвращаем только INBOX
	return []ports.MailboxInfo{
		{
			Name:     "INBOX",
			Messages: 0, // Можно получить через GetMailboxInfo
			Unseen:   0,
			Recent:   0,
		},
	}, nil
}

// SelectMailbox выбирает почтовый ящик
func (a *IMAPAdapter) SelectMailbox(ctx context.Context, name string) error {
	_, err := a.client.SelectMailbox(name, a.config.ReadOnly)
	return err
}

// GetMailboxInfo возвращает информацию о почтовом ящике
func (a *IMAPAdapter) GetMailboxInfo(ctx context.Context, name string) (*ports.MailboxInfo, error) {
	mailbox, err := a.client.GetMailboxInfo(name)
	if err != nil {
		return nil, fmt.Errorf("failed to get mailbox info: %w", err)
	}

	return &ports.MailboxInfo{
		Name:     mailbox.Name,
		Messages: int(mailbox.Messages),
		Unseen:   int(mailbox.Unseen),
		Recent:   int(mailbox.Recent),
	}, nil
}

// convertToIMAPCriteria конвертирует доменные критерии в IMAP-специфичные
func (a *IMAPAdapter) convertToIMAPCriteria(criteria ports.FetchCriteria) *imap.SearchCriteria {
	imapCriteria := &imap.SearchCriteria{}

	// Поиск по UID
	if criteria.SinceUID > 0 {
		imapCriteria.Uid = new(imap.SeqSet)
		imapCriteria.Uid.AddNum(criteria.SinceUID+1, 0) // 0 означает "*" - все последующие
	}

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
	// TODO: Добавить IDGenerator когда будем создавать сообщения
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
