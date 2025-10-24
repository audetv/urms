// internal/infrastructure/email/pipeline_factory.go
package email

import (
	"context"
	"fmt"

	"github.com/audetv/urms/internal/core/domain"
	"github.com/audetv/urms/internal/core/ports"
	"github.com/audetv/urms/internal/infrastructure/email/pipeline_strategies"
)

// EmailPipelineFactory создает и настраивает Email Pipeline
type EmailPipelineFactory struct {
	emailGateway     ports.EmailGateway
	messageProcessor ports.MessageProcessor
	searchFactory    ports.SearchStrategyFactory
	logger           ports.Logger
}

// NewEmailPipelineFactory создает новую фабрику
func NewEmailPipelineFactory(
	emailGateway ports.EmailGateway,
	messageProcessor ports.MessageProcessor,
	searchFactory ports.SearchStrategyFactory,
	logger ports.Logger,
) *EmailPipelineFactory {
	return &EmailPipelineFactory{
		emailGateway:     emailGateway,
		messageProcessor: messageProcessor,
		searchFactory:    searchFactory,
		logger:           logger,
	}
}

// CreatePipeline создает и настраивает полный Email Pipeline
func (f *EmailPipelineFactory) CreatePipeline(
	ctx context.Context,
	providerConfig *domain.EmailProviderConfig,
) (ports.EmailPipeline, error) {

	f.logger.Info(ctx, "🔧 Creating email pipeline",
		"provider_type", providerConfig.ProviderType,
		"pipeline_strategy", providerConfig.PipelineStrategy)

	// 1. Создаем Pipeline Strategy
	pipelineStrategy, err := f.createPipelineStrategy(providerConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create pipeline strategy: %w", err)
	}

	// 2. Создаем Fetch Criteria Builder
	criteriaBuilder := NewFetchCriteriaBuilder(f.searchFactory, nil, f.logger)

	// 3. Создаем Email Fetcher
	fetcher, err := f.createEmailFetcher(providerConfig, criteriaBuilder)
	if err != nil {
		return nil, fmt.Errorf("failed to create email fetcher: %w", err)
	}

	// 4. Создаем Message Queue
	queue := f.createMessageQueue(pipelineStrategy)

	// 5. Создаем Worker Pool
	workerPool, err := f.createWorkerPool(pipelineStrategy)
	if err != nil {
		return nil, fmt.Errorf("failed to create worker pool: %w", err)
	}

	// 6. Создаем и возвращаем Pipeline
	pipeline := NewEmailPipelineImpl(
		fetcher,
		queue,
		workerPool,
		pipelineStrategy,
		f.searchFactory,
		f.logger,
	)

	f.logger.Info(ctx, "✅ Email pipeline created successfully",
		"provider_type", providerConfig.ProviderType,
		"worker_count", pipelineStrategy.GetWorkerCount(),
		"batch_size", pipelineStrategy.GetBatchSize(),
		"queue_size", pipelineStrategy.GetQueueSize())

	return pipeline, nil
}

// createPipelineStrategy создает pipeline strategy
func (f *EmailPipelineFactory) createPipelineStrategy(
	config *domain.EmailProviderConfig,
) (ports.PipelineStrategy, error) {

	strategyFactory := pipeline_strategies.NewPipelineStrategyFactory(config, f.logger)
	strategy := strategyFactory.GetPipelineStrategy(config.ProviderType)

	f.logger.Debug(context.Background(), "Pipeline strategy created",
		"provider_type", config.ProviderType,
		"strategy_type", fmt.Sprintf("%T", strategy))

	return strategy, nil
}

// createEmailFetcher создает email fetcher
func (f *EmailPipelineFactory) createEmailFetcher(
	config *domain.EmailProviderConfig,
	criteriaBuilder *FetchCriteriaBuilder,
) (ports.EmailFetcher, error) {

	fetcher := NewEmailFetcherImpl(
		f.emailGateway,
		f.searchFactory,
		criteriaBuilder,
		f.logger,
		config.ProviderType,
	)

	// Устанавливаем fetcher в criteria builder
	criteriaBuilder.fetcher = fetcher

	f.logger.Debug(context.Background(), "Email fetcher created",
		"provider_type", config.ProviderType)

	return fetcher, nil
}

// createMessageQueue создает message queue
func (f *EmailPipelineFactory) createMessageQueue(
	strategy ports.PipelineStrategy,
) ports.MessageQueue {

	queueSize := strategy.GetQueueSize()
	queue := NewMessageQueueImpl(queueSize, f.logger)

	f.logger.Debug(context.Background(), "Message queue created",
		"queue_size", queueSize)

	return queue
}

// createWorkerPool создает worker pool
func (f *EmailPipelineFactory) createWorkerPool(
	strategy ports.PipelineStrategy,
) (ports.WorkerPool, error) {

	workerCount := strategy.GetWorkerCount()
	queueSize := strategy.GetQueueSize()

	workerPool := NewWorkerPoolImpl(
		workerCount,
		queueSize,
		f.messageProcessor,
		f.logger,
	)

	f.logger.Debug(context.Background(), "Worker pool created",
		"worker_count", workerCount,
		"queue_size", queueSize)

	return workerPool, nil
}
