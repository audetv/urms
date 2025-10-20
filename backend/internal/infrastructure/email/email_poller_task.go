// internal/infrastructure/email/email_poller_task.go
package email

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/audetv/urms/internal/core/ports"
	"github.com/audetv/urms/internal/core/services"
)

type EmailPollerTask struct {
	emailService     *services.EmailService
	pollInterval     time.Duration
	operationTimeout time.Duration
	logger           ports.Logger
	cancelFunc       context.CancelFunc
	isRunning        bool
	mu               sync.RWMutex
}

func NewEmailPollerTask(
	emailService *services.EmailService,
	pollInterval time.Duration,
	operationTimeout time.Duration,
	logger ports.Logger,
) *EmailPollerTask {
	return &EmailPollerTask{
		emailService:     emailService,
		pollInterval:     pollInterval,
		operationTimeout: operationTimeout,
		logger:           logger,
		isRunning:        false,
	}
}

func (t *EmailPollerTask) Start(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.isRunning {
		return fmt.Errorf("email poller task already running")
	}

	// Создаем контекст для управления жизненным циклом
	taskCtx, cancel := context.WithCancel(ctx)
	t.cancelFunc = cancel
	t.isRunning = true

	// Запускаем горутину для polling
	go t.pollingLoop(taskCtx)

	t.logger.Info(ctx, "email poller task started",
		"poll_interval", t.pollInterval,
		"operation_timeout", t.operationTimeout)

	return nil
}

func (t *EmailPollerTask) Stop(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.isRunning {
		return nil
	}

	if t.cancelFunc != nil {
		t.cancelFunc()
	}

	t.isRunning = false
	t.logger.Info(ctx, "email poller task stopped")
	return nil
}

func (t *EmailPollerTask) Name() string {
	return "email_poller"
}

func (t *EmailPollerTask) Health(ctx context.Context) error {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if !t.isRunning {
		return fmt.Errorf("email poller task is not running")
	}
	return nil
}

// internal/infrastructure/email/email_poller_task.go
func (t *EmailPollerTask) pollingLoop(ctx context.Context) {
	ticker := time.NewTicker(t.pollInterval)
	defer ticker.Stop()

	// ✅ ВОЗВРАЩАЕМ НОРМАЛЬНОЕ ЛОГИРОВАНИЕ (без принудительных сообщений)
	t.logger.Info(ctx, "email poller polling loop started",
		"interval", t.pollInterval)

	for {
		select {
		case <-ctx.Done():
			t.logger.Info(ctx, "email poller polling loop stopped")
			return
		case <-ticker.C:
			t.executePoll(ctx) // ✅ Без лишних логов
		}
	}
}

func (t *EmailPollerTask) executePoll(ctx context.Context) {
	pollCtx := context.WithValue(ctx, ports.CorrelationIDKey, "email-poller-"+generateShortID())

	// ✅ НОРМАЛЬНЫЕ ЛОГИ (без принудительных)
	t.logger.Info(pollCtx, "email poller running scheduled check")

	// Создаем контекст с таймаутом операции
	timeoutCtx, cancel := context.WithTimeout(pollCtx, t.operationTimeout)
	defer cancel()

	startTime := time.Now()
	if err := t.emailService.ProcessIncomingEmails(timeoutCtx); err != nil {
		t.logger.Error(pollCtx, "email poller error", "error", err)
	} else {
		duration := time.Since(startTime)
		t.logger.Info(pollCtx, "email poller completed successfully", "duration", duration)
	}
}
