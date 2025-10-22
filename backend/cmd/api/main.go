// backend/cmd/api/main.go
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/audetv/urms/internal/config"
	"github.com/audetv/urms/internal/core/domain"
	"github.com/audetv/urms/internal/core/ports"
	"github.com/audetv/urms/internal/core/services"
	"github.com/audetv/urms/internal/infrastructure/common/id"
	"github.com/audetv/urms/internal/infrastructure/email"
	imapclient "github.com/audetv/urms/internal/infrastructure/email/imap"
	"github.com/audetv/urms/internal/infrastructure/health"
	"github.com/audetv/urms/internal/infrastructure/http/handlers"
	"github.com/audetv/urms/internal/infrastructure/http/middleware"
	"github.com/audetv/urms/internal/infrastructure/logging"
	persistence "github.com/audetv/urms/internal/infrastructure/persistence/email"
	"github.com/audetv/urms/internal/infrastructure/persistence/email/postgres"
	"github.com/audetv/urms/internal/infrastructure/persistence/task/inmemory"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func main() {
	// Загружаем конфигурацию
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("❌ Failed to load configuration: %v", err)
	}

	// ✅ Создаем logger сразу для main
	logger := logging.NewZerologLogger(cfg.Logging.Level, cfg.Logging.Format)
	ctx := context.Background()

	logger.Info(ctx, "🚀 Starting URMS-OS API Server")

	// Инициализируем зависимости
	dependencies, err := setupDependencies(cfg, logger)
	if err != nil {
		logger.Error(ctx, "❌ Failed to setup dependencies", "error", err)
		os.Exit(1)
	}

	// ✅ ПЕРВОЕ: Запускаем миграции если используется PostgreSQL
	if cfg.Database.Provider == "postgres" {
		if err := runMigrations(cfg.Database.Postgres.DSN); err != nil {
			logger.Error(ctx, "❌ Database migrations failed", "error", err)
			os.Exit(1)
		}
		logger.Info(ctx, "✅ Database migrations completed")
	}

	// ✅ ВТОРОЕ: Запускаем HTTP сервер БЫСТРО
	server := setupHTTPServer(cfg, dependencies)

	// Запускаем сервер в отдельной goroutine
	go func() {
		logger.Info(ctx, "🌐 Starting HTTP server", "port", cfg.Server.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error(ctx, "❌ HTTP server failed", "error", err)
			os.Exit(1)
		}
	}()

	// ✅ КОРОТКАЯ ПАУЗА чтобы HTTP сервер успел запуститься
	time.Sleep(100 * time.Millisecond)

	// ✅ ТРЕТЬЕ: Запускаем фоновые задачи ПОСЛЕ HTTP сервера
	backgroundManager := services.NewBackgroundTaskManager(logger)

	// Регистрируем фоновые задачи
	if cfg.Email.IMAP.PollInterval > 0 {
		emailPollerTask := email.NewEmailPollerTask(
			dependencies.EmailService,
			cfg.Email.IMAP.PollInterval,
			cfg.Email.IMAP.OperationTimeout,
			logger,
		)
		backgroundManager.RegisterTask(emailPollerTask)
	}

	// Запускаем фоновые задачи
	if err := backgroundManager.StartAll(ctx); err != nil {
		logger.Error(ctx, "❌ CRITICAL: Failed to start background tasks - email processing unavailable",
			"error", err,
			"impact", "System cannot process incoming emails - core functionality impaired")
		os.Exit(1) // 🔴 ЖЕСТКИЙ FAIL - система не может функционировать
	}

	// ✅ УБИРАЕМ os.Stdout.Sync() - он может вызывать проблемы

	logger.Info(ctx, "✅ System startup completed - HTTP server running, background tasks started")

	// Ожидаем shutdown
	waitForShutdown(server, dependencies, backgroundManager)
}

