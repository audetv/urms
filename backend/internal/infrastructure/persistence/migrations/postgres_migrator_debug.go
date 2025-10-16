// backend/internal/infrastructure/persistence/migrations/postgres_migrator_debug.go
package migrations

import (
	"context"
	"database/sql"
	"fmt"
	"log"
)

// DebugMigrator для отладки проблем с транзакциями
type DebugMigrator struct {
	db *sql.DB
}

func NewDebugMigrator(db *sql.DB) *DebugMigrator {
	return &DebugMigrator{db: db}
}

func (m *DebugMigrator) TestTransaction(ctx context.Context) error {
	log.Println("🧪 Testing transaction lifecycle...")

	// Тест 1: Простая транзакция
	log.Println("1. Testing simple transaction...")
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	log.Println("   ✅ Transaction started")

	// Простой запрос
	_, err = tx.ExecContext(ctx, "SELECT 1")
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to execute query: %w", err)
	}
	log.Println("   ✅ Query executed")

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("failed to commit: %w", err)
	}
	log.Println("   ✅ Transaction committed")

	// Тест 2: Транзакция с CREATE TABLE
	log.Println("2. Testing transaction with CREATE TABLE...")
	tx2, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction 2: %w", err)
	}
	log.Println("   ✅ Transaction 2 started")

	_, err = tx2.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS test_migration_debug (
			id SERIAL PRIMARY KEY,
			name VARCHAR(100)
		)
	`)
	if err != nil {
		tx2.Rollback()
		return fmt.Errorf("failed to create test table: %w", err)
	}
	log.Println("   ✅ CREATE TABLE executed")

	err = tx2.Commit()
	if err != nil {
		return fmt.Errorf("failed to commit transaction 2: %w", err)
	}
	log.Println("   ✅ Transaction 2 committed")

	log.Println("🎉 All transaction tests passed!")
	return nil
}

// Добавим в postgres_migrator_debug.go

func (m *DebugMigrator) TestMigrationFlow(ctx context.Context) error {
	log.Println("🧪 Testing full migration flow...")

	// Создаем таблицу миграций
	log.Println("1. Creating migrations table...")
	_, err := m.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS debug_schema_migrations (
			version VARCHAR(50) PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			applied_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}
	log.Println("   ✅ Migrations table created")

	// Тест полного цикла миграции
	log.Println("2. Testing complete migration cycle...")

	// Транзакция для миграции
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin migration transaction: %w", err)
	}

	// Выполняем тестовую миграцию
	_, err = tx.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS debug_test_table (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			message_id VARCHAR(500) NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)
	`)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to create test table: %w", err)
	}
	log.Println("   ✅ Test table creation executed")

	// Записываем миграцию
	_, err = tx.ExecContext(ctx, `
		INSERT INTO debug_schema_migrations (version, name) VALUES ($1, $2)
	`, "debug_001", "test_migration")
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to record migration: %w", err)
	}
	log.Println("   ✅ Migration record inserted")

	// Коммитим
	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("failed to commit migration: %w", err)
	}
	log.Println("   ✅ Migration transaction committed")

	// Проверяем что все создалось
	var count int
	err = m.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM debug_test_table").Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to verify test table: %w", err)
	}
	log.Printf("   ✅ Test table verified, row count: %d", count)

	var migrationCount int
	err = m.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM debug_schema_migrations").Scan(&migrationCount)
	if err != nil {
		return fmt.Errorf("failed to verify migration record: %w", err)
	}
	log.Printf("   ✅ Migration record verified, count: %d", migrationCount)

	log.Println("🎉 Full migration flow test passed!")
	return nil
}
