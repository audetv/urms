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
)

// DomainError с кодом для лучшей обработки
type DomainError struct {
	Err     error
	Message string
	Code    string
	Domain  string // "email", "ticket", "customer"
}

func (e DomainError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Domain, e.Message, e.Err)
	}
	return fmt.Sprintf("[%s] %s", e.Domain, e.Message)
}

func NewEmailDomainError(message, code string, err error) DomainError {
	return DomainError{
		Message: message,
		Code:    code,
		Err:     err,
		Domain:  "email",
	}
}