// Dependencies содержит все зависимости приложения
type Dependencies struct {
	DB               *sqlx.DB
	EmailService     *services.EmailService
	HealthAggregator ports.HealthAggregator
	EmailGateway     ports.EmailGateway
	Logger           ports.Logger
	// ✅ ДОБАВЛЯЕМ Task Management сервисы
	TaskService     ports.TaskService
	CustomerService ports.CustomerService
	// ✅ ДОБАВЛЯЕМ конфигурационный провайдер
	SearchConfigProvider ports.EmailSearchConfigProvider
}

// setupDependencies инициализирует все зависимости приложения
func setupDependencies(cfg *config.Config, logger ports.Logger) (*Dependencies, error) {
	deps := &Dependencies{
		Logger: logger,
	}

	logger.Info(context.Background(), "🛠️ Initializing dependencies")

	// ✅ ИНИЦИАЛИЗИРУЕМ КОНФИГУРАЦИОННЫЙ ПРОВАЙДЕР ПЕРВЫМ
	deps.SearchConfigProvider = setupSearchConfig(cfg, logger)

	// Инициализируем базу данных если используется PostgreSQL
	if cfg.Database.Provider == "postgres" {
		db, err := setupDatabase(cfg.Database.Postgres)
		if err != nil {
			logger.Error(context.Background(), "Failed to setup database", "error", err)
			return nil, fmt.Errorf("failed to setup database: %w", err)
		}
		deps.DB = db
		logger.Info(context.Background(), "✅ Connected to PostgreSQL database")
	}

	// Инициализируем IMAP адаптер с новой конфигурацией таймаутов
	deps.EmailGateway = setupIMAPAdapter(cfg, logger, deps.SearchConfigProvider) // ✅ ПЕРЕДАЕМ конфигурацию

	// Инициализируем email репозиторий
	emailRepo, err := persistence.NewEmailRepository(
		persistence.RepositoryType(cfg.Database.Provider),
		deps.DB,
	)
	if err != nil {
		logger.Error(context.Background(), "Failed to create email repository", "error", err)
		return nil, fmt.Errorf("failed to create email repository: %w", err)
	}

	// ✅ ПЕРВОЕ: Инициализация Task Management сервисов
	taskRepo := inmemory.NewTaskRepository(logger)
	customerRepo := inmemory.NewCustomerRepository(logger)
	userRepo := inmemory.NewUserRepository(logger)

	deps.TaskService = services.NewTaskService(taskRepo, customerRepo, userRepo, logger)
	deps.CustomerService = services.NewCustomerService(customerRepo, taskRepo, logger)

	logger.Info(context.Background(), "✅ Task Management services initialized")

	// ✅ ВТОРОЕ: Теперь передаем уже созданные сервисы в EmailService
	deps.EmailService = setupEmailServiceWithTaskServices(
		deps.EmailGateway,
		emailRepo,
		deps.TaskService,
		deps.CustomerService,
		deps.SearchConfigProvider, // ✅ ПЕРЕДАЕМ конфигурацию
		logger,
	)

	// Инициализируем health checks
	deps.HealthAggregator = setupHealthChecks(deps.EmailGateway, deps.DB)

	logger.Info(context.Background(), "✅ Dependencies initialized successfully")

	return deps, nil
}

