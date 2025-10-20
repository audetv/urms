// internal/core/ports/background.go
package ports

import "context"

// BackgroundTask определяет контракт для фоновых задач
type BackgroundTask interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Name() string
	Health(ctx context.Context) error
}

// BackgroundManager управляет всеми фоновыми задачами
type BackgroundManager interface {
	RegisterTask(task BackgroundTask)
	StartAll(ctx context.Context) error
	StopAll(ctx context.Context) error
	GetTaskStatus(ctx context.Context) map[string]string
}
