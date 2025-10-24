// internal/infrastructure/email/search_strategies/yandex_search_strategy.go
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

// YandexSearchStrategy реализует ports.SearchStrategy для Yandex
// Соблюдает ограничения Yandex IMAP провайдера
type YandexSearchStrategy struct {
	config *domain.SearchStrategyConfig
	logger ports.Logger
}

// Configure настраивает стратегию с конфигурацией
func (s *YandexSearchStrategy) Configure(config *domain.SearchStrategyConfig) error {
	if config == nil {
		return fmt.Errorf("search strategy configuration is required")
	}

	if err := config.Validate(); err != nil {
		return fmt.Errorf("invalid search strategy configuration: %w", err)
	}

	s.config = config
	s.logger.Info(context.Background(),
		"Yandex search strategy configured",
		"complexity", s.GetComplexity().String(),
		"max_message_ids", s.GetMaxMessageIDs(),
		"timeframe_days", s.GetTimeframeDays())

	return nil
}

// GetComplexity возвращает уровень сложности поиска
func (s *YandexSearchStrategy) GetComplexity() domain.SearchComplexity {
	if s.config == nil {
		s.logger.Warn(context.Background(), "Search strategy not configured, using default complexity")
		return domain.SearchComplexitySimple
	}

	configuredComplexity := s.config.Complexity

	// Yandex limitation: принудительно используем Simple complexity
	if configuredComplexity != domain.SearchComplexitySimple {
		s.logger.Warn(context.Background(),
			"Yandex IMAP provider limitation enforced",
			"configured_complexity", configuredComplexity.String(),
			"enforced_complexity", domain.SearchComplexitySimple.String())
		return domain.SearchComplexitySimple
	}

	return configuredComplexity
}

// GetMaxMessageIDs возвращает максимальное количество Message-ID
func (s *YandexSearchStrategy) GetMaxMessageIDs() int {
	if s.config == nil {
		s.logger.Warn(context.Background(), "Search strategy not configured, using default max message IDs")
		return 1
	}

	configuredMax := s.config.MaxMessageIDs
	if configuredMax <= 0 {
		return 1 // Default for Yandex
	}

	// Yandex limitation: максимум 1 Message-ID
	if configuredMax > 1 {
		s.logger.Warn(context.Background(),
			"Yandex IMAP provider limitation enforced",
			"configured_max_message_ids", configuredMax,
			"enforced_max_message_ids", 1)
		return 1
	}

	return configuredMax
}

// GetTimeframeDays возвращает временной диапазон поиска
func (s *YandexSearchStrategy) GetTimeframeDays() int {
	if s.config == nil {
		s.logger.Warn(context.Background(), "Search strategy not configured, using default timeframe")
		return 180
	}

	if s.config.TimeframeDays > 0 {
		return s.config.TimeframeDays
	}

	return 180 // Default for Yandex
}

// CreateThreadSearchCriteria создает упрощенные IMAP критерии поиска для Yandex
func (s *YandexSearchStrategy) CreateThreadSearchCriteria(threadData ports.ThreadSearchCriteria) (*imap.SearchCriteria, error) {
	ctx := context.Background()
	criteria := &imap.SearchCriteria{}

	if s.config == nil {
		return nil, fmt.Errorf("search strategy not configured")
	}

	// YANDEX-COMPATIBLE: Используем только primary Message-ID для надежности
	if threadData.MessageID != "" {
		primaryMessageID := s.normalizeMessageID(threadData.MessageID)
		criteria.Header = map[string][]string{
			"Message-ID": []string{primaryMessageID},
		}

		s.logger.Debug(ctx, "Yandex search criteria created",
			"primary_message_id", primaryMessageID,
			"strategy", "single_message_id_optimized_for_reliability")
	}

	// CONFIGURATION-DRIVEN: Временной диапазон из конфигурации
	timeframeDays := s.GetTimeframeDays()
	criteria.Since = time.Now().Add(-time.Duration(timeframeDays) * 24 * time.Hour)

	s.logger.Debug(ctx, "Yandex search timeframe configured",
		"timeframe_days", timeframeDays,
		"since_date", criteria.Since.Format("2006-01-02"),
		"provider_limitation", "extended_timeframe_supported")

	return criteria, nil
}

// normalizeMessageID нормализует Message-ID для поиска
func (s *YandexSearchStrategy) normalizeMessageID(messageID string) string {
	normalized := strings.Trim(messageID, "<>")
	normalized = strings.TrimSpace(normalized)
	return normalized
}
