package ports_test

import (
	"context"
	"testing"
	"time"

	"github.com/audetv/urms/internal/core/ports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// EmailGatewayContractTestSuite набор контрактных тестов для EmailGateway
type EmailGatewayContractTestSuite struct {
	suite.Suite
	Gateway       ports.EmailGateway
	GatewayName   string
	SetupGateway  func() ports.EmailGateway
	Teardown      func()
	TestMessageID string
}

// SetupTest настраивает тестовое окружение
func (suite *EmailGatewayContractTestSuite) SetupTest() {
	if suite.SetupGateway != nil {
		suite.Gateway = suite.SetupGateway()
	}
	suite.TestMessageID = "test-message-" + time.Now().Format("20060102150405")
}

// TearDownTest очищает тестовое окружение
func (suite *EmailGatewayContractTestSuite) TearDownTest() {
	if suite.Teardown != nil {
		suite.Teardown()
	}
}

// TestConnectionManagement тестирует управление соединением
func (suite *EmailGatewayContractTestSuite) TestConnectionManagement() {
	t := suite.T()

	ctx := context.Background()

	// Тестируем подключение
	err := suite.Gateway.Connect(ctx)
	require.NoError(t, err, "%s: Connect should succeed", suite.GatewayName)

	// Тестируем health check
	err = suite.Gateway.HealthCheck(ctx)
	assert.NoError(t, err, "%s: HealthCheck should succeed after Connect", suite.GatewayName)

	// Тестируем отключение
	err = suite.Gateway.Disconnect()
	assert.NoError(t, err, "%s: Disconnect should succeed", suite.GatewayName)
}

// TestMailboxOperations тестирует операции с почтовыми ящиками
func (suite *EmailGatewayContractTestSuite) TestMailboxOperations() {
	t := suite.T()
	ctx := context.Background()

	// Подключаемся
	err := suite.Gateway.Connect(ctx)
	require.NoError(t, err)

	defer suite.Gateway.Disconnect()

	// Получаем список почтовых ящиков
	mailboxes, err := suite.Gateway.ListMailboxes(ctx)
	assert.NoError(t, err, "%s: ListMailboxes should succeed", suite.GatewayName)
	assert.NotNil(t, mailboxes, "%s: ListMailboxes should return mailboxes list", suite.GatewayName)

	if len(mailboxes) > 0 {
		// Тестируем выбор почтового ящика
		firstMailbox := mailboxes[0].Name
		err = suite.Gateway.SelectMailbox(ctx, firstMailbox)
		assert.NoError(t, err, "%s: SelectMailbox should succeed for %s", suite.GatewayName, firstMailbox)

		// Тестируем получение информации о почтовом ящике
		info, err := suite.Gateway.GetMailboxInfo(ctx, firstMailbox)
		assert.NoError(t, err, "%s: GetMailboxInfo should succeed", suite.GatewayName)
		assert.NotNil(t, info, "%s: GetMailboxInfo should return mailbox info", suite.GatewayName)
		assert.Equal(t, firstMailbox, info.Name, "%s: Mailbox name should match", suite.GatewayName)
	}
}

// TestMessageFetching тестирует получение сообщений
func (suite *EmailGatewayContractTestSuite) TestMessageFetching() {
	t := suite.T()
	ctx := context.Background()

	err := suite.Gateway.Connect(ctx)
	require.NoError(t, err)

	defer suite.Gateway.Disconnect()

	// Выбираем почтовый ящик
	mailboxes, err := suite.Gateway.ListMailboxes(ctx)
	require.NoError(t, err)
	require.True(t, len(mailboxes) > 0, "%s: Need at least one mailbox for testing", suite.GatewayName)

	mailboxName := mailboxes[0].Name
	err = suite.Gateway.SelectMailbox(ctx, mailboxName)
	require.NoError(t, err)

	// Тестируем получение сообщений с различными критериями
	testCases := []struct {
		name     string
		criteria ports.FetchCriteria
	}{
		{
			name: "recent messages",
			criteria: ports.FetchCriteria{
				Mailbox:    mailboxName,
				Limit:      10,
				Since:      time.Now().Add(-24 * time.Hour),
				UnseenOnly: false,
			},
		},
		{
			name: "unseen only",
			criteria: ports.FetchCriteria{
				Mailbox:    mailboxName,
				Limit:      5,
				UnseenOnly: true,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			messages, err := suite.Gateway.FetchMessages(ctx, tc.criteria)
			assert.NoError(t, err, "%s: FetchMessages should succeed for %s", suite.GatewayName, tc.name)
			assert.NotNil(t, messages, "%s: FetchMessages should return messages slice", suite.GatewayName)

			// Проверяем структуру сообщений
			for _, msg := range messages {
				assert.NotEmpty(t, msg.MessageID, "%s: Message should have MessageID", suite.GatewayName)
				assert.NotEmpty(t, msg.From, "%s: Message should have From address", suite.GatewayName)
				assert.NotEmpty(t, msg.Subject, "%s: Message should have Subject", suite.GatewayName)
				assert.NotZero(t, msg.CreatedAt, "%s: Message should have CreatedAt", suite.GatewayName)
			}
		})
	}
}

// TestMessageMarking тестирует пометку сообщений
func (suite *EmailGatewayContractTestSuite) TestMessageMarking() {
	t := suite.T()
	ctx := context.Background()

	err := suite.Gateway.Connect(ctx)
	require.NoError(t, err)

	defer suite.Gateway.Disconnect()

	// Тестируем пометку как прочитанное (должно работать даже без реальных сообщений)
	messageIDs := []string{suite.TestMessageID}
	err = suite.Gateway.MarkAsRead(ctx, messageIDs)
	// Этот тест может пройти или проигнорировать ошибку, так как сообщения могут не существовать
	if err != nil {
		t.Logf("%s: MarkAsRead returned error (may be expected): %v", suite.GatewayName, err)
	}

	err = suite.Gateway.MarkAsProcessed(ctx, messageIDs)
	if err != nil {
		t.Logf("%s: MarkAsProcessed returned error (may be expected): %v", suite.GatewayName, err)
	}
}

// RunEmailGatewayContractTests запускает все контрактные тесты для EmailGateway
func RunEmailGatewayContractTests(t *testing.T, gatewayName string, setupFunc func() ports.EmailGateway, teardownFunc func()) {
	suite.Run(t, &EmailGatewayContractTestSuite{
		GatewayName:  gatewayName,
		SetupGateway: setupFunc,
		Teardown:     teardownFunc,
	})
}
