package imapclient

import (
	"bufio"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/emersion/go-imap"
)

// EnvelopeInfo содержит полную информацию о письме из IMAP envelope
type EnvelopeInfo struct {
	From       []string  `json:"from"`
	To         []string  `json:"to"`
	CC         []string  `json:"cc"`
	BCC        []string  `json:"bcc"`
	ReplyTo    []string  `json:"reply_to"`
	Subject    string    `json:"subject"`
	MessageID  string    `json:"message_id"`
	InReplyTo  string    `json:"in_reply_to"`
	References []string  `json:"references"`
	Date       time.Time `json:"date"`
}

// GetMessageEnvelopeInfo извлекает полную информацию из IMAP envelope
func GetMessageEnvelopeInfo(msg *imap.Message) *EnvelopeInfo {
	if msg.Envelope == nil {
		return nil
	}

	info := &EnvelopeInfo{
		MessageID: msg.Envelope.MessageId,
		Subject:   msg.Envelope.Subject,
		Date:      msg.Envelope.Date,
		InReplyTo: msg.Envelope.InReplyTo, // ✅ ПРАВИЛЬНО: это строка
	}

	// From
	info.From = extractAddresses(msg.Envelope.From)

	// To
	info.To = extractAddresses(msg.Envelope.To)

	// CC
	info.CC = extractAddresses(msg.Envelope.Cc)

	// BCC
	info.BCC = extractAddresses(msg.Envelope.Bcc)

	// ReplyTo
	info.ReplyTo = extractAddresses(msg.Envelope.ReplyTo)

	// ✅ References нужно парсить из заголовков тела сообщения
	// Пока оставляем пустым, добавим позже
	info.References = []string{}

	return info
}

// extractAddresses извлекает адреса из imap.Address
func extractAddresses(addresses []*imap.Address) []string {
	result := make([]string, len(addresses))
	for i, addr := range addresses {
		result[i] = formatAddress(addr)
	}
	return result
}

// formatAddress форматирует адрес в строку
func formatAddress(addr *imap.Address) string {
	if addr == nil {
		return ""
	}
	if addr.PersonalName != "" {
		return fmt.Sprintf("%s <%s@%s>", addr.PersonalName, addr.MailboxName, addr.HostName)
	}
	return fmt.Sprintf("%s@%s", addr.MailboxName, addr.HostName)
}

// CreateFetchItems создает набор полей для получения сообщений
func CreateFetchItems(includeBody bool) []imap.FetchItem {
	items := []imap.FetchItem{
		imap.FetchEnvelope,
		imap.FetchFlags,
		imap.FetchInternalDate,
		imap.FetchUid,
	}

	if includeBody {
		// Получаем полное сообщение для парсинга
		items = append(items, imap.FetchRFC822)
	} else {
		// Только заголовки
		items = append(items, imap.FetchRFC822Header)
	}

	return items
}

// CreateSearchCriteriaSince создает критерии поиска для сообщений после указанного UID
func CreateSearchCriteriaSince(lastUID uint32) *imap.SearchCriteria {
	criteria := imap.NewSearchCriteria()

	if lastUID > 0 {
		criteria.Uid = new(imap.SeqSet)
		criteria.Uid.AddNum(lastUID+1, 0) // 0 означает "*" - все последующие
	} else {
		// Первый запуск - получаем все сообщения
		criteria.Uid = new(imap.SeqSet)
		criteria.Uid.AddNum(1, 0)
	}

	return criteria
}

// GetMessageWithReferences извлекает информацию о письме включая References
func GetMessageWithReferences(msg *imap.Message) *EnvelopeInfo {
	info := GetMessageEnvelopeInfo(msg)
	if info == nil {
		return nil
	}

	// ✅ ПАРСИМ REFERENCES ИЗ ЗАГОЛОВКОВ
	info.References = parseReferencesFromHeaders(msg, info)

	return info
}

// parseReferencesFromHeaders парсит References из заголовков email
func parseReferencesFromHeaders(msg *imap.Message, info *EnvelopeInfo) []string {
	if len(msg.Body) == 0 {
		return nil
	}

	var references []string

	// Ищем References в каждом секторе тела
	for _, body := range msg.Body {
		if body == nil {
			continue
		}

		// ✅ ПРАВИЛЬНО: imap.Literal реализует io.Reader
		reader, ok := body.(io.Reader)
		if !ok {
			continue
		}

		// Читаем заголовки
		scanner := bufio.NewScanner(reader)
		for scanner.Scan() {
			line := scanner.Text()

			// Конец заголовков
			if line == "" {
				break
			}

			// Ищем References заголовок
			if strings.HasPrefix(strings.ToLower(line), "references:") {
				refs := strings.TrimSpace(strings.TrimPrefix(line, "References:"))
				// Разбиваем по пробелам (References могут быть через пробел)
				if refs != "" {
					references = strings.Fields(refs)
					break
				}
			}

			// ✅ ИСПРАВЛЕНО: используем переданный info
			// Также ищем In-Reply-To если его нет в Envelope
			if strings.HasPrefix(strings.ToLower(line), "in-reply-to:") && info.InReplyTo == "" {
				inReplyTo := strings.TrimSpace(strings.TrimPrefix(line, "In-Reply-To:"))
				if inReplyTo != "" {
					info.InReplyTo = inReplyTo
				}
			}
		}

		if len(references) > 0 {
			break
		}
	}

	return references
}
