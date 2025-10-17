// backend/internal/infrastructure/persistence/email/factory.go
package persistence

import (
	"fmt"

	"github.com/audetv/urms/internal/core/ports"
	"github.com/audetv/urms/internal/infrastructure/persistence/email/inmemory"
	"github.com/audetv/urms/internal/infrastructure/persistence/email/postgres"
	"github.com/jmoiron/sqlx"
)

// RepositoryType тип репозитория
type RepositoryType string

const (
	RepositoryTypeInMemory RepositoryType = "inmemory"
	RepositoryTypePostgres RepositoryType = "postgres"
)

// NewEmailRepository создает репозиторий на основе конфигурации
func NewEmailRepository(repoType RepositoryType, db *sqlx.DB) (ports.EmailRepository, error) {
	switch repoType {
	case RepositoryTypePostgres:
		if db == nil {
			return nil, fmt.Errorf("database connection is required for PostgreSQL repository")
		}
		return postgres.NewPostgresEmailRepository(db), nil
	case RepositoryTypeInMemory:
		fallthrough
	default:
		return inmemory.NewInMemoryEmailRepo(), nil // ✅ Без ошибки для InMemory
	}
}
