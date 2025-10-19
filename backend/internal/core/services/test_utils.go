// internal/core/services/test_utils.go
package services

import "context"

// MockLogger для тестирования
type MockLogger struct{}

func (m *MockLogger) Debug(ctx context.Context, msg string, fields ...interface{}) {}
func (m *MockLogger) Info(ctx context.Context, msg string, fields ...interface{})  {}
func (m *MockLogger) Warn(ctx context.Context, msg string, fields ...interface{})  {}
func (m *MockLogger) Error(ctx context.Context, msg string, fields ...interface{}) {}
func (m *MockLogger) WithContext(ctx context.Context) context.Context              { return ctx }