// setupSearchConfig настраивает конфигурационную систему для email поиска
func setupSearchConfig(cfg *config.Config, logger ports.Logger) ports.EmailSearchConfigProvider {
	// ✅ СОЗДАЕМ КОНФИГУРАЦИЮ ДЛЯ EMAIL ПОИСКА
	searchConfig := &email.EmailSearchConfig{
		ThreadSearch: email.ThreadSearchConfig{
			DefaultDaysBack:     180, // 6 месяцев
			ExtendedDaysBack:    365, // 1 год
			MaxDaysBack:         730, // 2 года
			FetchTimeout:        120 * time.Second,
			IncludeSeenMessages: true,
			SubjectPrefixes: []string{
				"Re:", "RE:", "Fwd:", "FW:", "Ответ:", "FWD:",
			},
		},
		ProviderConfig: map[string]email.ProviderSearchConfig{
			"gmail": {
				MaxDaysBack:   365,
				SearchTimeout: 180 * time.Second,
				SupportedFlags: []string{
					"X-GM-RAW", "X-GM-THRID",
				},
				Optimizations: []string{
					"gmail_thread_id", "extended_history", "label_support",
				},
			},
			"yandex": {
				MaxDaysBack:    90,
				SearchTimeout:  90 * time.Second,
				SupportedFlags: []string{},
				Optimizations: []string{
					"russian_subject_support", "cyrillic_encoding",
				},
			},
			"outlook": {
				MaxDaysBack:    180,
				SearchTimeout:  120 * time.Second,
				SupportedFlags: []string{},
				Optimizations: []string{
					"exchange_support", "conversation_id",
				},
			},
			"generic": {
				MaxDaysBack:    180,
				SearchTimeout:  120 * time.Second,
				SupportedFlags: []string{},
				Optimizations: []string{
					"standard_search",
				},
			},
		},
	}

	adapter := email.NewSearchConfigAdapter(searchConfig, logger)

	// ✅ ВАЛИДИРУЕМ КОНФИГУРАЦИЮ
	ctx := context.Background()
	if err := adapter.ValidateConfig(ctx); err != nil {
		logger.Warn(ctx, "Search configuration validation warning",
			"error", err.Error())
	} else {
		logger.Info(ctx, "✅ Email search configuration validated successfully")
	}

	logger.Info(ctx, "🔧 Email search configuration loaded",
		"default_days", searchConfig.ThreadSearch.DefaultDaysBack,
		"extended_days", searchConfig.ThreadSearch.ExtendedDaysBack,
		"max_days", searchConfig.ThreadSearch.MaxDaysBack,
		"providers_supported", len(searchConfig.ProviderConfig))

	return adapter
}

