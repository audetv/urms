// backend/internal/infrastructure/persistence/migrations/migrate.go
package migrations

import (
	"database/sql"
	"fmt"
	"io/fs"
	"log"
	"path/filepath"
	"sort"
	"strings"

	_ "github.com/lib/pq"
)

// Migration представляет одну миграцию
type Migration struct {
	Version string
	Name    string
	Content string
}

// Migrator управляет миграциями базы данных
type Migrator struct {
	db         *sql.DB
	migrations []Migration
}

// NewMigrator создает новый мигратор
func NewMigrator(db *sql.DB) *Migrator {
	return &Migrator{
		db: db,
	}
}

// LoadMigrations загружает миграции из файловой системы
func (m *Migrator) LoadMigrations(migrationsFS fs.FS) error {
	m.migrations = []Migration{}

	err := fs.WalkDir(migrationsFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() || filepath.Ext(path) != ".sql" {
			return nil
		}

		// Извлекаем версию и имя из имени файла
		filename := filepath.Base(path)
		parts := strings.SplitN(filename, "_", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid migration filename: %s", filename)
		}

		version := parts[0]
		name := strings.TrimSuffix(parts[1], ".sql")

		// Читаем содержимое миграции
		content, err := fs.ReadFile(migrationsFS, path)
		if err != nil {
			return fmt.Errorf("failed to read migration %s: %w", path, err)
		}

		m.migrations = append(m.migrations, Migration{
			Version: version,
			Name:    name,
			Content: string(content),
		})

		return nil
	})

	if err != nil {
		return err
	}

	// Сортируем миграции по версии
	sort.Slice(m.migrations, func(i, j int) bool {
		return m.migrations[i].Version < m.migrations[j].Version
	})

	return nil
}

// CreateMigrationsTable создает таблицу для отслеживания миграций
func (m *Migrator) createMigrationsTable() error {
	query := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version VARCHAR(50) PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			applied_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)
	`
	_, err := m.db.Exec(query)
	return err
}

// GetAppliedMigrations возвращает список примененных миграций
func (m *Migrator) getAppliedMigrations() (map[string]bool, error) {
	applied := make(map[string]bool)

	query := `SELECT version FROM schema_migrations`
	rows, err := m.db.Query(query)
	if err != nil {
		// Таблица может не существовать
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

// Migrate применяет все непримененные миграции
func (m *Migrator) Migrate() error {
	// Создаем таблицу миграций
	if err := m.createMigrationsTable(); err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Получаем примененные миграции
	applied, err := m.getAppliedMigrations()
	if err != nil {
		return fmt.Errorf("failed to get applied migrations: %w", err)
	}

	// Применяем миграции
	for _, migration := range m.migrations {
		if applied[migration.Version] {
			log.Printf("Migration %s (%s) already applied, skipping", migration.Version, migration.Name)
			continue
		}

		log.Printf("Applying migration %s (%s)...", migration.Version, migration.Name)

		// Выполняем миграцию в транзакции
		tx, err := m.db.Begin()
		if err != nil {
			return fmt.Errorf("failed to begin transaction: %w", err)
		}

		// Выполняем SQL миграции
		if _, err := tx.Exec(migration.Content); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to execute migration %s: %w", migration.Version, err)
		}

		// Записываем в таблицу миграций
		insertQuery := `INSERT INTO schema_migrations (version, name) VALUES ($1, $2)`
		if _, err := tx.Exec(insertQuery, migration.Version, migration.Name); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to record migration %s: %w", migration.Version, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit migration %s: %w", migration.Version, err)
		}

		log.Printf("Migration %s (%s) applied successfully", migration.Version, migration.Name)
	}

	log.Printf("All migrations applied successfully")
	return nil
}

// Status показывает статус миграций
func (m *Migrator) Status() error {
	applied, err := m.getAppliedMigrations()
	if err != nil {
		return err
	}

	fmt.Println("Migration Status:")
	fmt.Println("=================")

	for _, migration := range m.migrations {
		status := "PENDING"
		if applied[migration.Version] {
			status = "APPLIED"
		}
		fmt.Printf("%s: %s [%s]\n", migration.Version, migration.Name, status)
	}

	return nil
}
