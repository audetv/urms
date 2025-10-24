// internal/infrastructure/email/search_strategies/gmail_search_strategy.go
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

// GmailSearchStrategy реализует ports.SearchStrategy для Gmail
type GmailSearchStrategy struct {
	config *domain.SearchStrategyConfig // ✅ ПРАВИЛЬНЫЙ ТИП
	logger ports.Logger
}

// Configure настраивает стратегию с конфигурацией
func (s *GmailSearchStrategy) Configure(config *domain.SearchStrategyConfig) error {
	if config == nil {
		return fmt.Errorf("search strategy configuration is required")
	}

	if err := config.Validate(); err != nil {
		return fmt.Errorf("invalid search strategy configuration: %w", err)
	}

	s.config = config
	s.logger.Info(context.Background(),
		"Gmail search strategy configured",
		"complexity", s.GetComplexity().String(),
		"max_message_ids", s.GetMaxMessageIDs(),
		"timeframe_days", s.GetTimeframeDays())

	return nil
}

// CreateThreadSearchCriteria создает комплексные IMAP критерии поиска для Gmail
func (s *GmailSearchStrategy) CreateThreadSearchCriteria(threadData ports.ThreadSearchCriteria) (*imap.SearchCriteria, error) {
	ctx := context.Background()
	criteria := &imap.SearchCriteria{}

	if s.config == nil {
		return nil, fmt.Errorf("search strategy not configured")
	}

	// GMAIL-OPTIMIZED: Полный набор критериев
	messageIDs := s.collectMessageIDs(threadData)

	if len(messageIDs) > 0 {
		criteria.Header = map[string][]string{
			"Message-ID":  messageIDs,
			"In-Reply-To": messageIDs,
		}

		s.logger.Debug(ctx, "Gmail comprehensive search criteria created",
			"message_ids_count", len(messageIDs),
			"max_allowed", s.GetMaxMessageIDs(),
			"strategy", "comprehensive_thread_search")
	}

	// CONFIGURATION-DRIVEN: Временной диапазон
	timeframeDays := s.GetTimeframeDays()
	criteria.Since = time.Now().Add(-time.Duration(timeframeDays) * 24 * time.Hour)

	// Gmail-specific: поиск по вариантам subject если настроено
	if threadData.Subject != "" && len(s.config.SubjectPrefixes) > 0 {
		subjectVariants := s.generateSubjectVariants(threadData.Subject)
		if criteria.Header == nil {
			criteria.Header = make(map[string][]string)
		}
		criteria.Header["Subject"] = subjectVariants

		s.logger.Debug(ctx, "Gmail subject search variants added",
			"original_subject", threadData.Subject,
			"variants_count", len(subjectVariants))
	}

	s.logger.Debug(ctx, "Gmail search timeframe configured",
		"timeframe_days", timeframeDays,
		"since_date", criteria.Since.Format("2006-01-02"))

	return criteria, nil
}

// GetComplexity возвращает уровень сложности поиска
func (s *GmailSearchStrategy) GetComplexity() domain.SearchComplexity {
	if s.config == nil {
		s.logger.Warn(context.Background(), "Search strategy not configured, using default complexity")
		return domain.SearchComplexityComplex
	}

	return s.config.Complexity // Gmail поддерживает любую сложность
}

// GetMaxMessageIDs возвращает максимальное количество Message-ID для поиска
func (s *GmailSearchStrategy) GetMaxMessageIDs() int {
	if s.config == nil {
		s.logger.Warn(context.Background(), "Search strategy not configured, using default max message IDs")
		return 10
	}

	if s.config.MaxMessageIDs > 0 {
		return s.config.MaxMessageIDs
	}

	return 10 // Default for Gmail
}

// GetTimeframeDays возвращает временной диапазон поиска в днях
func (s *GmailSearchStrategy) GetTimeframeDays() int {
	if s.config == nil {
		s.logger.Warn(context.Background(), "Search strategy not configured, using default timeframe")
		return 365
	}

	if s.config.TimeframeDays > 0 {
		return s.config.TimeframeDays
	}

	return 365 // Default for Gmail
}

// collectMessageIDs собирает Message-ID в пределах лимитов конфигурации
func (s *GmailSearchStrategy) collectMessageIDs(threadData ports.ThreadSearchCriteria) []string {
	maxMessageIDs := s.GetMaxMessageIDs()
	var messageIDs []string

	// Приоритет 1: Основной Message-ID
	if threadData.MessageID != "" {
		messageIDs = append(messageIDs, threadData.MessageID)
	}

	// Приоритет 2: In-Reply-To
	if threadData.InReplyTo != "" && len(messageIDs) < maxMessageIDs {
		messageIDs = append(messageIDs, threadData.InReplyTo)
	}

	// Приоритет 3: References (ограниченное количество)
	if len(threadData.References) > 0 && len(messageIDs) < maxMessageIDs {
		remainingSlots := maxMessageIDs - len(messageIDs)
		if remainingSlots > 0 {
			maxRefs := min(remainingSlots, len(threadData.References))
			messageIDs = append(messageIDs, threadData.References[:maxRefs]...)
		}
	}

	return s.normalizeMessageIDs(messageIDs)
}

// normalizeMessageIDs нормализует и удаляет дубликаты Message-ID
func (s *GmailSearchStrategy) normalizeMessageIDs(ids []string) []string {
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

// generateSubjectVariants создает варианты subject с префиксами из конфигурации
func (s *GmailSearchStrategy) generateSubjectVariants(baseSubject string) []string {
	variants := []string{baseSubject}

	if s.config == nil {
		return variants
	}

	// Получаем чистый subject без префиксов
	cleanSubject := s.extractCleanSubject(baseSubject)

	// Добавляем варианты с настроенными префиксами
	for _, prefix := range s.config.SubjectPrefixes {
		variants = append(variants, prefix+cleanSubject)
	}

	return variants
}

// extractCleanSubject удаляет существующие префиксы для получения чистого subject
func (s *GmailSearchStrategy) extractCleanSubject(subject string) string {
	cleanSubject := subject

	if s.config == nil {
		return cleanSubject
	}

	for _, prefix := range s.config.SubjectPrefixes {
		if strings.HasPrefix(strings.ToUpper(cleanSubject), strings.ToUpper(prefix)) {
			cleanSubject = strings.TrimSpace(cleanSubject[len(prefix):])
			break
		}
	}

	return cleanSubject
}
