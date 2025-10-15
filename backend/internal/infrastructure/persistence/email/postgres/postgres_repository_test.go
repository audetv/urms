// backend/internal/infrastructure/persistence/email/postgres_repository_test.go
package postgres

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/audetv/urms/internal/core/domain"
	"github.com/audetv/urms/internal/core/ports"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// PostgresEmailRepositoryTestSuite набор тестов для PostgreSQL репозитория
type PostgresEmailRepositoryTestSuite struct {
	suite.Suite
	db           *sqlx.DB
	repo         ports.EmailRepository
	ctx          context.Context
	testMessages []*domain.EmailMessage
}

// SetupSuite настраивает тестовое окружение
func (suite *PostgresEmailRepositoryTestSuite) SetupSuite() {
	suite.ctx = context.Background()

	// Получаем DSN из environment variables
	dsn := os.Getenv("TEST_POSTGRES_DSN")
	if dsn == "" {
		dsn = "postgres://urms_test:urms_test@localhost:5432/urms_test?sslmode=disable"
	}

	// Подключаемся к тестовой базе
	db, err := sqlx.Connect("postgres", dsn)
	require.NoError(suite.T(), err, "Failed to connect to test database")

	suite.db = db
	suite.repo = NewPostgresEmailRepository(db)

	// Создаем тестовые сообщения
	suite.testMessages = suite.createTestMessages()
}

// TearDownSuite очищает тестовое окружение
func (suite *PostgresEmailRepositoryTestSuite) TearDownSuite() {
	if suite.db != nil {
		suite.db.Close()
	}
}

// SetupTest подготавливает тестовые данные перед каждым тестом
func (suite *PostgresEmailRepositoryTestSuite) SetupTest() {
	// Очищаем таблицу перед каждым тестом
	_, err := suite.db.Exec("TRUNCATE TABLE email_messages CASCADE")
	require.NoError(suite.T(), err, "Failed to truncate table")
}

// createTestMessages создает тестовые сообщения
func (suite *PostgresEmailRepositoryTestSuite) createTestMessages() []*domain.EmailMessage {
	now := time.Now().UTC()

	return []*domain.EmailMessage{
		{
			ID:          domain.MessageID("test-msg-1"),
			MessageID:   "msg1@test.local",
			InReplyTo:   "",
			From:        domain.EmailAddress("sender1@example.com"),
			To:          []domain.EmailAddress{"recipient1@example.com"},
			Subject:     "Test Message 1",
			BodyText:    "This is test message 1",
			Direction:   domain.DirectionIncoming,
			Source:      "test",
			CreatedAt:   now.Add(-2 * time.Hour),
			UpdatedAt:   now.Add(-2 * time.Hour),
			Processed:   false,
			ProcessedAt: time.Time{},
		},
		{
			ID:          domain.MessageID("test-msg-2"),
			MessageID:   "msg2@test.local",
			InReplyTo:   "msg1@test.local",
			From:        domain.EmailAddress("sender2@example.com"),
			To:          []domain.EmailAddress{"recipient2@example.com"},
			Subject:     "Re: Test Message 1",
			BodyText:    "This is a reply to message 1",
			Direction:   domain.DirectionIncoming,
			Source:      "test",
			CreatedAt:   now.Add(-1 * time.Hour),
			UpdatedAt:   now.Add(-1 * time.Hour),
			Processed:   true,
			ProcessedAt: now,
		},
		{
			ID:        domain.MessageID("test-msg-3"),
			MessageID: "msg3@test.local",
			From:      domain.EmailAddress("sender3@example.com"),
			To:        []domain.EmailAddress{"recipient3@example.com"},
			Subject:   "Test Message 3",
			BodyText:  "This is test message 3",
			Direction: domain.DirectionOutgoing,
			Source:    "test",
			CreatedAt: now,
			UpdatedAt: now,
			Processed: false,
		},
	}
}

// TestCRUDOperations тестирует базовые CRUD операции
func (suite *PostgresEmailRepositoryTestSuite) TestCRUDOperations() {
	t := suite.T()
	ctx := suite.ctx

	// Test Save
	for _, msg := range suite.testMessages {
		err := suite.repo.Save(ctx, msg)
		assert.NoError(t, err, "Save should succeed")
	}

	// Test FindByID
	foundMsg, err := suite.repo.FindByID(ctx, suite.testMessages[0].ID)
	assert.NoError(t, err, "FindByID should succeed")
	assert.NotNil(t, foundMsg, "FindByID should return message")
	assert.Equal(t, suite.testMessages[0].MessageID, foundMsg.MessageID)

	// Test FindByMessageID
	foundByMsgID, err := suite.repo.FindByMessageID(ctx, suite.testMessages[1].MessageID)
	assert.NoError(t, err, "FindByMessageID should succeed")
	assert.NotNil(t, foundByMsgID, "FindByMessageID should return message")
	assert.Equal(t, suite.testMessages[1].Subject, foundByMsgID.Subject)

	// Test Update
	updatedMsg := *foundByMsgID
	updatedMsg.Subject = "Updated Subject"
	updatedMsg.BodyText = "Updated body text"
	updatedMsg.UpdatedAt = time.Now().UTC()

	err = suite.repo.Update(ctx, &updatedMsg)
	assert.NoError(t, err, "Update should succeed")

	// Verify update
	verifyMsg, err := suite.repo.FindByID(ctx, updatedMsg.ID)
	assert.NoError(t, err, "FindByID should succeed after update")
	assert.Equal(t, "Updated Subject", verifyMsg.Subject)

	// Test Delete
	err = suite.repo.Delete(ctx, suite.testMessages[2].ID)
	assert.NoError(t, err, "Delete should succeed")

	// Verify deletion
	deletedMsg, err := suite.repo.FindByID(ctx, suite.testMessages[2].ID)
	assert.Error(t, err, "FindByID should fail for deleted message")
	assert.Nil(t, deletedMsg, "Deleted message should not be found")
}

