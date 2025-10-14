// backend/internal/email/imapclient/utils.go
package imapclient

import (
	"fmt"
	"time"

	"github.com/emersion/go-imap"
)

// EnvelopeInfo содержит базовую информацию о письме из IMAP envelope
type EnvelopeInfo struct {
	From      []string  `json:"from"`
	To        []string  `json:"to"`
	CC        []string  `json:"cc"`
	BCC       []string  `json:"bcc"`
	Subject   string    `json:"subject"`
	MessageID string    `json:"message_id"`
	InReplyTo string    `json:"in_reply_to"`
	Date      time.Time `json:"date"`
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

// CreateFetchItems создает набор полей для получения сообщений
func CreateFetchItems(includeBody bool) []imap.FetchItem {
	items := []imap.FetchItem{
		imap.FetchEnvelope,
		imap.FetchFlags,
		imap.FetchInternalDate,
		imap.FetchUid,
	}

	if includeBody {
		items = append(items, imap.FetchBody, imap.FetchBodyStructure)
	}

	return items
}

// GetMessageEnvelopeInfo извлекает базовую информацию из IMAP envelope
func GetMessageEnvelopeInfo(msg *imap.Message) *EnvelopeInfo {
	if msg.Envelope == nil {
		return nil
	}

	info := &EnvelopeInfo{
		MessageID: msg.Envelope.MessageId,
		Subject:   msg.Envelope.Subject,
		Date:      msg.Envelope.Date,
	}

	// From
	for _, addr := range msg.Envelope.From {
		info.From = append(info.From, formatAddress(addr))
	}

	// To
	for _, addr := range msg.Envelope.To {
		info.To = append(info.To, formatAddress(addr))
	}

	// CC
	for _, addr := range msg.Envelope.Cc {
		info.CC = append(info.CC, formatAddress(addr))
	}

	// BCC
	for _, addr := range msg.Envelope.Bcc {
		info.BCC = append(info.BCC, formatAddress(addr))
	}

	// In-Reply-To (берем первый если несколько и конвертируем []byte в string)
	if len(msg.Envelope.InReplyTo) > 0 {
		info.InReplyTo = string(msg.Envelope.InReplyTo[0])
	}

	return info
}

func formatAddress(addr *imap.Address) string {
	if addr == nil {
		return ""
	}
	if addr.PersonalName != "" {
		return fmt.Sprintf("%s <%s@%s>", addr.PersonalName, addr.MailboxName, addr.HostName)
	}
	return fmt.Sprintf("%s@%s", addr.MailboxName, addr.HostName)
}