// setupDatabase настраивает соединение с базой данных
func setupDatabase(cfg config.PostgresConfig) (*sqlx.DB, error) {
	db, err := sqlx.Connect("postgres", cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Настраиваем connection pool
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	// ✅ LOG: Убираем log.Printf, логируем в вызывающем коде
	return db, nil
}

// setupIMAPAdapter настраивает IMAP адаптер с поддержкой таймаутов
func setupIMAPAdapter(cfg *config.Config, logger ports.Logger, searchConfig ports.EmailSearchConfigProvider) ports.EmailGateway {
	// Создаем конфигурацию IMAP клиента
	imapConfig := &imapclient.Config{
		Server:   cfg.Email.IMAP.Server,
		Port:     cfg.Email.IMAP.Port,
		Username: cfg.Email.IMAP.Username,
		Password: cfg.Email.IMAP.Password,
		Mailbox:  cfg.Email.IMAP.Mailbox,
		SSL:      cfg.Email.IMAP.SSL,
		Interval: cfg.Email.IMAP.PollInterval,
		ReadOnly: cfg.Email.IMAP.ReadOnly,
		Timeout:  cfg.Email.IMAP.OperationTimeout,

		// ✅ NEW: Extended timeout configuration
		ConnectTimeout:   cfg.Email.IMAP.ConnectTimeout,
		LoginTimeout:     cfg.Email.IMAP.LoginTimeout,
		FetchTimeout:     cfg.Email.IMAP.FetchTimeout,
		OperationTimeout: cfg.Email.IMAP.OperationTimeout,
		PageSize:         cfg.Email.IMAP.PageSize,
	}

	// Создаем конфигурацию таймаутов для адаптера
	timeoutConfig := email.TimeoutConfig{
		ConnectTimeout:   cfg.Email.IMAP.ConnectTimeout,
		LoginTimeout:     cfg.Email.IMAP.LoginTimeout,
		FetchTimeout:     cfg.Email.IMAP.FetchTimeout,
		OperationTimeout: cfg.Email.IMAP.OperationTimeout,
		PageSize:         cfg.Email.IMAP.PageSize,
		MaxMessages:      cfg.Email.IMAP.MaxMessagesPerPoll,
		MaxRetries:       cfg.Email.IMAP.MaxRetries,
		RetryDelay:       cfg.Email.IMAP.RetryDelay,
	}

	logger.Info(context.Background(), "🔧 IMAP Adapter configured with timeouts",
		"connect_timeout", timeoutConfig.ConnectTimeout,
		"login_timeout", timeoutConfig.LoginTimeout,
		"fetch_timeout", timeoutConfig.FetchTimeout,
		"operation_timeout", timeoutConfig.OperationTimeout,
		"page_size", timeoutConfig.PageSize,
		"max_messages", timeoutConfig.MaxMessages,
		"max_retries", timeoutConfig.MaxRetries)

	// ✅ ИСПОЛЬЗУЕМ КОНСТРУКТОР С КОНФИГУРАЦИЕЙ ПОИСКА
	return email.NewIMAPAdapterWithTimeoutsAndConfig(imapConfig, timeoutConfig, searchConfig, logger)
}

// setupEmailServiceWithTaskServices настраивает email сервис с уже созданными Task сервисами
func setupEmailServiceWithTaskServices(
	gateway ports.EmailGateway,
	repo ports.EmailRepository,
	taskService ports.TaskService,
	customerService ports.CustomerService,
	searchConfig ports.EmailSearchConfigProvider, // ✅ ДОБАВЛЯЕМ параметр
	logger ports.Logger,
) *services.EmailService {
	// Создаем политику обработки email
	policy := domain.EmailProcessingPolicy{
		ReadOnlyMode:   true, // Для начала используем read-only режим
		AutoReply:      false,
		SpamFilter:     true,
		MaxMessageSize: 10 * 1024 * 1024, // 10MB
	}

	// Используем существующую реализацию из infrastructure
	idGenerator := id.NewUUIDGenerator()

	// ✅ ИСПОЛЬЗУЕМ уже созданные TaskService и CustomerService + SearchConfig
	messageProcessor := email.NewMessageProcessor(
		taskService,
		customerService,
		gateway,
		searchConfig, // ✅ ПЕРЕДАЕМ конфигурацию
		logger,
	)

	logger.Info(context.Background(), "✅ MessageProcessor activated with enhanced search configuration",
		"type", "MessageProcessor",
		"search_config", "enabled")

	return services.NewEmailService(
		gateway,
		repo,
		messageProcessor,
		idGenerator,
		policy,
		logger,
	)
}

// setupHealthChecks настраивает систему health checks
func setupHealthChecks(imapAdapter ports.EmailGateway, db *sqlx.DB) ports.HealthAggregator {
	aggregator := health.NewHealthAggregator()

	// Регистрируем health check для Email Gateway через адаптер
	if imapAdapter != nil {
		emailHealthChecker := email.NewEmailGatewayHealthAdapter(imapAdapter)
		aggregator.Register(emailHealthChecker)
	} else {
		log.Printf("⚠️  Email gateway is nil, skipping health check")
	}

	// Регистрируем health check для PostgreSQL если используется
	if db != nil {
		postgresChecker := postgres.NewPostgresHealthChecker(db)
		aggregator.Register(postgresChecker)
	}

	return aggregator
}

// setupGinRouter настраивает роутинг с Gin
func setupGinRouter(deps *Dependencies, logger ports.Logger) *gin.Engine {
	router := gin.Default()

	// Настраиваем middleware
	middleware.SetupMiddleware(router, logger)

	// Инициализируем handlers
	taskHandler := handlers.NewTaskHandler(deps.TaskService, logger)
	customerHandler := handlers.NewCustomerHandler(deps.CustomerService, deps.TaskService, logger)
	healthHandler := handlers.NewHealthHandler(deps.HealthAggregator)

	// API Routes v1
	api := router.Group("/api/v1")
	{
		// Tasks
		tasks := api.Group("/tasks")
		{
			tasks.GET("", taskHandler.ListTasks)
			tasks.POST("", taskHandler.CreateTask)
			tasks.POST("/support", taskHandler.CreateSupportTask)
			tasks.GET("/:id", taskHandler.GetTask)
			tasks.PUT("/:id", taskHandler.UpdateTask)
			tasks.DELETE("/:id", taskHandler.DeleteTask)
			tasks.PUT("/:id/status", taskHandler.ChangeStatus)
			tasks.PUT("/:id/assign", taskHandler.AssignTask)
			tasks.GET("/:id/messages", taskHandler.GetTaskMessages)
			tasks.POST("/:id/messages", taskHandler.AddMessage)
		}

		// Customers
		customers := api.Group("/customers")
		{
			customers.GET("", customerHandler.ListCustomers)
			customers.POST("", customerHandler.CreateCustomer)
			customers.GET("/find-or-create", customerHandler.FindOrCreateCustomer)
			customers.GET("/:id", customerHandler.GetCustomer)
			customers.PUT("/:id", customerHandler.UpdateCustomer)
			customers.DELETE("/:id", customerHandler.DeleteCustomer)
			customers.GET("/:id/profile", customerHandler.GetCustomerProfile)
			customers.GET("/:id/tasks", customerHandler.GetCustomerTasks)
		}
	}

	// System routes (legacy compatibility)
	router.GET("/health", healthHandler.HealthCheck)
	router.GET("/ready", healthHandler.ReadyCheck)
	router.GET("/live", healthHandler.LiveCheck)

	// Root endpoint
	router.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"service":     "URMS-OS",
			"version":     "1.0.0",
			"status":      "running",
			"api_version": "v1",
		})
	})

	logger.Info(context.Background(), "✅ Gin router configured with Task Management API")
	return router
}

