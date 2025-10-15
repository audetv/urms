package email

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/audetv/urms/internal/core/domain"
	"github.com/audetv/urms/internal/core/ports"
	"github.com/rs/zerolog/log"
)

// IMAPPoller реализует автоматический опрос IMAP почтовых ящиков
type IMAPPoller struct {
	gateway   ports.EmailGateway
	repo      ports.EmailRepository
	processor ports.MessageProcessor
	config    *PollerConfig
	state     *PollerState
	mu        sync.RWMutex
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
}

// PollerConfig конфигурация IMAP Poller
type PollerConfig struct {
	Mailbox           string
	PollInterval      time.Duration
	BatchSize         int
	ReadOnly          bool
	MaxRetries        int
	RetryDelay        time.Duration
	ReconnectTimeout  time.Duration
	HealthCheckPeriod time.Duration
}

// PollerState состояние poller'а
type PollerState struct {
	LastUID        uint32
	LastPollTime   time.Time
	LastSuccessUID uint32
	IsRunning      bool
	ErrorCount     int
	TotalPolls     int64
	TotalMessages  int64
}

// NewIMAPPoller создает новый IMAP Poller
func NewIMAPPoller(
	gateway ports.EmailGateway,
	repo ports.EmailRepository,
	processor ports.MessageProcessor,
	config *PollerConfig,
) *IMAPPoller {
	if config.PollInterval == 0 {
		config.PollInterval = 30 * time.Second
	}
	if config.BatchSize == 0 {
		config.BatchSize = 100
	}
	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}
	if config.RetryDelay == 0 {
		config.RetryDelay = 10 * time.Second
	}

	return &IMAPPoller{
		gateway:   gateway,
		repo:      repo,
		processor: processor,
		config:    config,
		state: &PollerState{
			LastUID:      0,
			LastPollTime: time.Now().Add(-24 * time.Hour), // Начинаем с сообщений за последние 24 часа
		},
	}
}

// Start запускает фоновый опрос почтового ящика
func (p *IMAPPoller) Start(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.state.IsRunning {
		return fmt.Errorf("poller is already running")
	}

	// Создаем контекст с отменой
	p.ctx, p.cancel = context.WithCancel(ctx)
	p.state.IsRunning = true

	// Загружаем последнее состояние
	if err := p.loadState(); err != nil {
		log.Warn().Err(err).Msg("Failed to load poller state, starting fresh")
	}

	// Запускаем горутину для опроса
	p.wg.Add(1)
	go p.pollLoop()

	// Запускаем health check горутину
	if p.config.HealthCheckPeriod > 0 {
		p.wg.Add(1)
		go p.healthCheckLoop()
	}

	log.Info().
		Str("mailbox", p.config.Mailbox).
		Dur("interval", p.config.PollInterval).
		Uint32("last_uid", p.state.LastUID).
		Msg("IMAP poller started")

	return nil
}

// Stop останавливает опрос
func (p *IMAPPoller) Stop() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.state.IsRunning {
		return nil
	}

	log.Info().Msg("Stopping IMAP poller...")

	// Отменяем контекст
	if p.cancel != nil {
		p.cancel()
	}

	// Сохраняем состояние
	if err := p.saveState(); err != nil {
		log.Error().Err(err).Msg("Failed to save poller state")
	}

	p.state.IsRunning = false

	// Ждем завершения горутин
	p.wg.Wait()

	log.Info().Msg("IMAP poller stopped successfully")
	return nil
}

// pollLoop основной цикл опроса
func (p *IMAPPoller) pollLoop() {
	defer p.wg.Done()

	ticker := time.NewTicker(p.config.PollInterval)
	defer ticker.Stop()

	// Выполняем немедленный первый опрос
	if err := p.pollNewMessages(p.ctx); err != nil {
		log.Error().Err(err).Msg("Initial poll failed")
	}

	for {
		select {
		case <-p.ctx.Done():
			return
		case <-ticker.C:
			if err := p.pollNewMessages(p.ctx); err != nil {
				log.Error().Err(err).Msg("Poll failed")
				p.state.ErrorCount++
			} else {
				p.state.ErrorCount = 0 // Сбрасываем счетчик ошибок при успехе
			}
		}
	}
}

// healthCheckLoop периодическая проверка здоровья соединения
func (p *IMAPPoller) healthCheckLoop() {
	defer p.wg.Done()

	if p.config.HealthCheckPeriod == 0 {
		return
	}

	ticker := time.NewTicker(p.config.HealthCheckPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-p.ctx.Done():
			return
		case <-ticker.C:
			if err := p.gateway.HealthCheck(p.ctx); err != nil {
				log.Warn().Err(err).Msg("Health check failed")
			}
		}
	}
}

