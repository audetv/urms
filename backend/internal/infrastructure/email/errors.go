// backend/internal/infrastructure/email/errors.go
package email

import "fmt"

// IMAPError представляет ошибку IMAP операций
type IMAPError struct {
	Operation string
	Code      string
	Message   string
	Details   interface{}
	Retryable bool
	Permanent bool
}

func NewIMAPError(operation, code, message string, details interface{}) *IMAPError {
	return &IMAPError{
		Operation: operation,
		Code:      code,
		Message:   message,
		Details:   details,
		Retryable: isIMAPRetryableError(code),
		Permanent: isIMAPPermanentError(code),
	}
}

func (e *IMAPError) Error() string {
	return fmt.Sprintf("IMAP %s failed [%s]: %s", e.Operation, e.Code, e.Message)
}

func (e *IMAPError) IsRetryable() bool {
	return e.Retryable
}

func (e *IMAPError) IsPermanent() bool {
	return e.Permanent
}

// IMAP коды ошибок
const (
	IMAPErrorConnection = "CONNECTION_ERROR"
	IMAPErrorAuth       = "AUTHENTICATION_ERROR"
	IMAPErrorTimeout    = "TIMEOUT_ERROR"
	IMAPErrorServer     = "SERVER_ERROR"
	IMAPErrorProtocol   = "PROTOCOL_ERROR"
	IMAPErrorQuota      = "QUOTA_EXCEEDED"
	IMAPErrorNotFound   = "MAILBOX_NOT_FOUND"
)

func isIMAPRetryableError(code string) bool {
	retryableCodes := map[string]bool{
		IMAPErrorConnection: true,
		IMAPErrorTimeout:    true,
		IMAPErrorServer:     true,
	}
	return retryableCodes[code]
}

func isIMAPPermanentError(code string) bool {
	permanentCodes := map[string]bool{
		IMAPErrorAuth:     true,
		IMAPErrorProtocol: true,
		IMAPErrorQuota:    true,
		IMAPErrorNotFound: true,
	}
	return permanentCodes[code]
}
