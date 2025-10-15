// backend/internal/infrastructure/persistence/email/factory.go
package persistence

import (
	"github.com/audetv/urms/internal/core/ports"
	"github.com/audetv/urms/internal/infrastructure/persistence/email/inmemory"
	"github.com/audetv/urms/internal/infrastructure/persistence/email/postgres"
	"github.com/jmoiron/sqlx"
)

type RepositoryType string

const (
	RepositoryTypeInMemory RepositoryType = "inmemory"
	RepositoryTypePostgres RepositoryType = "postgres"
)

func NewEmailRepository(repoType RepositoryType, db *sqlx.DB) ports.EmailRepository {
	switch repoType {
	case RepositoryTypePostgres:
		return postgres.NewPostgresEmailRepository(db)
	case RepositoryTypeInMemory:
		fallthrough
	default:
		return inmemory.NewInMemoryEmailRepo()
	}
}
