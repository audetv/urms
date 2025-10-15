package ports_test

import (
	"context"
	"testing"
	"time"

	"github.com/audetv/urms/internal/core/domain"
	"github.com/audetv/urms/internal/core/ports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// EmailRepositoryContractTestSuite набор контрактных тестов для EmailRepository
type EmailRepositoryContractTestSuite struct {
	suite.Suite
	Repo         ports.EmailRepository
	RepoName     string
	SetupRepo    func() ports.EmailRepository
	Teardown     func()
	TestMessages []*domain.EmailMessage
}

// SetupTest настраивает тестовое окружение
func (suite *EmailRepositoryContractTestSuite) SetupTest() {
	if suite.SetupRepo != nil {
		suite.Repo = suite.SetupRepo()
	}
	suite.TestMessages = suite.createTestMessages()
}

// TearDownTest очищает тестовое окружение
func (suite *EmailRepositoryContractTestSuite) TearDownTest() {
	if suite.Teardown != nil {
		suite.Teardown()
	}
}

// createTestMessages создает тестовые сообщения
func (suite *EmailRepositoryContractTestSuite) createTestMessages() []*domain.EmailMessage {
	now := time.Now()
	return []*domain.EmailMessage{
		{
			ID:        domain.MessageID("test-msg-1"),
			MessageID: "msg1@test.local",
			From:      "sender1@example.com",
			To:        []domain.EmailAddress{"recipient1@example.com"},
			Subject:   "Test Message 1",
			BodyText:  "This is test message 1",
			Direction: domain.DirectionIncoming,
			Source:    "test",
			CreatedAt: now.Add(-2 * time.Hour),
			UpdatedAt: now.Add(-2 * time.Hour),
			Processed: false,
		},
		{
			ID:          domain.MessageID("test-msg-2"),
			MessageID:   "msg2@test.local",
			From:        "sender2@example.com",
			To:          []domain.EmailAddress{"recipient2@example.com"},
			Subject:     "Test Message 2",
			BodyText:    "This is test message 2",
			BodyHTML:    "<p>This is test message 2</p>",
			Direction:   domain.DirectionOutgoing,
			Source:      "test",
			CreatedAt:   now.Add(-1 * time.Hour),
			UpdatedAt:   now.Add(-1 * time.Hour),
			Processed:   true,
			ProcessedAt: now.Add(-30 * time.Minute),
		},
		{
			ID:        domain.MessageID("test-msg-3"),
			MessageID: "msg3@test.local",
			InReplyTo: "msg1@test.local",
			From:      "sender3@example.com",
			To:        []domain.EmailAddress{"recipient3@example.com"},
			Subject:   "Re: Test Message 1",
			BodyText:  "This is a reply to message 1",
			Direction: domain.DirectionIncoming,
			Source:    "test",
			CreatedAt: now,
			UpdatedAt: now,
			Processed: false,
		},
	}
}

// TestCRUDOperations тестирует базовые CRUD операции
func (suite *EmailRepositoryContractTestSuite) TestCRUDOperations() {
	t := suite.T()
	ctx := context.Background()

	// Test Save
	for _, msg := range suite.TestMessages {
		err := suite.Repo.Save(ctx, msg)
		assert.NoError(t, err, "%s: Save should succeed for message %s", suite.RepoName, msg.MessageID)
	}

	// Test FindByID
	foundMsg, err := suite.Repo.FindByID(ctx, suite.TestMessages[0].ID)
	assert.NoError(t, err, "%s: FindByID should succeed", suite.RepoName)
	assert.NotNil(t, foundMsg, "%s: FindByID should return message", suite.RepoName)
	assert.Equal(t, suite.TestMessages[0].MessageID, foundMsg.MessageID, "%s: MessageID should match", suite.RepoName)

	// Test FindByMessageID
	foundByMsgID, err := suite.Repo.FindByMessageID(ctx, suite.TestMessages[1].MessageID)
	assert.NoError(t, err, "%s: FindByMessageID should succeed", suite.RepoName)
	assert.NotNil(t, foundByMsgID, "%s: FindByMessageID should return message", suite.RepoName)
	assert.Equal(t, suite.TestMessages[1].Subject, foundByMsgID.Subject, "%s: Subject should match", suite.RepoName)

	// Test Update
	updatedMsg := *foundByMsgID
	updatedMsg.Subject = "Updated Subject"
	updatedMsg.BodyText = "Updated body text"
	updatedMsg.UpdatedAt = time.Now()

	err = suite.Repo.Update(ctx, &updatedMsg)
	assert.NoError(t, err, "%s: Update should succeed", suite.RepoName)

	// Verify update
	verifyMsg, err := suite.Repo.FindByID(ctx, updatedMsg.ID)
	assert.NoError(t, err, "%s: FindByID should succeed after update", suite.RepoName)
	assert.Equal(t, "Updated Subject", verifyMsg.Subject, "%s: Subject should be updated", suite.RepoName)

	// Test Delete
	err = suite.Repo.Delete(ctx, suite.TestMessages[2].ID)
	assert.NoError(t, err, "%s: Delete should succeed", suite.RepoName)

	// Verify deletion
	deletedMsg, err := suite.Repo.FindByID(ctx, suite.TestMessages[2].ID)
	assert.Error(t, err, "%s: FindByID should fail for deleted message", suite.RepoName)
	assert.Nil(t, deletedMsg, "%s: Deleted message should not be found", suite.RepoName)
}

// TestQueryOperations тестирует операции запросов
func (suite *EmailRepositoryContractTestSuite) TestQueryOperations() {
	t := suite.T()
	ctx := context.Background()

	// Сохраняем тестовые сообщения
	for _, msg := range suite.TestMessages {
		err := suite.Repo.Save(ctx, msg)
		require.NoError(t, err)
	}

	// Test FindUnprocessed
	unprocessed, err := suite.Repo.FindUnprocessed(ctx)
	assert.NoError(t, err, "%s: FindUnprocessed should succeed", suite.RepoName)
	assert.Len(t, unprocessed, 2, "%s: Should find 2 unprocessed messages", suite.RepoName)

	for _, msg := range unprocessed {
		assert.False(t, msg.Processed, "%s: All returned messages should be unprocessed", suite.RepoName)
	}

	// Test FindByPeriod
	startTime := time.Now().Add(-3 * time.Hour)
	endTime := time.Now().Add(1 * time.Hour)
	messagesInPeriod, err := suite.Repo.FindByPeriod(ctx, startTime, endTime)
	assert.NoError(t, err, "%s: FindByPeriod should succeed", suite.RepoName)
	assert.Len(t, messagesInPeriod, 3, "%s: Should find all test messages in period", suite.RepoName)

	// Test FindByInReplyTo
	replies, err := suite.Repo.FindByInReplyTo(ctx, "msg1@test.local")
	assert.NoError(t, err, "%s: FindByInReplyTo should succeed", suite.RepoName)
	assert.Len(t, replies, 1, "%s: Should find one reply", suite.RepoName)
	assert.Equal(t, "msg3@test.local", replies[0].MessageID, "%s: Should find the correct reply", suite.RepoName)

	// Test FindByReferences
	references := []string{"msg1@test.local", "msg2@test.local"}
	messagesWithRefs, err := suite.Repo.FindByReferences(ctx, references)
	assert.NoError(t, err, "%s: FindByReferences should succeed", suite.RepoName)
	// Может вернуть 0 или более сообщений в зависимости от реализации
	assert.NotNil(t, messagesWithRefs, "%s: FindByReferences should return slice", suite.RepoName)
}

// TestEdgeCases тестирует граничные случаи
func (suite *EmailRepositoryContractTestSuite) TestEdgeCases() {
	t := suite.T()
	ctx := context.Background()

	// Test FindByID with non-existent ID
	nonExistentID := domain.MessageID("non-existent-id")
	msg, err := suite.Repo.FindByID(ctx, nonExistentID)
	assert.Error(t, err, "%s: FindByID should fail for non-existent ID", suite.RepoName)
	assert.Nil(t, msg, "%s: Should return nil for non-existent ID", suite.RepoName)

	// Test FindByMessageID with non-existent MessageID
	nonExistentMsgID := "non-existent-msg-id@test.local"
	msg, err = suite.Repo.FindByMessageID(ctx, nonExistentMsgID)
	assert.Error(t, err, "%s: FindByMessageID should fail for non-existent MessageID", suite.RepoName)
	assert.Nil(t, msg, "%s: Should return nil for non-existent MessageID", suite.RepoName)

	// Test Update with non-existent message
	nonExistentMsg := &domain.EmailMessage{
		ID:        domain.MessageID("non-existent-update"),
		MessageID: "update@test.local",
		From:      "test@example.com",
		To:        []domain.EmailAddress{"test@example.com"},
		Subject:   "Test Update",
	}
	err = suite.Repo.Update(ctx, nonExistentMsg)
	assert.Error(t, err, "%s: Update should fail for non-existent message", suite.RepoName)

	// Test Delete with non-existent ID
	err = suite.Repo.Delete(ctx, domain.MessageID("non-existent-delete"))
	assert.Error(t, err, "%s: Delete should fail for non-existent ID", suite.RepoName)
}

// RunEmailRepositoryContractTests запускает все контрактные тесты для EmailRepository
func RunEmailRepositoryContractTests(t *testing.T, repoName string, setupFunc func() ports.EmailRepository, teardownFunc func()) {
	suite.Run(t, &EmailRepositoryContractTestSuite{
		RepoName:  repoName,
		SetupRepo: setupFunc,
		Teardown:  teardownFunc,
	})
}
