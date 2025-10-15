// backend/internal/infrastructure/persistence/migrations/factory.go
package migrations

import (
	"database/sql"
	"fmt"

	"github.com/audetv/urms/internal/core/ports"
	_ "github.com/lib/pq"
)

// NewMigrationGateway создает мигратор для указанного провайдера
func NewMigrationGateway(provider ports.MigrationProviderType, db *sql.DB) (ports.MigrationGateway, error) {
	switch provider {
	case ports.PostgreSQLProvider:
		return NewPostgresMigrator(db)
	case ports.MySQLProvider:
		return nil, fmt.Errorf("MySQL migrations not implemented yet")
	case ports.SQLiteProvider:
		return nil, fmt.Errorf("SQLite migrations not implemented yet")
	default:
		return nil, fmt.Errorf("unsupported database provider: %s", provider)
	}
}
