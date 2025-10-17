// backend/internal/core/services/console_logger.go
package services

import "context"

// ConsoleLogger - заглушка логгера для разработки
type ConsoleLogger struct{}

func (l *ConsoleLogger) Debug(ctx context.Context, msg string, fields ...interface{}) {}
func (l *ConsoleLogger) Info(ctx context.Context, msg string, fields ...interface{})  {}
func (l *ConsoleLogger) Warn(ctx context.Context, msg string, fields ...interface{})  {}
func (l *ConsoleLogger) Error(ctx context.Context, msg string, fields ...interface{}) {}
