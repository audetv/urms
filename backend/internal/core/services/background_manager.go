// internal/core/services/background_manager.go
package services

import (
	"context"
	"sync"
	"time"

	"github.com/audetv/urms/internal/core/ports"
)

type BackgroundTaskManager struct {
	tasks      []ports.BackgroundTask
	wg         sync.WaitGroup
	mu         sync.RWMutex
	isRunning  bool
	logger     ports.Logger
	taskStatus map[string]string
}

func NewBackgroundTaskManager(logger ports.Logger) *BackgroundTaskManager {
	return &BackgroundTaskManager{
		tasks:      make([]ports.BackgroundTask, 0),
		logger:     logger,
		isRunning:  false,
		taskStatus: make(map[string]string),
	}
}

func (m *BackgroundTaskManager) RegisterTask(task ports.BackgroundTask) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.tasks = append(m.tasks, task)
	m.taskStatus[task.Name()] = "registered"
	m.logger.Info(context.Background(), "background task registered",
		"task_name", task.Name())
}

func (m *BackgroundTaskManager) StartAll(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.isRunning {
		m.logger.Info(ctx, "background tasks already running")
		return nil
	}

	// ✅ ВОЗВРАЩАЕМ НОРМАЛЬНОЕ ЛОГИРОВАНИЕ
	m.logger.Info(ctx, "starting all background tasks", "task_count", len(m.tasks))

	for _, task := range m.tasks {
		m.wg.Add(1)
		go func(t ports.BackgroundTask) {
			defer m.wg.Done()

			m.taskStatus[t.Name()] = "starting"
			if err := t.Start(ctx); err != nil {
				m.logger.Error(ctx, "failed to start background task",
					"task_name", t.Name(),
					"error", err.Error())
				m.taskStatus[t.Name()] = "error"
			} else {
				m.logger.Info(ctx, "background task started successfully",
					"task_name", t.Name())
				m.taskStatus[t.Name()] = "running"
			}
		}(task)
	}

	m.isRunning = true
	m.logger.Info(ctx, "all background tasks startup initiated")
	return nil
}

func (m *BackgroundTaskManager) StopAll(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.isRunning {
		m.logger.Info(ctx, "background tasks already stopped")
		return nil
	}

	m.logger.Info(ctx, "stopping all background tasks", "task_count", len(m.tasks))

	// Останавливаем задачи
	for _, task := range m.tasks {
		m.taskStatus[task.Name()] = "stopping"
		if err := task.Stop(ctx); err != nil {
			m.logger.Error(ctx, "failed to stop background task",
				"task_name", task.Name(),
				"error", err.Error())
			m.taskStatus[task.Name()] = "error"
		} else {
			m.taskStatus[task.Name()] = "stopped"
		}
	}

	// Ждем завершения всех горутин
	done := make(chan struct{})
	go func() {
		m.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		m.logger.Info(ctx, "all background tasks stopped successfully")
	case <-time.After(30 * time.Second):
		m.logger.Warn(ctx, "timeout waiting for background tasks to stop")
	}

	m.isRunning = false
	return nil
}

func (m *BackgroundTaskManager) GetTaskStatus(ctx context.Context) map[string]string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	status := make(map[string]string)
	for k, v := range m.taskStatus {
		status[k] = v
	}
	return status
}
