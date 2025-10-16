// backend/internal/core/ports/migration_gateway.go
package ports

import (
	"context"
	"time"
)

// MigrationGateway определяет контракт для управления миграциями БД
type MigrationGateway interface {
	// Migrate применяет все непримененные миграции
	Migrate(ctx context.Context) error

	// Status возвращает статус миграций
	Status(ctx context.Context) (*MigrationStatus, error)

	// CreateMigration создает новую миграцию (для разработки)
	CreateMigration(ctx context.Context, name string) error

	// GetProviderInfo возвращает информацию о провайдере БД
	GetProviderInfo() ProviderInfo
}

// ProviderInfo содержит информацию о возможностях провайдера БД
type ProviderInfo struct {
	Name                    MigrationProviderType
	SupportsDDLTransactions bool            // Поддерживает ли DDL в транзакциях
	TransactionMode         TransactionMode // Рекомендуемый режим транзакций
	Description             string
}

// TransactionMode определяет режим работы с транзакциями
type TransactionMode string

const (
	TransactionModeFull    TransactionMode = "full"     // Полная поддержка транзакций
	TransactionModeDDLOnly TransactionMode = "ddl_only" // Только DML транзакции
	TransactionModeNone    TransactionMode = "none"     // Без транзакций
)

// MigrationAnalysis анализирует миграцию для определения стратегии выполнения
type MigrationAnalysis struct {
	UseTransaction bool
	Reason         string
	Warnings       []string
}

// Добавляем константы для анализа SQL
const (
	SQLTypeDDL = "DDL"
	SQLTypeDML = "DML"
	SQLTypeDCL = "DCL"
)

// MigrationStatus представляет статус миграций
type MigrationStatus struct {
	AppliedMigrations []MigrationInfo
	PendingMigrations []MigrationInfo
	TotalCount        int
	DatabaseType      string
}

// MigrationInfo представляет информацию об одной миграции
type MigrationInfo struct {
	Version   string
	Name      string
	AppliedAt *time.Time
	Status    string
}

// const (
// 	TransactionModeAuto   TransactionMode = "auto"   // Решает мигратор
// 	TransactionModeAlways TransactionMode = "always" // Всегда использовать транзакции
// 	TransactionModeNever  TransactionMode = "never"  // Никогда не использовать
// )

// MigrationProviderType тип провайдера БД
type MigrationProviderType string

const (
	PostgreSQLProvider MigrationProviderType = "postgres"
	MySQLProvider      MigrationProviderType = "mysql"
	SQLiteProvider     MigrationProviderType = "sqlite"
)

// MigrationConfig конфигурация для миграций
type MigrationConfig struct {
	Provider     MigrationProviderType
	DataSource   string
	MigrationsFS MigrationFileSystem
}

// MigrationFileSystem абстракция для файловой системы миграций
type MigrationFileSystem interface {
	ReadMigration(version string) ([]byte, error)
	ListMigrations() ([]MigrationFile, error)
}

// MigrationFile представляет файл миграции
type MigrationFile struct {
	Version string
	Name    string
	Content []byte
}