// pollNewMessages опрашивает новые сообщения
func (p *IMAPPoller) pollNewMessages(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.state.TotalPolls++

	log.Debug().
		Uint32("since_uid", p.state.LastUID).
		Str("mailbox", p.config.Mailbox).
		Msg("Polling for new messages")

	// Выбираем почтовый ящик
	if err := p.gateway.SelectMailbox(ctx, p.config.Mailbox); err != nil {
		return fmt.Errorf("failed to select mailbox: %w", err)
	}

	// Критерии поиска новых сообщений
	criteria := ports.FetchCriteria{
		SinceUID:   p.state.LastUID,
		Mailbox:    p.config.Mailbox,
		Limit:      p.config.BatchSize,
		UnseenOnly: true,
	}

	// Получаем сообщения
	messages, err := p.gateway.FetchMessages(ctx, criteria)
	if err != nil {
		return fmt.Errorf("failed to fetch messages: %w", err)
	}

	if len(messages) == 0 {
		log.Debug().Msg("No new messages found")
		return nil
	}

	log.Info().
		Int("count", len(messages)).
		Msg("Found new messages")

	// Обрабатываем пачку сообщений
	if err := p.processMessageBatch(ctx, messages); err != nil {
		return fmt.Errorf("failed to process message batch: %w", err)
	}

	// Обновляем последний UID
	if len(messages) > 0 {
		// Предполагаем, что сообщения приходят в порядке возрастания UID
		// В реальной реализации нужно извлекать UID из IMAP сообщений
		p.state.LastUID = p.extractLastUID(messages)
		p.state.LastPollTime = time.Now()
		p.state.TotalMessages += int64(len(messages))
	}

	// Сохраняем состояние
	if err := p.saveState(); err != nil {
		log.Warn().Err(err).Msg("Failed to save poller state")
	}

	log.Info().
		Int("processed", len(messages)).
		Uint32("last_uid", p.state.LastUID).
		Msg("Message batch processed successfully")

	return nil
}

// processMessageBatch обрабатывает пачку сообщений
func (p *IMAPPoller) processMessageBatch(ctx context.Context, messages []domain.EmailMessage) error {
	for i, msg := range messages {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Обрабатываем каждое сообщение
			if err := p.processSingleMessage(ctx, msg); err != nil {
				log.Error().
					Err(err).
					Str("message_id", msg.MessageID).
					Int("index", i).
					Msg("Failed to process message")
				// Продолжаем обработку остальных сообщений
				continue
			}
		}
	}
	return nil
}

// processSingleMessage обрабатывает одно сообщение
func (p *IMAPPoller) processSingleMessage(ctx context.Context, msg domain.EmailMessage) error {
	// Сохраняем в репозиторий
	if err := p.repo.Save(ctx, &msg); err != nil {
		return fmt.Errorf("failed to save message: %w", err)
	}

	// Передаем процессору если он есть
	if p.processor != nil {
		if err := p.processor.ProcessIncomingEmail(ctx, msg); err != nil {
			return fmt.Errorf("failed to process message: %w", err)
		}
	}

	// Помечаем как прочитанное если не в read-only режиме
	if !p.config.ReadOnly {
		if err := p.gateway.MarkAsRead(ctx, []string{msg.MessageID}); err != nil {
			log.Warn().
				Err(err).
				Str("message_id", msg.MessageID).
				Msg("Failed to mark message as read")
		}
	}

	return nil
}

// extractLastUID извлекает максимальный UID из пачки сообщений
func (p *IMAPPoller) extractLastUID(messages []domain.EmailMessage) uint32 {
	// Временная реализация - возвращаем увеличенный LastUID
	// В Phase 1B.2 добавим реальное извлечение UID из IMAP сообщений
	return p.state.LastUID + uint32(len(messages))
}

// loadState загружает состояние poller'а (заглушка)
func (p *IMAPPoller) loadState() error {
	// TODO: Реализовать загрузку состояния из persistent storage
	// Пока используем состояние в памяти
	return nil
}

// saveState сохраняет состояние poller'а (заглушка)
func (p *IMAPPoller) saveState() error {
	// TODO: Реализовать сохранение состояния в persistent storage
	return nil
}

// GetState возвращает текущее состояние poller'а
func (p *IMAPPoller) GetState() PollerState {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return *p.state
}

// GetStats возвращает статистику poller'а
func (p *IMAPPoller) GetStats() map[string]interface{} {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return map[string]interface{}{
		"is_running":     p.state.IsRunning,
		"last_uid":       p.state.LastUID,
		"last_poll_time": p.state.LastPollTime,
		"total_polls":    p.state.TotalPolls,
		"total_messages": p.state.TotalMessages,
		"error_count":    p.state.ErrorCount,
		"mailbox":        p.config.Mailbox,
		"poll_interval":  p.config.PollInterval.String(),
	}
}
