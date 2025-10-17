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
	httphandler "github.com/audetv/urms/internal/infrastructure/http"
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

	log.Printf("🚀 Starting URMS-OS API Server")
	log.Printf("📋 Configuration:")
	log.Printf("   Database: %s", cfg.Database.Provider)
	log.Printf("   Server Port: %d", cfg.Server.Port)
	log.Printf("   Logging Level: %s", cfg.Logging.Level)
	log.Printf("   IMAP Timeouts: Connect=%v, Fetch=%v, Operation=%v",
		cfg.Email.IMAP.ConnectTimeout, cfg.Email.IMAP.FetchTimeout, cfg.Email.IMAP.OperationTimeout)
	log.Printf("   IMAP Pagination: PageSize=%d, MaxMessages=%d",
		cfg.Email.IMAP.PageSize, cfg.Email.IMAP.MaxMessagesPerPoll)

	// Инициализируем зависимости
	dependencies, err := setupDependencies(cfg)
	if err != nil {
		log.Fatalf("❌ Failed to setup dependencies: %v", err)
	}

	// Запускаем миграции если используется PostgreSQL
	if cfg.Database.Provider == "postgres" {
		if err := runMigrations(cfg.Database.Postgres.DSN); err != nil {
			log.Fatalf("❌ Database migrations failed: %v", err)
		}
		log.Printf("✅ Database migrations completed")
	}

	// Создаем HTTP сервер
	server := setupHTTPServer(cfg, dependencies)

	// Запускаем фоновые процессы
	startBackgroundProcesses(context.Background(), cfg, dependencies)

	// Запускаем сервер
	go func() {
		log.Printf("🌐 Starting HTTP server on :%d", cfg.Server.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("❌ HTTP server failed: %v", err)
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
}

// setupDependencies инициализирует все зависимости приложения
func setupDependencies(cfg *config.Config) (*Dependencies, error) {
	deps := &Dependencies{}

	// Инициализируем базу данных если используется PostgreSQL
	if cfg.Database.Provider == "postgres" {
		db, err := setupDatabase(cfg.Database.Postgres)
		if err != nil {
			return nil, fmt.Errorf("failed to setup database: %w", err)
		}
		deps.DB = db
	}

	// Инициализируем IMAP адаптер с новой конфигурацией таймаутов
	deps.EmailGateway = setupIMAPAdapter(cfg)

	// Инициализируем email репозиторий
	emailRepo, err := persistence.NewEmailRepository(
		persistence.RepositoryType(cfg.Database.Provider),
		deps.DB,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create email repository: %w", err)
	}

	// Инициализируем email сервис
	deps.EmailService = setupEmailService(deps.EmailGateway, emailRepo)

	// Инициализируем health checks
	deps.HealthAggregator = setupHealthChecks(deps.EmailGateway, deps.DB)

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

	log.Printf("✅ Connected to PostgreSQL database")

	return db, nil
}

// setupIMAPAdapter настраивает IMAP адаптер с поддержкой таймаутов
func setupIMAPAdapter(cfg *config.Config) ports.EmailGateway {
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

	log.Printf("🔧 IMAP Adapter configured with timeouts:")
	log.Printf("   - Connect: %v", timeoutConfig.ConnectTimeout)
	log.Printf("   - Login: %v", timeoutConfig.LoginTimeout)
	log.Printf("   - Fetch: %v", timeoutConfig.FetchTimeout)
	log.Printf("   - Operation: %v", timeoutConfig.OperationTimeout)
	log.Printf("   - Page Size: %d", timeoutConfig.PageSize)
	log.Printf("   - Max Messages: %d", timeoutConfig.MaxMessages)
	log.Printf("   - Max Retries: %d", timeoutConfig.MaxRetries)

	// ✅ Используем новый конструктор с поддержкой таймаутов
	return email.NewIMAPAdapterWithTimeouts(imapConfig, timeoutConfig)
}

// setupEmailService настраивает email сервис
func setupEmailService(gateway ports.EmailGateway, repo ports.EmailRepository) *services.EmailService {
	// Создаем политику обработки email
	policy := domain.EmailProcessingPolicy{
		ReadOnlyMode:   true, // Для начала используем read-only режим
		AutoReply:      false,
		SpamFilter:     true,
		MaxMessageSize: 10 * 1024 * 1024, // 10MB
	}

	// Используем существующую реализацию из infrastructure
	idGenerator := id.NewUUIDGenerator()
	// TODO: Добавить реальный и Logger
	// Временно используем заглушки из domain пакета
	logger := &services.ConsoleLogger{}

	return services.NewEmailService(gateway, repo, nil, idGenerator, policy, logger)
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

	// TODO: Добавить остальные API endpoints
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"service": "URMS-OS", "version": "1.0.0", "status": "running"}`)
	})

	// ✅ NEW: Добавляем endpoint для тестирования IMAP с таймаутами
	mux.HandleFunc("/test-imap", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, `{"error": "Method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()

		// Тестируем получение сообщений с новой системой таймаутов
		criteria := ports.FetchCriteria{
			Mailbox:    "INBOX",
			Limit:      10, // Только 10 сообщений для теста
			UnseenOnly: false,
			Since:      time.Now().Add(-24 * time.Hour),
		}

		messages, err := deps.EmailGateway.FetchMessages(ctx, criteria)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, `{"error": "IMAP test failed", "details": "%s"}`, err.Error())
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status": "success", "messages_fetched": %d, "timeout_config": "active"}`, len(messages))
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
	log.Printf("🔄 Starting background processes...")

	// Запускаем IMAP poller если настроен
	if cfg.Email.IMAP.PollInterval > 0 {
		go startIMAPPoller(ctx, cfg, deps)
	}

	log.Printf("✅ Background processes initialized")
}

// startIMAPPoller запускает IMAP poller с поддержкой context и таймаутов
func startIMAPPoller(ctx context.Context, cfg *config.Config, deps *Dependencies) {
	log.Printf("📧 Starting IMAP poller with interval: %v", cfg.Email.IMAP.PollInterval)
	log.Printf("   Timeout configuration active:")
	log.Printf("   - Fetch: %v", cfg.Email.IMAP.FetchTimeout)
	log.Printf("   - Operation: %v", cfg.Email.IMAP.OperationTimeout)
	log.Printf("   - Page Size: %d", cfg.Email.IMAP.PageSize)

	ticker := time.NewTicker(cfg.Email.IMAP.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Printf("🛑 IMAP poller stopped")
			return
		case <-ticker.C:
			log.Printf("🔄 IMAP poller running scheduled check with timeout protection...")

			// Создаем контекст с таймаутом операции
			pollCtx, cancel := context.WithTimeout(ctx, cfg.Email.IMAP.OperationTimeout)

			startTime := time.Now()
			if err := deps.EmailService.ProcessIncomingEmails(pollCtx); err != nil {
				log.Printf("❌ IMAP poller error: %v", err)
			} else {
				duration := time.Since(startTime)
				log.Printf("✅ IMAP poller completed successfully in %v", duration)
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

	log.Printf("🛑 Shutting down server...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("❌ Server shutdown failed: %v", err)
	}

	// Закрываем соединения с БД
	if deps.DB != nil {
		deps.DB.Close()
	}

	log.Printf("✅ Server stopped gracefully")
}
