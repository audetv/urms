package email

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/audetv/urms/internal/core/domain"
	"github.com/audetv/urms/internal/core/ports"
	"github.com/audetv/urms/internal/core/services"
	"github.com/audetv/urms/internal/infrastructure/common/id"
	imapclient "github.com/audetv/urms/internal/infrastructure/email/imap"
	"github.com/audetv/urms/internal/infrastructure/logging"
	persistence "github.com/audetv/urms/internal/infrastructure/persistence/email"
	"github.com/stretchr/testify/assert"
)

// TestEmailIntegration тестирует полный цикл обработки email
func TestEmailIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()

	// Используем InMemory репозиторий через фабрику
	repo, err := persistence.NewEmailRepository(persistence.RepositoryTypeInMemory, nil)
	if err != nil {
		t.Fatalf("Failed to create email repository: %v", err)
	}
	idGenerator := id.NewUUIDGenerator()

	// Создаем тестовый gateway (заглушка)
	testGateway := &TestEmailGateway{}

	// Создаем политику обработки
	policy := domain.EmailProcessingPolicy{
		ReadOnlyMode:   true,
		AutoReply:      false,
		SpamFilter:     true,
		MaxMessageSize: 10 * 1024 * 1024,
	}

	// Создаем email service
	emailService := services.NewEmailService(
		testGateway,
		repo,
		nil, // Без процессора для простоты
		idGenerator,
		policy,
		&TestLogger{},
	)

	// Тестируем полный цикл
	t.Run("FullEmailProcessingCycle", func(t *testing.T) {
		// 1. Тестируем соединение
		err := emailService.TestConnection(ctx)
		assert.NoError(t, err, "TestConnection should succeed")

		// 2. Тестируем обработку входящих сообщений
		err = emailService.ProcessIncomingEmails(ctx)
		assert.NoError(t, err, "ProcessIncomingEmails should succeed")

		// 3. Тестируем отправку сообщения
		testEmail := domain.EmailMessage{
			From:     domain.EmailAddress("test@urms.local"),
			To:       []domain.EmailAddress{"recipient@example.com"},
			Subject:  "Integration Test Email",
			BodyText: "This is an integration test email",
		}

		err = emailService.SendEmail(ctx, testEmail)
		// В read-only режиме отправка должна быть пропущена
		if err != nil && err.Error() != "read-only mode" {
			assert.NoError(t, err, "SendEmail should succeed or be skipped in read-only mode")
		}

		// 4. Проверяем статистику
		stats, err := emailService.GetEmailStatistics(ctx)
		assert.NoError(t, err, "GetEmailStatistics should succeed")
		assert.NotNil(t, stats, "Statistics should be returned")
	})
}

// TestEmailGatewayContractWithIMAP тестирует контракт с реальным IMAP адаптером
func TestEmailGatewayContractWithIMAP(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping IMAP contract test in short mode")
	}

	// Этот тест требует реальной IMAP конфигурации
	// В CI/CD можно пропустить или использовать тестовый сервер

	imapConfig := &imapclient.Config{
		Server:   "localhost", // Тестовый IMAP сервер
		Port:     1143,        // Тестовый порт
		Username: "test",
		Password: "test",
		SSL:      false,
	}

	setupGateway := func() ports.EmailGateway {
		// ✅ ИСПРАВЛЕНО: Используем legacy конструктор
		return NewIMAPAdapterLegacy(imapConfig)
	}

	// Запускаем контрактные тесты
	RunEmailGatewayContractTests(t, "IMAPAdapter", setupGateway, nil)
}

// TestEmailGatewayContractWithIMAPAndTimeouts тестирует контракт с IMAP адаптером с таймаутами
func TestEmailGatewayContractWithIMAPAndTimeouts(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping IMAP contract test with timeouts in short mode")
	}

	// Создаем тестовый logger
	logger := logging.NewTestLogger()

	imapConfig := &imapclient.Config{
		Server:   "localhost",
		Port:     1143,
		Username: "test",
		Password: "test",
		SSL:      false,
	}

	timeoutConfig := TimeoutConfig{
		ConnectTimeout:   30 * time.Second,
		LoginTimeout:     15 * time.Second,
		FetchTimeout:     60 * time.Second,
		OperationTimeout: 120 * time.Second,
		PageSize:         100,
		MaxMessages:      500,
		MaxRetries:       3,
		RetryDelay:       10 * time.Second,
	}

	setupGateway := func() ports.EmailGateway {
		// ✅ ИСПРАВЛЕНО: Используем новый конструктор с таймаутами
		return NewIMAPAdapter(imapConfig, timeoutConfig, nil, logger)
	}

	// Запускаем контрактные тесты
	RunEmailGatewayContractTests(t, "IMAPAdapterWithTimeouts", setupGateway, nil)
}

