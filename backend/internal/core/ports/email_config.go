// internal/core/ports/email_config.go
package ports

import (
	"context"
	"time"
)

// EmailSearchConfigProvider порт для конфигурации поиска email сообщений
type EmailSearchConfigProvider interface {
	// GetThreadSearchConfig возвращает конфигурацию для thread-aware поиска
	GetThreadSearchConfig(ctx context.Context) (*ThreadSearchConfig, error)

	// GetProviderSpecificConfig возвращает провайдер-специфичную конфигурацию
	GetProviderSpecificConfig(ctx context.Context, provider string) (*ProviderSearchConfig, error)

	// ValidateConfig проверяет валидность конфигурации
	ValidateConfig(ctx context.Context) error
}

// ThreadSearchConfig конфигурация thread-aware поиска
type ThreadSearchConfig struct {
	DefaultDaysBack     int           `yaml:"default_days_back"`
	ExtendedDaysBack    int           `yaml:"extended_days_back"`
	MaxDaysBack         int           `yaml:"max_days_back"`
	FetchTimeout        time.Duration `yaml:"fetch_timeout"`
	IncludeSeenMessages bool          `yaml:"include_seen_messages"`
	SubjectPrefixes     []string      `yaml:"subject_prefixes"`
}

// ProviderSearchConfig провайдер-специфичная конфигурация
type ProviderSearchConfig struct {
	ProviderName   string        `yaml:"provider_name"`
	MaxDaysBack    int           `yaml:"max_days_back"`
	SearchTimeout  time.Duration `yaml:"search_timeout"`
	SupportedFlags []string      `yaml:"supported_flags"`
	Optimizations  []string      `yaml:"optimizations"`
}

// SearchType тип поиска для определения временного диапазона
type SearchType string

const (
	SearchTypeStandard SearchType = "standard"
	SearchTypeExtended SearchType = "extended"
	SearchTypeMaximum  SearchType = "maximum"
)
