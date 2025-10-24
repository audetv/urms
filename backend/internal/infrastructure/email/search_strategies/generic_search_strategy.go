// internal/infrastructure/email/search_strategies/generic_search_strategy.go
package search_strategies

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/audetv/urms/internal/core/domain"
	"github.com/audetv/urms/internal/core/ports"
	"github.com/emersion/go-imap"
)

// GenericSearchStrategy реализует ports.SearchStrategy для универсальных провайдеров
type GenericSearchStrategy struct {
	config *domain.SearchStrategyConfig // ✅ ПРАВИЛЬНЫЙ ТИП
	logger ports.Logger
}

// Configure настраивает стратегию с конфигурацией
func (s *GenericSearchStrategy) Configure(config *domain.SearchStrategyConfig) error {
	if config == nil {
		return fmt.Errorf("search strategy configuration is required")
	}

	if err := config.Validate(); err != nil {
		return fmt.Errorf("invalid search strategy configuration: %w", err)
	}

	s.config = config
	s.logger.Info(context.Background(),
		"Generic search strategy configured",
		"complexity", s.GetComplexity().String(),
		"max_message_ids", s.GetMaxMessageIDs(),
		"timeframe_days", s.GetTimeframeDays())

	return nil
}

// CreateThreadSearchCriteria создает сбалансированные IMAP критерии поиска
func (s *GenericSearchStrategy) CreateThreadSearchCriteria(threadData ports.ThreadSearchCriteria) (*imap.SearchCriteria, error) {
	ctx := context.Background()
	criteria := &imap.SearchCriteria{}

	if s.config == nil {
		return nil, fmt.Errorf("search strategy not configured")
	}

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
		"since_date", criteria.Since.Format("2006-01-02"))

	return criteria, nil
}

// GetComplexity возвращает уровень сложности поиска из конфигурации
func (s *GenericSearchStrategy) GetComplexity() domain.SearchComplexity {
	if s.config == nil {
		s.logger.Warn(context.Background(), "Search strategy not configured, using default complexity")
		return domain.SearchComplexityModerate
	}

	complexity := s.config.Complexity

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
	if s.config == nil {
		s.logger.Warn(context.Background(), "Search strategy not configured, using default max message IDs")
		return 5
	}

	configuredMax := s.config.MaxMessageIDs
	if configuredMax <= 0 {
		return 5 // Default for generic providers
	}

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
	if s.config == nil {
		s.logger.Warn(context.Background(), "Search strategy not configured, using default timeframe")
		return 90
	}

	configuredDays := s.config.TimeframeDays
	if configuredDays <= 0 {
		return 90 // Default for generic providers
	}

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
