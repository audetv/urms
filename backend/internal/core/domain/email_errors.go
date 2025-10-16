// backend/internal/core/domain/email_errors.go
package domain

import (
	"errors"
	"fmt"
)

// Email Domain Specific Errors
var (
	ErrEmailNotFound         = errors.New("email message not found")
	ErrEmailSendFailed       = errors.New("failed to send email")
	ErrEmailFetchFailed      = errors.New("failed to fetch emails")
	ErrConnectionFailed      = errors.New("email connection failed")
	ErrAuthenticationFailed  = errors.New("email authentication failed")
	ErrMailboxNotFound       = errors.New("mailbox not found")
	ErrEmailAlreadyProcessed = errors.New("email already processed")
	ErrInvalidEmailAddress   = errors.New("invalid email address")
	ErrEmptySubject          = errors.New("email subject cannot be empty")
	ErrMessageTooLarge       = errors.New("email message size exceeds limit")
)

// DomainError с кодом для лучшей обработки
type DomainError struct {
	Err       error
	Message   string
	Code      string
	Domain    string // "email", "ticket", "customer"
	Retryable bool
}

func (e DomainError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Domain, e.Message, e.Err)
	}
	return fmt.Sprintf("[%s] %s", e.Domain, e.Message)
}

func (e DomainError) IsRetryable() bool {
	return e.Retryable
}

func NewEmailDomainError(message, code string, err error) DomainError {
	return DomainError{
		Message:   message,
		Code:      code,
		Err:       err,
		Domain:    "email",
		Retryable: isRetryableError(code),
	}
}

// isRetryableError определяет, является ли код ошибки временной
func isRetryableError(code string) bool {
	retryableCodes := map[string]bool{
		"NETWORK_ERROR":       true,
		"TIMEOUT_ERROR":       true,
		"SERVER_UNAVAILABLE":  true,
		"RATE_LIMIT_EXCEEDED": true,
		"CONNECTION_ERROR":    true,
	}
	return retryableCodes[code]
}