// TestEmailRepositoryContractWithPostgres тестирует контракт с Postgres репозиторием
func TestEmailRepositoryContractWithPostgres(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Postgres contract test in short mode")
	}

	setupRepo := func() ports.EmailRepository {
		// TODO: Реализовать создание Postgres репозитория с реальной БД
		repo, err := persistence.NewEmailRepository(persistence.RepositoryTypeInMemory, nil)
		if err != nil {
			t.Fatalf("Failed to create repository: %v", err)
		}
		return repo
	}

	// Запускаем контрактные тесты
	RunEmailRepositoryContractTests(t, "PostgresEmailRepository", setupRepo, nil)
}

// TestEmailGateway тестовый gateway для интеграционных тестов
type TestEmailGateway struct {
	Connected bool
}

func (g *TestEmailGateway) Connect(ctx context.Context) error {
	g.Connected = true
	return nil
}

func (g *TestEmailGateway) Disconnect() error {
	g.Connected = false
	return nil
}

func (g *TestEmailGateway) HealthCheck(ctx context.Context) error {
	if !g.Connected {
		return fmt.Errorf("not connected")
	}
	return nil
}

func (g *TestEmailGateway) FetchMessages(ctx context.Context, criteria ports.FetchCriteria) ([]domain.EmailMessage, error) {
	// Возвращаем пустой список для тестирования
	return []domain.EmailMessage{}, nil
}

func (g *TestEmailGateway) SendMessage(ctx context.Context, msg domain.EmailMessage) error {
	if !g.Connected {
		return fmt.Errorf("not connected")
	}
	return nil
}

func (g *TestEmailGateway) MarkAsRead(ctx context.Context, messageIDs []string) error {
	return nil
}

func (g *TestEmailGateway) MarkAsProcessed(ctx context.Context, messageIDs []string) error {
	return nil
}

// ✅ ДОБАВЛЯЕМ НОВЫЙ МЕТОД
func (g *TestEmailGateway) SearchThreadMessages(ctx context.Context, criteria ports.ThreadSearchCriteria) ([]domain.EmailMessage, error) {
	if !g.Connected {
		return nil, fmt.Errorf("not connected")
	}
	// Возвращаем пустой список для тестирования
	return []domain.EmailMessage{}, nil
}

func (g *TestEmailGateway) ListMailboxes(ctx context.Context) ([]ports.MailboxInfo, error) {
	if !g.Connected {
		return nil, fmt.Errorf("not connected")
	}
	// Возвращаем тестовый почтовый ящик
	return []ports.MailboxInfo{
		{
			Name:     "INBOX",
			Messages: 0,
			Unseen:   0,
			Recent:   0,
		},
	}, nil
}

func (g *TestEmailGateway) SelectMailbox(ctx context.Context, name string) error {
	if !g.Connected {
		return fmt.Errorf("not connected")
	}
	return nil
}

func (g *TestEmailGateway) GetMailboxInfo(ctx context.Context, name string) (*ports.MailboxInfo, error) {
	if !g.Connected {
		return nil, fmt.Errorf("not connected")
	}
	return &ports.MailboxInfo{
		Name:     name,
		Messages: 0,
		Unseen:   0,
		Recent:   0,
	}, nil
}

// TestLogger тестовый логгер
type TestLogger struct{}

func (l *TestLogger) Debug(ctx context.Context, msg string, fields ...interface{}) {}
func (l *TestLogger) Info(ctx context.Context, msg string, fields ...interface{})  {}
func (l *TestLogger) Warn(ctx context.Context, msg string, fields ...interface{})  {}
func (l *TestLogger) Error(ctx context.Context, msg string, fields ...interface{}) {}

func (l *TestLogger) WithContext(ctx context.Context) context.Context {
	return ctx
}
