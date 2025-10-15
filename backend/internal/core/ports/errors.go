package ports

import "fmt"

// EmailError ошибка email модуля
type EmailError struct {
	Message   string
	Code      string
	Details   interface{}
	Retryable bool
}

// NewEmailError создает новую EmailError
func NewEmailError(message, code string, details interface{}) *EmailError {
	return &EmailError{
		Message:   message,
		Code:      code,
		Details:   details,
		Retryable: isRetryableError(code),
	}
}

func (e *EmailError) Error() string {
	if e.Details != nil {
		return fmt.Sprintf("%s [%s]: %v", e.Message, e.Code, e.Details)
	}
	return fmt.Sprintf("%s [%s]", e.Message, e.Code)
}

// IsRetryable проверяет, является ли ошибка временной
func (e *EmailError) IsRetryable() bool {
	return e.Retryable
}

// isRetryableError определяет, является ли код ошибки временной
func isRetryableError(code string) bool {
	retryableCodes := map[string]bool{
		"CONNECTION_ERROR":    true,
		"NETWORK_ERROR":       true,
		"TIMEOUT_ERROR":       true,
		"SERVER_UNAVAILABLE":  true,
		"RATE_LIMIT_EXCEEDED": true,
	}
	return retryableCodes[code]
}
