// internal/infrastructure/email/email_fetcher_impl.go
package email

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/audetv/urms/internal/core/domain"
	"github.com/audetv/urms/internal/core/ports"
)

// EmailFetcherImpl реализует ports.EmailFetcher
type EmailFetcherImpl struct {
	gateway         ports.EmailGateway
	searchFactory   ports.SearchStrategyFactory
	criteriaBuilder *FetchCriteriaBuilder
	logger          ports.Logger
	progress        *FetchProgressTracker
	providerType    string
	mu              sync.RWMutex
}

// FetchProgressTracker отслеживает прогресс выборки
type FetchProgressTracker struct {
	totalMessages      int
	fetchedCount       int
	lastFetchTime      time.Time
	currentBatch       int
	estimatedRemaining time.Duration
	status             string
	mu                 sync.RWMutex
}

// NewEmailFetcherImpl создает новый экземпляр Email Fetcher
func NewEmailFetcherImpl(
	gateway ports.EmailGateway,
	searchFactory ports.SearchStrategyFactory,
	criteriaBuilder *FetchCriteriaBuilder,
	logger ports.Logger,
	providerType string,
) *EmailFetcherImpl {

	return &EmailFetcherImpl{
		gateway:         gateway,
		searchFactory:   searchFactory,
		criteriaBuilder: criteriaBuilder,
		logger:          logger,
		progress:        &FetchProgressTracker{status: "initialized"},
		providerType:    providerType,
	}
}

// FetchBatch получает батч сообщений по критериям
func (f *EmailFetcherImpl) FetchBatch(ctx context.Context, criteria ports.FetchCriteria) ([]domain.EmailMessage, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.logger.Info(ctx, "🔍 Starting email batch fetch",
		"provider", f.providerType,
		"since", criteria.Since.Format("2006-01-02"),
		"limit", criteria.Limit,
		"unseen_only", criteria.UnseenOnly)

	startTime := time.Now()
	f.progress.setStatus("fetching")

	// Обновляем прогресс
	f.progress.startNewBatch()

	// Получаем сообщения через gateway
	messages, err := f.gateway.FetchMessages(ctx, criteria)
	if err != nil {
		f.progress.setStatus("failed")
		f.logger.Error(ctx, "Failed to fetch email batch",
			"provider", f.providerType,
			"error", err.Error())
		return nil, fmt.Errorf("failed to fetch email batch: %w", err)
	}

	// Обновляем прогресс
	f.progress.recordBatch(len(messages), time.Since(startTime))
	f.progress.setStatus("completed")

	f.logger.Info(ctx, "✅ Email batch fetch completed",
		"provider", f.providerType,
		"message_count", len(messages),
		"fetch_duration", time.Since(startTime).String(),
		"first_subject", f.getFirstSubjectPreview(messages),
		"last_subject", f.getLastSubjectPreview(messages))

	return messages, nil
}

