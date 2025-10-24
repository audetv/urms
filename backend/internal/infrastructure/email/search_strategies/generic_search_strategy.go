// internal/infrastructure/email/search_strategies/generic_search_strategy.go
package search_strategies

import (
	"context"
	"strings"
	"time"

	"github.com/audetv/urms/internal/core/domain"
	"github.com/audetv/urms/internal/core/ports"
	"github.com/emersion/go-imap"
)

// GenericSearchStrategy реализует ports.SearchStrategy для универсальных провайдеров
// Балансирует между производительностью и надежностью
type GenericSearchStrategy struct {
	config *domain.EmailProviderConfig
	logger ports.Logger
}

// NewGenericSearchStrategy создает новую универсальную поисковую стратегию
func NewGenericSearchStrategy(
	config *domain.EmailProviderConfig,
	logger ports.Logger,
) *GenericSearchStrategy {
	// Валидируем конфигурацию при создании
	if err := config.Validate(); err != nil {
		logger.Warn(context.Background(),
			"Generic search strategy configuration validation warning",
			"error", err.Error())
	}

	return &GenericSearchStrategy{
		config: config,
		logger: logger,
	}
}

// CreateThreadSearchCriteria создает сбалансированные IMAP критерии поиска
func (s *GenericSearchStrategy) CreateThreadSearchCriteria(threadData ports.ThreadSearchCriteria) (*imap.SearchCriteria, error) {
	ctx := context.Background()
	criteria := &imap.SearchCriteria{}

	// GENERIC-OPTIMIZED: Сбалансированный набор критериев
	messageIDs := s.collectMessageIDs(threadData)

	if len(messageIDs) > 0 {
		// Для универсальных провайдеров используем только Message-ID для надежности
		criteria.Header = map[string][]string{
			"Message-ID": messageIDs,
		}

		s.logger.Debug(ctx, "Generic search criteria created",
			"message_ids_count", len(messageIDs),
			"max_allowed", s.GetMaxMessageIDs(),
			"strategy", "balanced_thread_search")
	}

	// CONFIGURATION-DRIVEN: Временной диапазон из конфигурации
	timeframeDays := s.GetTimeframeDays()
	criteria.Since = time.Now().Add(-time.Duration(timeframeDays) * 24 * time.Hour)

	s.logger.Debug(ctx, "Generic search timeframe configured",
		"timeframe_days", timeframeDays,
		"since_date", criteria.Since.Format("2006-01-02"),
		"provider_compatibility", "moderate_timeframe_recommended")

	return criteria, nil
}

// GetComplexity возвращает уровень сложности поиска из конфигурации
func (s *GenericSearchStrategy) GetComplexity() domain.SearchComplexity {
	complexity := s.config.GetSearchComplexity()

	// Для универсальных провайдеров рекомендуем Moderate complexity
	if complexity == domain.SearchComplexityComplex {
		s.logger.Info(context.Background(),
			"Generic provider complexity adjustment recommendation",
			"configured_complexity", complexity.String(),
			"recommended_complexity", domain.SearchComplexityModerate.String(),
			"reason", "better_compatibility_with_generic_providers")
	}

	return complexity
}

// GetMaxMessageIDs возвращает максимальное количество Message-ID для поиска
func (s *GenericSearchStrategy) GetMaxMessageIDs() int {
	configuredMax := s.config.GetMaxMessageIDs()

	// Для универсальных провайдеров ограничиваем разумным максимумом
	if configuredMax > 10 {
		s.logger.Warn(context.Background(),
			"Generic provider message ID limit recommendation",
			"configured_max", configuredMax,
			"recommended_max", 10,
			"reason", "performance_and_compatibility_considerations")
		return 10
	}

	return configuredMax
}

// GetTimeframeDays возвращает временной диапазон поиска в днях
func (s *GenericSearchStrategy) GetTimeframeDays() int {
	configuredDays := s.config.GetTimeframeDays()

	// Для универсальных провайдеров ограничиваем разумным максимумом
	if configuredDays > 365 {
		s.logger.Warn(context.Background(),
			"Generic provider timeframe limit recommendation",
			"configured_days", configuredDays,
			"recommended_max", 365,
			"reason", "performance_considerations_for_generic_providers")
		return 365
	}

	return configuredDays
}

// collectMessageIDs собирает Message-ID в пределах лимитов конфигурации
func (s *GenericSearchStrategy) collectMessageIDs(threadData ports.ThreadSearchCriteria) []string {
	maxMessageIDs := s.GetMaxMessageIDs()
	var messageIDs []string

	// Приоритет 1: Основной Message-ID
	if threadData.MessageID != "" {
		messageIDs = append(messageIDs, threadData.MessageID)
	}

	// Приоритет 2: In-Reply-To (если есть место)
	if threadData.InReplyTo != "" && len(messageIDs) < maxMessageIDs {
		messageIDs = append(messageIDs, threadData.InReplyTo)
	}

	// Приоритет 3: References (ограниченное количество, берем самые свежие)
	if len(threadData.References) > 0 && len(messageIDs) < maxMessageIDs {
		remainingSlots := maxMessageIDs - len(messageIDs)
		if remainingSlots > 0 {
			// Берем последние References (предполагаем, что они упорядочены от старых к новым)
			startIdx := max(0, len(threadData.References)-remainingSlots)
			messageIDs = append(messageIDs, threadData.References[startIdx:]...)
		}
	}

	return s.normalizeMessageIDs(messageIDs)
}

// normalizeMessageIDs нормализует и удаляет дубликаты Message-ID
func (s *GenericSearchStrategy) normalizeMessageIDs(ids []string) []string {
	seen := make(map[string]bool)
	var result []string

	for _, id := range ids {
		normalizedID := strings.Trim(id, "<>")
		normalizedID = strings.TrimSpace(normalizedID)

		if normalizedID != "" && !seen[normalizedID] {
			seen[normalizedID] = true
			result = append(result, normalizedID)
		}
	}

	return result
}
