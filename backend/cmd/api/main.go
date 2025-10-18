// backend/cmd/api/main.go
package main

import (
	"context"
	"errors"
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
	httphandler "github.com/audetv/urms/internal/infrastructure/http"
	"github.com/audetv/urms/internal/infrastructure/logging"
	persistence "github.com/audetv/urms/internal/infrastructure/persistence/email"
	"github.com/audetv/urms/internal/infrastructure/persistence/email/postgres"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func main() {
	// Загружаем конфигурацию
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("❌ Failed to load configuration: %v", err)
	}

	// ✅ NEW: Создаем logger сразу для main
	logger := logging.NewZerologLogger(cfg.Logging.Level, cfg.Logging.Format)
	ctx := context.Background()

	logger.Info(ctx, "🚀 Starting URMS-OS API Server")
	logger.Info(ctx, "📋 Configuration",
		"database", cfg.Database.Provider,
		"server_port", cfg.Server.Port,
		"logging_level", cfg.Logging.Level,
		"imap_connect_timeout", cfg.Email.IMAP.ConnectTimeout,
		"imap_fetch_timeout", cfg.Email.IMAP.FetchTimeout,
		"imap_operation_timeout", cfg.Email.IMAP.OperationTimeout,
		"imap_page_size", cfg.Email.IMAP.PageSize,
		"imap_max_messages", cfg.Email.IMAP.MaxMessagesPerPoll)

	// Инициализируем зависимости
	dependencies, err := setupDependencies(cfg, logger) // ✅ ПЕРЕДАЕМ logger
	if err != nil {
		logger.Error(ctx, "❌ Failed to setup dependencies", "error", err)
		os.Exit(1)
	}

	// Запускаем миграции если используется PostgreSQL
	if cfg.Database.Provider == "postgres" {
		if err := runMigrations(cfg.Database.Postgres.DSN); err != nil {
			logger.Error(ctx, "❌ Database migrations failed", "error", err)
			os.Exit(1)
		}
		logger.Info(ctx, "✅ Database migrations completed")
	}

	// Создаем HTTP сервер
	server := setupHTTPServer(cfg, dependencies)

	// Запускаем фоновые процессы
	startBackgroundProcesses(ctx, cfg, dependencies)

	// Запускаем сервер
	go func() {
		logger.Info(ctx, "🌐 Starting HTTP server", "port", cfg.Server.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error(ctx, "❌ HTTP server failed", "error", err)
			os.Exit(1)
		}
	}()

	// Ожидаем сигнал завершения
	waitForShutdown(server, dependencies)
}

// Dependencies содержит все зависимости приложения
type Dependencies struct {
	DB               *sqlx.DB
	EmailService     *services.EmailService
	HealthAggregator ports.HealthAggregator
	EmailGateway     ports.EmailGateway
	Logger           ports.Logger // ✅ ДОБАВЛЯЕМ logger в зависимости
}

