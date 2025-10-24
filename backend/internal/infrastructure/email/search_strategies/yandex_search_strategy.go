// internal/infrastructure/email/search_strategies/yandex_search_strategy.go
package search_strategies

import (
	"context"
	"strings"
	"time"

	"github.com/audetv/urms/internal/core/domain"
	"github.com/audetv/urms/internal/core/ports"
	"github.com/emersion/go-imap"
)

// YandexSearchStrategy реализует ports.SearchStrategy для Yandex
// Соблюдает ограничения Yandex IMAP провайдера
type YandexSearchStrategy struct {
	config *domain.EmailProviderConfig
	logger ports.Logger
}

// NewYandexSearchStrategy создает новую Yandex-оптимизированную стратегию
func NewYandexSearchStrategy(
	config *domain.EmailProviderConfig,
	logger ports.Logger,
) *YandexSearchStrategy {
	// Валидируем конфигурацию при создании
	if err := config.Validate(); err != nil {
		logger.Warn(context.Background(),
			"Yandex search strategy configuration validation warning",
			"error", err.Error())
	}

	return &YandexSearchStrategy{
		config: config,
		logger: logger,
	}
}

// CreateThreadSearchCriteria создает упрощенные IMAP критерии поиска для Yandex
func (s *YandexSearchStrategy) CreateThreadSearchCriteria(threadData ports.ThreadSearchCriteria) (*imap.SearchCriteria, error) {
	ctx := context.Background()
	criteria := &imap.SearchCriteria{}

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

// GetComplexity возвращает уровень сложности поиска с учетом ограничений Yandex
func (s *YandexSearchStrategy) GetComplexity() domain.SearchComplexity {
	configuredComplexity := s.config.GetSearchComplexity()

	// Yandex limitation: принудительно используем Simple complexity для надежности
	if configuredComplexity != domain.SearchComplexitySimple {
		s.logger.Warn(context.Background(),
			"Yandex IMAP provider limitation enforced",
			"configured_complexity", configuredComplexity.String(),
			"enforced_complexity", domain.SearchComplexitySimple.String(),
			"reason", "yandex_imap_limitation_simple_search_only")
		return domain.SearchComplexitySimple
	}

	return configuredComplexity
}

// GetMaxMessageIDs возвращает максимальное количество Message-ID для поиска
func (s *YandexSearchStrategy) GetMaxMessageIDs() int {
	configuredMax := s.config.GetMaxMessageIDs()

	// Yandex limitation: максимум 1 Message-ID для надежной работы
	if configuredMax > 1 {
		s.logger.Warn(context.Background(),
			"Yandex IMAP provider limitation enforced",
			"configured_max_message_ids", configuredMax,
			"enforced_max_message_ids", 1,
			"reason", "yandex_imap_limitation_single_message_id")
		return 1
	}

	return configuredMax
}

// GetTimeframeDays возвращает временной диапазон поиска в днях
func (s *YandexSearchStrategy) GetTimeframeDays() int {
	return s.config.GetTimeframeDays() // Yandex поддерживает extended timeframe
}

// normalizeMessageID нормализует Message-ID для поиска
func (s *YandexSearchStrategy) normalizeMessageID(messageID string) string {
	normalized := strings.Trim(messageID, "<>")
	normalized = strings.TrimSpace(normalized)
	return normalized
}
