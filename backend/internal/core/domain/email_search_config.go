// internal/core/domain/email_search_config.go
package domain

import (
	"fmt"
	"strings"
	"time"
)

// EmailSearchConfig доменная сущность для конфигурации поиска email
type EmailSearchConfig struct {
	defaultDaysBack     int
	extendedDaysBack    int
	maxDaysBack         int
	fetchTimeout        time.Duration
	includeSeenMessages bool
	subjectPrefixes     []string
}

// NewEmailSearchConfig создает новую валидированную конфигурацию поиска
func NewEmailSearchConfig(
	defaultDaysBack int,
	extendedDaysBack int,
	maxDaysBack int,
	fetchTimeout time.Duration,
	includeSeenMessages bool,
	subjectPrefixes []string,
) (*EmailSearchConfig, error) {

	// Валидация бизнес-правил
	if err := validateSearchConfig(
		defaultDaysBack,
		extendedDaysBack,
		maxDaysBack,
		fetchTimeout,
	); err != nil {
		return nil, err
	}

	// Нормализация subject prefixes
	normalizedPrefixes := normalizeSubjectPrefixes(subjectPrefixes)

	config := &EmailSearchConfig{
		defaultDaysBack:     defaultDaysBack,
		extendedDaysBack:    extendedDaysBack,
		maxDaysBack:         maxDaysBack,
		fetchTimeout:        fetchTimeout,
		includeSeenMessages: includeSeenMessages,
		subjectPrefixes:     normalizedPrefixes,
	}

	return config, nil
}

// validateSearchConfig валидирует параметры конфигурации
func validateSearchConfig(
	defaultDaysBack int,
	extendedDaysBack int,
	maxDaysBack int,
	fetchTimeout time.Duration,
) error {
	if defaultDaysBack <= 0 {
		return fmt.Errorf("defaultDaysBack must be positive, got %d", defaultDaysBack)
	}
	if extendedDaysBack < defaultDaysBack {
		return fmt.Errorf("extendedDaysBack (%d) must be greater than or equal to defaultDaysBack (%d)",
			extendedDaysBack, defaultDaysBack)
	}
	if maxDaysBack < extendedDaysBack {
		return fmt.Errorf("maxDaysBack (%d) must be greater than or equal to extendedDaysBack (%d)",
			maxDaysBack, extendedDaysBack)
	}
	if fetchTimeout <= 0 {
		return fmt.Errorf("fetchTimeout must be positive, got %v", fetchTimeout)
	}
	if maxDaysBack > 730 {
		return fmt.Errorf("maxDaysBack cannot exceed 730 days (2 years), got %d", maxDaysBack)
	}

	return nil
}

// normalizeSubjectPrefixes нормализует префиксы subject
func normalizeSubjectPrefixes(prefixes []string) []string {
	if len(prefixes) == 0 {
		// Значения по умолчанию
		return []string{"Re:", "RE:", "Fwd:", "FW:", "Ответ:"}
	}

	normalized := make([]string, 0, len(prefixes))
	seen := make(map[string]bool)

	for _, prefix := range prefixes {
		trimmed := strings.TrimSpace(prefix)
		if trimmed == "" {
			continue
		}
		// Ensure prefix ends with colon and space handling
		if !strings.HasSuffix(trimmed, ":") {
			trimmed = trimmed + ":"
		}
		if !seen[trimmed] {
			seen[trimmed] = true
			normalized = append(normalized, trimmed)
		}
	}

	return normalized
}

// Getter методы для иммутабельности

func (c *EmailSearchConfig) DefaultDaysBack() int {
	return c.defaultDaysBack
}

func (c *EmailSearchConfig) ExtendedDaysBack() int {
	return c.extendedDaysBack
}

func (c *EmailSearchConfig) MaxDaysBack() int {
	return c.maxDaysBack
}

func (c *EmailSearchConfig) FetchTimeout() time.Duration {
	return c.fetchTimeout
}

func (c *EmailSearchConfig) IncludeSeenMessages() bool {
	return c.includeSeenMessages
}

func (c *EmailSearchConfig) SubjectPrefixes() []string {
	// Возвращаем копию для иммутабельности
	result := make([]string, len(c.subjectPrefixes))
	copy(result, c.subjectPrefixes)
	return result
}

// GetSearchSince вычисляет дату для поиска на основе типа
func (c *EmailSearchConfig) GetSearchSince(searchType string) time.Time {
	var daysBack int

	switch searchType {
	case "standard":
		daysBack = c.defaultDaysBack
	case "extended":
		daysBack = c.extendedDaysBack
	case "maximum":
		daysBack = c.maxDaysBack
	default:
		daysBack = c.defaultDaysBack
	}

	return time.Now().Add(-time.Duration(daysBack) * 24 * time.Hour)
}

// GenerateSubjectVariants генерирует варианты subject для поиска
func (c *EmailSearchConfig) GenerateSubjectVariants(baseSubject string) []string {
	if baseSubject == "" {
		return []string{}
	}

	variants := make([]string, 0, len(c.subjectPrefixes)+1)
	variants = append(variants, baseSubject)

	for _, prefix := range c.subjectPrefixes {
		variants = append(variants, prefix+" "+baseSubject)
	}

	return variants
}

// IsValidSearchType проверяет валидность типа поиска
func (c *EmailSearchConfig) IsValidSearchType(searchType string) bool {
	switch searchType {
	case "standard", "extended", "maximum":
		return true
	default:
		return false
	}
}
