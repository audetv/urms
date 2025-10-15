package email

import (
	"testing"

	"github.com/audetv/urms/internal/core/ports"
)

// RunEmailGatewayContractTests запускает контрактные тесты для EmailGateway
func RunEmailGatewayContractTests(t *testing.T, gatewayName string, setupFunc func() ports.EmailGateway, teardownFunc func()) {
	// Временная реализация - просто создаем тестовый gateway
	if setupFunc != nil {
		_ = setupFunc()
	}
	t.Logf("Contract tests for %s would run here", gatewayName)
}

// RunEmailRepositoryContractTests запускает контрактные тесты для EmailRepository
func RunEmailRepositoryContractTests(t *testing.T, repoName string, setupFunc func() ports.EmailRepository, teardownFunc func()) {
	// Временная реализация - используем InMemory репозиторий
	if setupFunc != nil {
		_ = setupFunc()
	}
	t.Logf("Contract tests for %s would run here", repoName)
}