// setupHTTPServer настраивает HTTP сервер
func setupHTTPServer(cfg *config.Config, deps *Dependencies) *http.Server {
	// Создаем Gin router
	router := setupGinRouter(deps, deps.Logger)

	return &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}
}

// runMigrations запускает миграции базы данных
func runMigrations(dsn string) error {
	log.Printf("🏗️  Running database migrations...")

	// TODO: Интегрировать систему миграций из cmd/migrate
	// Временно просто логируем
	log.Printf("📝 Migration system would run here for DSN: %s", dsn)

	return nil
}

// waitForShutdown ожидает сигнал завершения и graceful shutdown
func waitForShutdown(server *http.Server, deps *Dependencies, bgManager *services.BackgroundTaskManager) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	deps.Logger.Info(ctx, "🛑 Shutting down server...")

	// ✅ 1. ОСТАНАВЛИВАЕМ ФОНОВЫЕ ЗАДАЧИ
	if err := bgManager.StopAll(ctx); err != nil {
		deps.Logger.Error(ctx, "❌ Error stopping background tasks", "error", err)
	}

	// ✅ 2. ОСТАНАВЛИВАЕМ HTTP СЕРВЕР
	if err := server.Shutdown(ctx); err != nil {
		deps.Logger.Error(ctx, "❌ Server shutdown error", "error", err)
	}

	// ✅ 3. ЗАКРЫВАЕМ СОЕДИНЕНИЯ С БД (ВАЖНО!)
	if deps.DB != nil {
		if err := deps.DB.Close(); err != nil {
			deps.Logger.Error(ctx, "❌ Database connection close error", "error", err)
		} else {
			deps.Logger.Info(ctx, "✅ Database connections closed")
		}
	}

	deps.Logger.Info(ctx, "✅ Server stopped gracefully")
}