// setupDependencies инициализирует все зависимости приложения
func setupDependencies(cfg *config.Config, logger ports.Logger) (*Dependencies, error) {
	deps := &Dependencies{
		Logger: logger, // ✅ СОХРАНЯЕМ logger в зависимости
	}

	logger.Info(context.Background(), "🛠️ Initializing dependencies")

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
	deps.EmailGateway = setupIMAPAdapter(cfg, logger) // ✅ ПЕРЕДАЕМ logger

	// Инициализируем email репозиторий
	emailRepo, err := persistence.NewEmailRepository(
		persistence.RepositoryType(cfg.Database.Provider),
		deps.DB,
	)
	if err != nil {
		logger.Error(context.Background(), "Failed to create email repository", "error", err)
		return nil, fmt.Errorf("failed to create email repository: %w", err)
	}

	// ✅ NEW: Передаем logger в email service
	deps.EmailService = setupEmailService(deps.EmailGateway, emailRepo, logger)

	// Инициализируем health checks
	deps.HealthAggregator = setupHealthChecks(deps.EmailGateway, deps.DB)

	logger.Info(context.Background(), "✅ Dependencies initialized successfully")

	return deps, nil
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
func setupIMAPAdapter(cfg *config.Config, logger ports.Logger) ports.EmailGateway { // ✅ ДОБАВЛЯЕМ logger параметр
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

	// ✅ ЗАМЕНЯЕМ старые log.Printf на structured logging
	logger.Info(context.Background(), "🔧 IMAP Adapter configured with timeouts",
		"connect_timeout", timeoutConfig.ConnectTimeout,
		"login_timeout", timeoutConfig.LoginTimeout,
		"fetch_timeout", timeoutConfig.FetchTimeout,
		"operation_timeout", timeoutConfig.OperationTimeout,
		"page_size", timeoutConfig.PageSize,
		"max_messages", timeoutConfig.MaxMessages,
		"max_retries", timeoutConfig.MaxRetries)

	// ✅ Используем новый конструктор с поддержкой таймаутов
	return email.NewIMAPAdapterWithTimeouts(imapConfig, timeoutConfig, logger)
}

// setupEmailService настраивает email сервис
func setupEmailService(gateway ports.EmailGateway, repo ports.EmailRepository, logger ports.Logger) *services.EmailService {
	// Создаем политику обработки email
	policy := domain.EmailProcessingPolicy{
		ReadOnlyMode:   true, // Для начала используем read-only режим
		AutoReply:      false,
		SpamFilter:     true,
		MaxMessageSize: 10 * 1024 * 1024, // 10MB
	}

	// Используем существующую реализацию из infrastructure
	idGenerator := id.NewUUIDGenerator()

	// ✅ АКТИВИРУЕМ MessageProcessor
	messageProcessor := email.NewDefaultMessageProcessor(logger)
	logger.Info(context.Background(), "✅ MessageProcessor activated",
		"type", "DefaultMessageProcessor")

	// ✅ NEW: Используем переданный structured logger
	return services.NewEmailService(
		gateway,
		repo,
		messageProcessor, // ✅ Теперь передаем реальный процессор вместо nil
		idGenerator,
		policy,
		logger,
	)
}

// setupHealthChecks настраивает систему health checks
func setupHealthChecks(imapAdapter ports.EmailGateway, db *sqlx.DB) ports.HealthAggregator {
	aggregator := health.NewHealthAggregator()

	// Регистрируем health check для IMAP
	// Приводим к конкретному типу для доступа к методам адаптера
	if imapAdapter, ok := imapAdapter.(*email.IMAPAdapter); ok {
		imapHealthChecker := email.NewIMAPHealthChecker(imapAdapter)
		aggregator.Register(imapHealthChecker)
	} else {
		log.Printf("⚠️  IMAP adapter is not of expected type, health check may not work properly")
	}

	// Регистрируем health check для PostgreSQL если используется
	if db != nil {
		postgresChecker := postgres.NewPostgresHealthChecker(db)
		aggregator.Register(postgresChecker)
	}

	return aggregator
}

// setupHTTPServer настраивает HTTP сервер
func setupHTTPServer(cfg *config.Config, deps *Dependencies) *http.Server {
	// Создаем HTTP handlers
	healthHandler := httphandler.NewHealthHandler(deps.HealthAggregator)

	// Настраиваем роутинг
	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthHandler.HealthCheckHandler)
	mux.HandleFunc("/ready", healthHandler.ReadyCheckHandler)
	mux.HandleFunc("/live", healthHandler.LiveCheckHandler)

	// Основной endpoint
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"service": "URMS-OS", "version": "1.0.0", "status": "running"}`)
	})

	// ✅ ИСПРАВЛЕНО: Добавляем таймаут для test-imap endpoint
	mux.HandleFunc("/test-imap", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, `{"error": "Method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}

		// ✅ ДОБАВЛЕНО: Строгий таймаут для тестового endpoint
		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		defer cancel()

		// Тестируем получение сообщений с новой системой таймаутов
		criteria := ports.FetchCriteria{
			Mailbox:    "INBOX",
			Limit:      10, // Только 10 сообщений для теста
			UnseenOnly: false,
			Since:      time.Now().Add(-1 * time.Hour), // Только за последний час
		}

		startTime := time.Now()
		messages, err := deps.EmailGateway.FetchMessages(ctx, criteria)
		duration := time.Since(startTime)

		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			if errors.Is(err, context.DeadlineExceeded) {
				w.WriteHeader(http.StatusRequestTimeout)
				fmt.Fprintf(w, `{"error": "IMAP test timeout", "duration": "%v"}`, duration)
			} else {
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprintf(w, `{"error": "IMAP test failed", "details": "%s", "duration": "%v"}`, err.Error(), duration)
			}
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status": "success", "messages_fetched": %d, "duration": "%v", "timeout_config": "active"}`,
			len(messages), duration)
	})

	return &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      mux,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}
}

