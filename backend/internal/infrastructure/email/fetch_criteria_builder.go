// internal/infrastructure/email/fetch_criteria_builder.go
package email

import (
	"context"
	"time"

	"github.com/audetv/urms/internal/core/ports"
)

// FetchCriteriaBuilder создает конфигурируемые критерии поиска
type FetchCriteriaBuilder struct {
	searchFactory ports.SearchStrategyFactory
	fetcher       ports.EmailFetcher
	logger        ports.Logger
}

// NewFetchCriteriaBuilder создает новый builder
func NewFetchCriteriaBuilder(
	searchFactory ports.SearchStrategyFactory,
	fetcher ports.EmailFetcher,
	logger ports.Logger,
) *FetchCriteriaBuilder {
	return &FetchCriteriaBuilder{
		searchFactory: searchFactory,
		fetcher:       fetcher,
		logger:        logger,
	}
}

// BuildStandardCriteria создает стандартные критерии из конфигурации
func (b *FetchCriteriaBuilder) BuildStandardCriteria(ctx context.Context) ports.FetchCriteria {
	providerType := b.fetcher.GetProviderType()

	// Получаем search strategy для конфигурации
	searchStrategy, err := b.searchFactory.GetSearchStrategy(ctx, providerType)
	if err != nil {
		b.logger.Warn(ctx,
			"Failed to get search strategy for criteria, using fallback",
			"provider", providerType,
			"error", err.Error())
		return b.buildFallbackCriteria()
	}

	// CONFIGURATION-DRIVEN параметры
	timeframeDays := searchStrategy.GetTimeframeDays()
	sinceTime := time.Now().Add(-time.Duration(timeframeDays) * 24 * time.Hour)

	b.logger.Debug(ctx, "Fetch criteria built from configuration",
		"provider", providerType,
		"timeframe_days", timeframeDays,
		"since_date", sinceTime.Format("2006-01-02"),
		"strategy_complexity", searchStrategy.GetComplexity().String())

	return ports.FetchCriteria{
		Since:      sinceTime,
		Mailbox:    "INBOX",
		Limit:      b.getBatchSizeFromStrategy(ctx),
		UnseenOnly: true,
	}
}

// BuildThreadSearchCriteria создает критерии для thread search
func (b *FetchCriteriaBuilder) BuildThreadSearchCriteria(
	ctx context.Context,
	threadData ports.ThreadSearchCriteria,
) ports.FetchCriteria {

	providerType := b.fetcher.GetProviderType()
	searchStrategy, err := b.searchFactory.GetSearchStrategy(ctx, providerType)
	if err != nil {
		b.logger.Warn(ctx,
			"Failed to get search strategy for thread criteria, using fallback",
			"provider", providerType,
			"error", err.Error())
		return b.buildFallbackCriteria()
	}

	// Thread-specific критерии с extended timeframe
	extendedTimeframe := b.getExtendedTimeframe(searchStrategy.GetTimeframeDays())
	sinceTime := time.Now().Add(-time.Duration(extendedTimeframe) * 24 * time.Hour)

	return ports.FetchCriteria{
		Since:      sinceTime,
		Mailbox:    threadData.Mailbox,
		Limit:      b.getExtendedBatchSize(ctx),
		UnseenOnly: false, // Для thread search включаем прочитанные
		Subject:    threadData.Subject,
	}
}

// buildFallbackCriteria создает fallback критерии
func (b *FetchCriteriaBuilder) buildFallbackCriteria() ports.FetchCriteria {
	return ports.FetchCriteria{
		Since:      time.Now().Add(-24 * time.Hour),
		Mailbox:    "INBOX",
		Limit:      50,
		UnseenOnly: true,
	}
}

// getBatchSizeFromStrategy получает batch size из стратегии
func (b *FetchCriteriaBuilder) getBatchSizeFromStrategy(ctx context.Context) int {
	// Можно получить из pipeline strategy или использовать разумный дефолт
	// Пока используем консервативный дефолт
	return 50
}

// getExtendedBatchSize получает увеличенный batch size для thread search
func (b *FetchCriteriaBuilder) getExtendedBatchSize(ctx context.Context) int {
	// Для thread search можно использовать большие батчи
	return 100
}

// getExtendedTimeframe расширяет timeframe для thread search
func (b *FetchCriteriaBuilder) getExtendedTimeframe(baseTimeframe int) int {
	// Увеличиваем timeframe на 50% для thread search
	return baseTimeframe + (baseTimeframe / 2)
}
