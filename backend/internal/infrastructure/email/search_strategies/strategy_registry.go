// internal/infrastructure/email/search_strategies/strategy_registry.go
package search_strategies

import (
	"context"
	"fmt"

	"github.com/audetv/urms/internal/core/domain"
	"github.com/audetv/urms/internal/core/ports"
)

// StrategyRegistry регистрирует все доступные поисковые стратегии
type StrategyRegistry struct {
	factory ports.SearchStrategyFactory
	logger  ports.Logger
}

// NewStrategyRegistry создает новый реестр стратегий
func NewStrategyRegistry(factory ports.SearchStrategyFactory, logger ports.Logger) *StrategyRegistry {
	return &StrategyRegistry{
		factory: factory,
		logger:  logger,
	}
}

// RegisterAllStrategies регистрирует все стандартные стратегии
func (r *StrategyRegistry) RegisterAllStrategies(ctx context.Context) error {
	// Базовая конфигурация для инициализации
	baseConfig := &domain.SearchStrategyConfig{
		SubjectPrefixes: []string{"Re:", "RE:", "Fwd:", "FW:", "Ответ:"},
		Enabled:         true,
	}

	// Регистрируем Yandex стратегию
	yandexStrategy := &YandexSearchStrategy{logger: r.logger}
	if err := yandexStrategy.Configure(baseConfig); err != nil {
		return fmt.Errorf("failed to configure yandex search strategy: %w", err)
	}
	if err := r.factory.RegisterSearchStrategy(ctx, "yandex", yandexStrategy); err != nil {
		return fmt.Errorf("failed to register yandex search strategy: %w", err)
	}

	// Регистрируем Gmail стратегию
	gmailStrategy := &GmailSearchStrategy{logger: r.logger}
	if err := gmailStrategy.Configure(baseConfig); err != nil {
		return fmt.Errorf("failed to configure gmail search strategy: %w", err)
	}
	if err := r.factory.RegisterSearchStrategy(ctx, "gmail", gmailStrategy); err != nil {
		return fmt.Errorf("failed to register gmail search strategy: %w", err)
	}

	// Регистрируем Generic стратегию
	genericStrategy := &GenericSearchStrategy{logger: r.logger}
	if err := genericStrategy.Configure(baseConfig); err != nil {
		return fmt.Errorf("failed to configure generic search strategy: %w", err)
	}
	if err := r.factory.RegisterSearchStrategy(ctx, "generic", genericStrategy); err != nil {
		return fmt.Errorf("failed to register generic search strategy: %w", err)
	}

	// Регистрируем алиасы для лучшего обнаружения
	aliases := map[string]ports.SearchStrategy{
		"imap.yandex.ru": yandexStrategy,
		"imap.gmail.com": gmailStrategy,
		"imap":           genericStrategy,
		"default":        genericStrategy,
	}

	for alias, strategy := range aliases {
		if err := r.factory.RegisterSearchStrategy(ctx, alias, strategy); err != nil {
			r.logger.Warn(ctx, "Failed to register strategy alias",
				"alias", alias,
				"error", err.Error())
		}
	}

	r.logger.Info(ctx, "All search strategies registered successfully",
		"total_strategies", len(aliases)+3, // 3 основные + алиасы
		"main_strategies", []string{"yandex", "gmail", "generic"})

	return nil
}