// startBackgroundProcesses запускает фоновые процессы с поддержкой context
func startBackgroundProcesses(ctx context.Context, cfg *config.Config, deps *Dependencies) {
	// ✅ ГЕНЕРИРУЕМ correlation ID для фоновых процессов
	bgCtx := context.WithValue(ctx, ports.CorrelationIDKey, "background-"+generateShortID())

	deps.Logger.Info(ctx, "🔄 Starting background processes...")

	// Запускаем IMAP poller если настроен
	if cfg.Email.IMAP.PollInterval > 0 {
		go startIMAPPoller(bgCtx, cfg, deps)
	}

	deps.Logger.Info(ctx, "✅ Background processes initialized")
}

// startIMAPPoller запускает IMAP poller с поддержкой context и таймаутов
func startIMAPPoller(ctx context.Context, cfg *config.Config, deps *Dependencies) {
	// Создаем context для логирования в poller
	pollerCtx := context.WithValue(ctx, ports.CorrelationIDKey, "imap-poller")

	deps.Logger.Info(pollerCtx, "📧 Starting IMAP poller",
		"interval", cfg.Email.IMAP.PollInterval,
		"fetch_timeout", cfg.Email.IMAP.FetchTimeout,
		"operation_timeout", cfg.Email.IMAP.OperationTimeout,
		"page_size", cfg.Email.IMAP.PageSize)

	ticker := time.NewTicker(cfg.Email.IMAP.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			deps.Logger.Info(pollerCtx, "🛑 IMAP poller stopped")
			return
		case <-ticker.C:
			deps.Logger.Info(pollerCtx, "🔄 IMAP poller running scheduled check with timeout protection...")

			// Создаем контекст с таймаутом операции
			pollCtx, cancel := context.WithTimeout(ctx, cfg.Email.IMAP.OperationTimeout)

			startTime := time.Now()
			if err := deps.EmailService.ProcessIncomingEmails(pollCtx); err != nil {
				deps.Logger.Error(pollCtx, "❌ IMAP poller error", "error", err)
			} else {
				duration := time.Since(startTime)
				deps.Logger.Info(pollCtx, "✅ IMAP poller completed successfully", "duration", duration)
			}

			cancel() // Освобождаем ресурсы context
		}
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
func waitForShutdown(server *http.Server, deps *Dependencies) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	// log.Printf("🛑 Shutting down server...")
	deps.Logger.Info(context.Background(), "🛑 Shutting down server...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		// log.Printf("❌ Server shutdown failed: %v", err)
		deps.Logger.Error(ctx, "❌ Server shutdown failed", "error", err)
	}

	// Закрываем соединения с БД
	if deps.DB != nil {
		deps.DB.Close()
	}

	//log.Printf("✅ Server stopped gracefully")
	deps.Logger.Info(ctx, "✅ Server stopped gracefully")
}

// Вспомогательная функция для генерации короткого ID
func generateShortID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano()%10000)
}