// TestQueryOperations тестирует операции запросов
func (suite *PostgresEmailRepositoryTestSuite) TestQueryOperations() {
	t := suite.T()
	ctx := suite.ctx

	// Сохраняем тестовые сообщения
	for _, msg := range suite.testMessages {
		err := suite.repo.Save(ctx, msg)
		require.NoError(t, err)
	}

	// Test FindUnprocessed
	unprocessed, err := suite.repo.FindUnprocessed(ctx)
	assert.NoError(t, err, "FindUnprocessed should succeed")
	assert.Len(t, unprocessed, 2, "Should find 2 unprocessed messages")

	for _, msg := range unprocessed {
		assert.False(t, msg.Processed, "All returned messages should be unprocessed")
	}

	// Test FindByPeriod
	startTime := time.Now().UTC().Add(-3 * time.Hour)
	endTime := time.Now().UTC().Add(1 * time.Hour)
	messagesInPeriod, err := suite.repo.FindByPeriod(ctx, startTime, endTime)
	assert.NoError(t, err, "FindByPeriod should succeed")
	assert.Len(t, messagesInPeriod, 3, "Should find all test messages in period")

	// Test FindByInReplyTo
	replies, err := suite.repo.FindByInReplyTo(ctx, "msg1@test.local")
	assert.NoError(t, err, "FindByInReplyTo should succeed")
	assert.Len(t, replies, 1, "Should find one reply")
	assert.Equal(t, "msg2@test.local", replies[0].MessageID)
}

// TestEdgeCases тестирует граничные случаи
func (suite *PostgresEmailRepositoryTestSuite) TestEdgeCases() {
	t := suite.T()
	ctx := suite.ctx

	// Test FindByID with non-existent ID
	nonExistentID := domain.MessageID("non-existent-id")
	msg, err := suite.repo.FindByID(ctx, nonExistentID)
	assert.Error(t, err, "FindByID should fail for non-existent ID")
	assert.Nil(t, msg, "Should return nil for non-existent ID")

	// Test Update with non-existent message
	nonExistentMsg := &domain.EmailMessage{
		ID:        domain.MessageID("non-existent-update"),
		MessageID: "update@test.local",
		From:      domain.EmailAddress("test@example.com"),
		To:        []domain.EmailAddress{"test@example.com"},
		Subject:   "Test Update",
	}
	err = suite.repo.Update(ctx, nonExistentMsg)
	assert.Error(t, err, "Update should fail for non-existent message")
}

// TestConcurrentAccess тестирует конкурентный доступ
func (suite *PostgresEmailRepositoryTestSuite) TestConcurrentAccess() {
	t := suite.T()
	ctx := suite.ctx

	// Сохраняем базовое сообщение
	baseMsg := suite.testMessages[0]
	err := suite.repo.Save(ctx, baseMsg)
	require.NoError(t, err)

	// Запускаем несколько горутин для конкурентного обновления
	const goroutines = 5
	errors := make(chan error, goroutines)

	for i := 0; i < goroutines; i++ {
		go func(index int) {
			// Каждая горутина читает, обновляет и сохраняет
			msg, err := suite.repo.FindByID(ctx, baseMsg.ID)
			if err != nil {
				errors <- err
				return
			}

			msg.Subject = fmt.Sprintf("Updated by goroutine %d", index)
			msg.UpdatedAt = time.Now().UTC()

			err = suite.repo.Update(ctx, msg)
			errors <- err
		}(i)
	}

	// Ждем завершения всех горутин
	for i := 0; i < goroutines; i++ {
		err := <-errors
		assert.NoError(t, err, "Concurrent operation should succeed")
	}
}

// RunPostgresEmailRepositoryTests запускает все тесты для PostgreSQL репозитория
func RunPostgresEmailRepositoryTests(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping PostgreSQL integration tests in short mode")
	}

	suite.Run(t, new(PostgresEmailRepositoryTestSuite))
}
