// backend/internal/infrastructure/logging/test_logger.go
package logging

import (
	"context"
	"fmt"
)

// TestLogger простой logger для тестов
type TestLogger struct{}

func NewTestLogger() *TestLogger {
	return &TestLogger{}
}

func (l *TestLogger) Debug(ctx context.Context, msg string, fields ...interface{}) {
	fmt.Printf("[DEBUG] %s %v\n", msg, fields)
}

func (l *TestLogger) Info(ctx context.Context, msg string, fields ...interface{}) {
	fmt.Printf("[INFO] %s %v\n", msg, fields)
}

func (l *TestLogger) Warn(ctx context.Context, msg string, fields ...interface{}) {
	fmt.Printf("[WARN] %s %v\n", msg, fields)
}

func (l *TestLogger) Error(ctx context.Context, msg string, fields ...interface{}) {
	fmt.Printf("[ERROR] %s %v\n", msg, fields)
}

func (l *TestLogger) WithContext(ctx context.Context) context.Context {
	return ctx
}
