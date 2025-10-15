// backend/internal/infrastructure/persistence/migrations/postgres_migrator.go
package migrations

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"log"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/audetv/urms/internal/core/ports"
	_ "github.com/lib/pq"
)

//go:embed postgres/*.sql
var postgresMigrations embed.FS

// PostgresMigrator реализует ports.MigrationGateway для PostgreSQL
type PostgresMigrator struct {
	db           *sql.DB
	migrations   []migration
	migrationsFS ports.MigrationFileSystem
}

type migration struct {
	version string
	name    string
	content string
}

// NewPostgresMigrator создает новый мигратор для PostgreSQL
func NewPostgresMigrator(db *sql.DB) (ports.MigrationGateway, error) {
	migrator := &PostgresMigrator{
		db:           db,
		migrationsFS: &postgresMigrationFS{},
	}

	if err := migrator.loadMigrations(); err != nil {
		return nil, fmt.Errorf("failed to load migrations: %w", err)
	}

	return migrator, nil
}

// Migrate применяет все непримененные миграции
func (m *PostgresMigrator) Migrate(ctx context.Context) error {
	if err := m.createMigrationsTable(ctx); err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	applied, err := m.getAppliedMigrations(ctx)
	if err != nil {
		return fmt.Errorf("failed to get applied migrations: %w", err)
	}

	for _, migration := range m.migrations {
		if applied[migration.version] {
			log.Printf("Migration %s (%s) already applied, skipping", migration.version, migration.name)
			continue
		}

		log.Printf("Applying migration %s (%s)...", migration.version, migration.name)

		tx, err := m.db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("failed to begin transaction: %w", err)
		}

		if _, err := tx.ExecContext(ctx, migration.content); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to execute migration %s: %w", migration.version, err)
		}

		insertQuery := `INSERT INTO schema_migrations (version, name, applied_at) VALUES ($1, $2, $3)`
		if _, err := tx.ExecContext(ctx, insertQuery, migration.version, migration.name, time.Now().UTC()); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to record migration %s: %w", migration.version, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit migration %s: %w", migration.version, err)
		}

		log.Printf("Migration %s (%s) applied successfully", migration.version, migration.name)
	}

	log.Printf("All migrations applied successfully")
	return nil
}

// Status возвращает статус миграций
func (m *PostgresMigrator) Status(ctx context.Context) (*ports.MigrationStatus, error) {
	appliedMap, err := m.getAppliedMigrationsWithTime(ctx)
	if err != nil {
		return nil, err
	}

	status := &ports.MigrationStatus{
		DatabaseType: "PostgreSQL",
		TotalCount:   len(m.migrations),
	}

	for _, migration := range m.migrations {
		info := ports.MigrationInfo{
			Version: migration.version,
			Name:    migration.name,
			Status:  "PENDING",
		}

		if appliedTime, exists := appliedMap[migration.version]; exists {
			info.Status = "APPLIED"
			info.AppliedAt = &appliedTime
			status.AppliedMigrations = append(status.AppliedMigrations, info)
		} else {
			status.PendingMigrations = append(status.PendingMigrations, info)
		}
	}

	return status, nil
}

// CreateMigration создает новую миграцию (заглушка для будущей реализации)
func (m *PostgresMigrator) CreateMigration(ctx context.Context, name string) error {
	return fmt.Errorf("CreateMigration not implemented for PostgreSQL")
}

// createMigrationsTable создает таблицу для отслеживания миграций
func (m *PostgresMigrator) createMigrationsTable(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version VARCHAR(50) PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			applied_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)
	`
	_, err := m.db.ExecContext(ctx, query)
	return err
}

// getAppliedMigrations возвращает карту примененных миграций
func (m *PostgresMigrator) getAppliedMigrations(ctx context.Context) (map[string]bool, error) {
	applied := make(map[string]bool)

	query := `SELECT version FROM schema_migrations`
	rows, err := m.db.QueryContext(ctx, query)
	if err != nil {
		return applied, nil
	}
	defer rows.Close()

	for rows.Next() {
		var version string
		if err := rows.Scan(&version); err != nil {
			return nil, err
		}
		applied[version] = true
	}

	return applied, nil
}

// getAppliedMigrationsWithTime возвращает примененные миграции с временем применения
func (m *PostgresMigrator) getAppliedMigrationsWithTime(ctx context.Context) (map[string]time.Time, error) {
	applied := make(map[string]time.Time)

	query := `SELECT version, applied_at FROM schema_migrations`
	rows, err := m.db.QueryContext(ctx, query)
	if err != nil {
		return applied, nil
	}
	defer rows.Close()

	for rows.Next() {
		var version string
		var appliedAt time.Time
		if err := rows.Scan(&version, &appliedAt); err != nil {
			return nil, err
		}
		applied[version] = appliedAt
	}

	return applied, nil
}

// loadMigrations загружает миграции из embedded файловой системы
func (m *PostgresMigrator) loadMigrations() error {
	m.migrations = []migration{}

	entries, err := postgresMigrations.ReadDir("postgres")
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".sql" {
			continue
		}

		filename := entry.Name()
		parts := strings.SplitN(filename, "_", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid migration filename: %s", filename)
		}

		version := parts[0]
		name := strings.TrimSuffix(parts[1], ".sql")

		content, err := postgresMigrations.ReadFile("postgres/" + filename)
		if err != nil {
			return fmt.Errorf("failed to read migration %s: %w", filename, err)
		}

		m.migrations = append(m.migrations, migration{
			version: version,
			name:    name,
			content: string(content),
		})
	}

	sort.Slice(m.migrations, func(i, j int) bool {
		return m.migrations[i].version < m.migrations[j].version
	})

	return nil
}

// postgresMigrationFS реализует ports.MigrationFileSystem для PostgreSQL
type postgresMigrationFS struct{}

func (fs *postgresMigrationFS) ReadMigration(version string) ([]byte, error) {
	// Поиск файла по версии
	entries, err := postgresMigrations.ReadDir("postgres")
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), version+"_") {
			return postgresMigrations.ReadFile("postgres/" + entry.Name())
		}
	}

	return nil, fmt.Errorf("migration not found: %s", version)
}

func (fs *postgresMigrationFS) ListMigrations() ([]ports.MigrationFile, error) {
	var files []ports.MigrationFile

	entries, err := postgresMigrations.ReadDir("postgres")
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".sql" {
			continue
		}

		filename := entry.Name()
		parts := strings.SplitN(filename, "_", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid migration filename: %s", filename)
		}

		version := parts[0]
		name := strings.TrimSuffix(parts[1], ".sql")

		content, err := postgresMigrations.ReadFile("postgres/" + filename)
		if err != nil {
			return nil, err
		}

		files = append(files, ports.MigrationFile{
			Version: version,
			Name:    name,
			Content: content,
		})
	}

	return files, nil
}
