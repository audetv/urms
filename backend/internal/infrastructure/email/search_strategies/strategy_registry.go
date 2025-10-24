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

// RegisterAllStrategies регистрирует все стандартные стратегииии
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

	r.logger.Info(ctx, "All search strategies registered successfully",
		"total_strategies", 3,
		"strategies", []string{"yandex", "gmail", "generic"})

	return nil
}
