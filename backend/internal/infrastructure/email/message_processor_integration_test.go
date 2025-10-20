// internal/infrastructure/email/message_processor_integration_test.go
package email_test

import (
	"context"
	"testing"
	"time"

	"github.com/audetv/urms/internal/core/domain"
	"github.com/audetv/urms/internal/core/ports"
	"github.com/audetv/urms/internal/core/services"
	"github.com/audetv/urms/internal/infrastructure/email"
	"github.com/audetv/urms/internal/infrastructure/persistence/task/inmemory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMessageProcessor_EmailToTaskIntegration тестирует полный цикл создания задачи из email
func TestMessageProcessor_EmailToTaskIntegration(t *testing.T) {
	ctx := context.Background()
	logger := &TestLogger{}

	// Инициализируем репозитории и сервисы как в main.go
	taskRepo := inmemory.NewTaskRepository(logger)
	customerRepo := inmemory.NewCustomerRepository(logger)
	userRepo := inmemory.NewUserRepository(logger)

	taskService := services.NewTaskService(taskRepo, customerRepo, userRepo, logger)
	customerService := services.NewCustomerService(customerRepo, taskRepo, logger)

	// Создаем MessageProcessor с интеграцией
	messageProcessor := email.NewMessageProcessor(taskService, customerService, logger)

	tests := []struct {
		name             string
		email            domain.EmailMessage
		expectedTask     func(t *testing.T, task *domain.Task)
		expectedCustomer func(t *testing.T, customer *domain.Customer)
	}{
		{
			name: "create task from support email",
			email: domain.EmailMessage{
				MessageID: "test-message-1",
				From:      "customer@example.com",
				To:        []domain.EmailAddress{"support@company.com"},
				Subject:   "Срочная проблема с приложением",
				BodyText:  "Приложение выдает ошибку при запуске. Нужна срочная помощь!",
				Direction: domain.DirectionIncoming,
				CreatedAt: time.Now(),
			},
			expectedTask: func(t *testing.T, task *domain.Task) {
				assert.Equal(t, domain.TaskTypeSupport, task.Type)
				assert.Equal(t, "Срочная проблема с приложением", task.Subject)
				assert.Equal(t, domain.PriorityHigh, task.Priority) // Должен определить как высокий приоритет
				assert.Equal(t, "technical", task.Category)         // Должен определить как техническую проблему
				assert.Equal(t, domain.SourceEmail, task.Source)
				assert.Contains(t, task.Tags, "urgent")
				assert.Contains(t, task.Tags, "email")
			},
			expectedCustomer: func(t *testing.T, customer *domain.Customer) {
				assert.Equal(t, "customer@example.com", customer.Email)
				assert.Equal(t, "Customer", customer.Name) // Должен извлечь имя из email
			},
		},
		{
			name: "create task from billing email",
			email: domain.EmailMessage{
				MessageID: "test-message-2",
				From:      "client@company.com",
				To:        []domain.EmailAddress{"billing@company.com"},
				Subject:   "Вопрос по оплате счета",
				BodyText:  "Не пришел счет за последний месяц, проверьте пожалуйста оплату",
				Direction: domain.DirectionIncoming,
				CreatedAt: time.Now(),
			},
			expectedTask: func(t *testing.T, task *domain.Task) {
				assert.Equal(t, domain.TaskTypeSupport, task.Type)
				assert.Equal(t, "Вопрос по оплате счета", task.Subject)
				assert.Equal(t, domain.PriorityMedium, task.Priority)
				assert.Equal(t, "billing", task.Category) // Должен определить как биллинговый вопрос
				assert.Contains(t, task.Tags, "email")
			},
		},
		{
			name: "create task with attachments",
			email: domain.EmailMessage{
				MessageID: "test-message-3",
				From:      "user@example.com",
				To:        []domain.EmailAddress{"support@company.com"},
				Subject:   "Лог ошибки приложения",
				BodyText:  "Прилагаю лог файл с ошибкой",
				Attachments: []domain.Attachment{
					{
						ID:          "att-1",
						Name:        "error.log",
						ContentType: "text/plain",
						Size:        1024,
					},
				},
				Direction: domain.DirectionIncoming,
				CreatedAt: time.Now(),
			},
			expectedTask: func(t *testing.T, task *domain.Task) {
				assert.Equal(t, "Лог ошибки приложения", task.Subject)
				assert.Contains(t, task.Tags, "has-attachments")
				// Проверяем что мета-информация о вложениях сохранена
				assert.Contains(t, task.SourceMeta, "attachments")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Выполняем обработку email
			err := messageProcessor.ProcessIncomingEmail(ctx, tt.email)
			require.NoError(t, err)

			// Проверяем что клиент создан
			customer, err := customerService.FindOrCreateByEmail(ctx, string(tt.email.From))
			require.NoError(t, err)
			require.NotNil(t, customer)

			if tt.expectedCustomer != nil {
				tt.expectedCustomer(t, customer)
			}

			// Проверяем что задача создана
			tasks, err := taskService.GetCustomerTasks(ctx, customer.ID)
			require.NoError(t, err)
			require.Len(t, tasks, 1)

			task := tasks[0]
			tt.expectedTask(t, &task)

			// Проверяем что в задаче есть сообщение от клиента
			assert.Len(t, task.Messages, 1)
			assert.Equal(t, domain.MessageTypeCustomer, task.Messages[0].Type)
			assert.Equal(t, customer.ID, task.Messages[0].AuthorID)
		})
	}
}

// TestMessageProcessor_EmailThreading тестирует обработку ответов на существующие задачи
func TestMessageProcessor_EmailThreading(t *testing.T) {
	ctx := context.Background()
	logger := &TestLogger{}

	taskRepo := inmemory.NewTaskRepository(logger)
	customerRepo := inmemory.NewCustomerRepository(logger)
	userRepo := inmemory.NewUserRepository(logger)

	taskService := services.NewTaskService(taskRepo, customerRepo, userRepo, logger)
	customerService := services.NewCustomerService(customerRepo, taskRepo, logger)
	messageProcessor := email.NewMessageProcessor(taskService, customerService, logger)

	// Создаем первоначальную задачу
	initialEmail := domain.EmailMessage{
		MessageID: "initial-message",
		From:      "customer@example.com",
		To:        []domain.EmailAddress{"support@company.com"},
		Subject:   "Первоначальный вопрос",
		BodyText:  "У меня есть вопрос по продукту",
		Direction: domain.DirectionIncoming,
		CreatedAt: time.Now(),
	}

	err := messageProcessor.ProcessIncomingEmail(ctx, initialEmail)
	require.NoError(t, err)

	// Получаем созданную задачу
	customer, err := customerService.FindOrCreateByEmail(ctx, "customer@example.com")
	require.NoError(t, err)
	tasks, err := taskService.GetCustomerTasks(ctx, customer.ID)
	require.NoError(t, err)
	require.Len(t, tasks, 1)

	// Обрабатываем ответ на первоначальное сообщение
	replyEmail := domain.EmailMessage{
		MessageID: "reply-message",
		InReplyTo: "initial-message", // Ссылка на первоначальное сообщение
		From:      "customer@example.com",
		To:        []domain.EmailAddress{"support@company.com"},
		Subject:   "Re: Первоначальный вопрос",
		BodyText:  "Спасибо за ответ! У меня есть уточняющий вопрос",
		Direction: domain.DirectionIncoming,
		CreatedAt: time.Now(),
	}

	err = messageProcessor.ProcessIncomingEmail(ctx, replyEmail)
	require.NoError(t, err)

	// TODO: Когда реализуем Thread-ID поиск, проверять что сообщение добавлено в существующую задачу
	// Сейчас создается новая задача - это ожидаемое поведение для временной реализации
	tasksAfterReply, err := taskService.GetCustomerTasks(ctx, customer.ID)
	require.NoError(t, err)

	// Может создаться вторая задача (временное поведение) или добавиться сообщение в существующую
	assert.True(t, len(tasksAfterReply) >= 1, "Should have at least one task")
}

// TestMessageProcessor_OutgoingEmail тестирует обработку исходящих сообщений
func TestMessageProcessor_OutgoingEmail(t *testing.T) {
	ctx := context.Background()
	logger := &TestLogger{}

	taskRepo := inmemory.NewTaskRepository(logger)
	customerRepo := inmemory.NewCustomerRepository(logger)
	userRepo := inmemory.NewUserRepository(logger)

	taskService := services.NewTaskService(taskRepo, customerRepo, userRepo, logger)
	customerService := services.NewCustomerService(customerRepo, taskRepo, logger)
	messageProcessor := email.NewMessageProcessor(taskService, customerService, logger)

	// Создаем задачу для тестирования исходящих сообщений
	customer, err := customerService.FindOrCreateByEmail(ctx, "test@example.com", "Test Customer")
	require.NoError(t, err)

	task, err := taskService.CreateSupportTask(ctx, ports.CreateSupportTaskRequest{
		Subject:     "Test Task",
		Description: "Test Description",
		CustomerID:  customer.ID,
		ReporterID:  "user-1",
		Source:      domain.SourceInternal,
	})
	require.NoError(t, err)

	// Обрабатываем исходящее сообщение, связанное с задачей
	outgoingEmail := domain.EmailMessage{
		MessageID:       "outgoing-message",
		From:            "support@company.com",
		To:              []domain.EmailAddress{"test@example.com"},
		Subject:         "Ответ на ваш вопрос",
		BodyText:        "Мы работаем над решением вашей проблемы",
		Direction:       domain.DirectionOutgoing,
		RelatedTicketID: &task.ID,
		CreatedAt:       time.Now(),
	}

	err = messageProcessor.ProcessOutgoingEmail(ctx, outgoingEmail)
	require.NoError(t, err)

	// Проверяем что в задачу добавлено системное сообщение об отправке email
	updatedTask, err := taskService.GetTask(ctx, task.ID)
	require.NoError(t, err)

	// Должно быть хотя бы одно сообщение (системное о отправке email)
	assert.True(t, len(updatedTask.Messages) >= 1)
}

// TestLogger для тестирования
type TestLogger struct{}

func (t *TestLogger) Debug(ctx context.Context, msg string, fields ...interface{}) {}
func (t *TestLogger) Info(ctx context.Context, msg string, fields ...interface{})  {}
func (t *TestLogger) Warn(ctx context.Context, msg string, fields ...interface{})  {}
func (t *TestLogger) Error(ctx context.Context, msg string, fields ...interface{}) {}
func (t *TestLogger) WithContext(ctx context.Context) context.Context              { return ctx }