// FetchThreadMessages получает сообщения цепочки по threading данным
func (f *EmailFetcherImpl) FetchThreadMessages(
	ctx context.Context,
	threadData ports.ThreadSearchCriteria,
) ([]domain.EmailMessage, error) {

	f.mu.Lock()
	defer f.mu.Unlock()

	f.logger.Info(ctx, "🧵 Fetching thread messages",
		"provider", f.providerType,
		"message_id", threadData.MessageID,
		"in_reply_to", threadData.InReplyTo,
		"references_count", len(threadData.References),
		"subject", threadData.Subject)

	startTime := time.Now()
	f.progress.setStatus("thread_fetching")

	// Получаем search strategy для thread search
	searchStrategy, err := f.searchFactory.GetSearchStrategy(ctx, f.providerType)
	if err != nil {
		f.logger.Error(ctx, "Failed to get search strategy for thread fetch",
			"provider", f.providerType,
			"error", err.Error())
		return nil, fmt.Errorf("failed to get search strategy: %w", err)
	}

	// Создаем IMAP критерии через стратегию
	imapCriteria, err := searchStrategy.CreateThreadSearchCriteria(threadData)
	if err != nil {
		f.logger.Error(ctx, "Failed to create thread search criteria",
			"provider", f.providerType,
			"error", err.Error())
		return nil, fmt.Errorf("failed to create thread search criteria: %w", err)
	}

	f.logger.Debug(ctx, "Thread search criteria created",
		"provider", f.providerType,
		"criteria_since", imapCriteria.Since.Format("2006-01-02"),
		"headers_count", len(imapCriteria.Header))

	// Выполняем thread-aware поиск через gateway
	threadMessages, err := f.gateway.SearchThreadMessages(ctx, threadData)
	if err != nil {
		f.progress.setStatus("thread_fetch_failed")
		f.logger.Error(ctx, "Thread search failed",
			"provider", f.providerType,
			"message_id", threadData.MessageID,
			"error", err.Error())
		return nil, fmt.Errorf("thread search failed: %w", err)
	}

	f.progress.recordBatch(len(threadMessages), time.Since(startTime))
	f.progress.setStatus("thread_fetch_completed")

	f.logger.Info(ctx, "✅ Thread messages fetch completed",
		"provider", f.providerType,
		"thread_message_count", len(threadMessages),
		"fetch_duration", time.Since(startTime).String(),
		"original_message_id", threadData.MessageID)

	return threadMessages, nil
}

// GetProviderType возвращает тип провайдера
func (f *EmailFetcherImpl) GetProviderType() string {
	return f.providerType
}

// GetProgress возвращает прогресс операции выборки
func (f *EmailFetcherImpl) GetProgress(ctx context.Context) *ports.FetchProgress {
	f.progress.mu.RLock()
	defer f.progress.mu.RUnlock()

	return &ports.FetchProgress{
		TotalMessages:      f.progress.totalMessages,
		FetchedCount:       f.progress.fetchedCount,
		LastFetchTime:      f.progress.lastFetchTime,
		CurrentBatch:       f.progress.currentBatch,
		EstimatedRemaining: f.progress.estimatedRemaining,
		Status:             f.progress.status,
	}
}

// Health проверяет здоровье fetcher
func (f *EmailFetcherImpl) Health(ctx context.Context) error {
	// Проверяем соединение с gateway
	if err := f.gateway.HealthCheck(ctx); err != nil {
		return fmt.Errorf("email gateway health check failed: %w", err)
	}

	// Проверяем search factory
	if err := f.searchFactory.Health(ctx); err != nil {
		return fmt.Errorf("search factory health check failed: %w", err)
	}

	f.logger.Debug(ctx, "Email fetcher health check passed",
		"provider", f.providerType)

	return nil
}

// getFirstSubjectPreview возвращает preview первого subject
func (f *EmailFetcherImpl) getFirstSubjectPreview(messages []domain.EmailMessage) string {
	if len(messages) == 0 {
		return "no_messages"
	}
	return f.truncateSubject(messages[0].Subject)
}

// getLastSubjectPreview возвращает preview последнего subject
func (f *EmailFetcherImpl) getLastSubjectPreview(messages []domain.EmailMessage) string {
	if len(messages) == 0 {
		return "no_messages"
	}
	return f.truncateSubject(messages[len(messages)-1].Subject)
}

// truncateSubject обрезает subject для preview
func (f *EmailFetcherImpl) truncateSubject(subject string) string {
	if len(subject) <= 30 {
		return subject
	}
	return subject[:27] + "..."
}

// Методы для FetchProgressTracker
func (p *FetchProgressTracker) startNewBatch() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.currentBatch++
	p.status = "fetching"
}

func (p *FetchProgressTracker) recordBatch(messageCount int, duration time.Duration) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.fetchedCount += messageCount
	p.totalMessages += messageCount
	p.lastFetchTime = time.Now()

	// Простая оценка оставшегося времени
	if messageCount > 0 && duration > 0 {
		avgTimePerMessage := duration / time.Duration(messageCount)
		estimatedRemaining := avgTimePerMessage * time.Duration(p.totalMessages-p.fetchedCount)
		p.estimatedRemaining = estimatedRemaining
	}
}

func (p *FetchProgressTracker) setStatus(status string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.status = status
}
