// backend/internal/infrastructure/persistence/migrations/postgres_transaction_manager.go
package migrations

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"
)

// PostgresTransactionManager управляет транзакциями с учетом особенностей PostgreSQL
type PostgresTransactionManager struct {
	db *sql.DB
}

// NewPostgresTransactionManager создает новый менеджер транзакций
func NewPostgresTransactionManager(db *sql.DB) *PostgresTransactionManager {
	return &PostgresTransactionManager{db: db}
}

// ExecuteMigration выполняет миграцию с безопасной обработкой транзакций
func (m *PostgresTransactionManager) ExecuteMigration(ctx context.Context, sqlContent string) error {
	// Разбиваем SQL на отдельные statements
	statements := m.splitSQLStatements(sqlContent)

	if len(statements) == 0 {
		return fmt.Errorf("no SQL statements found")
	}

	log.Printf("  Executing %d SQL statements with transaction safety", len(statements))

	// Начинаем транзакцию
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Выполняем каждый statement с мониторингом состояния транзакции
	for i, stmt := range statements {
		if strings.TrimSpace(stmt) == "" {
			continue
		}

		cleanStmt := strings.TrimSpace(stmt)
		log.Printf("  [%d/%d] Executing: %s", i+1, len(statements), m.truncateStatement(cleanStmt))

		// Проверяем, нужно ли выполнять этот statement вне транзакции
		if m.shouldExecuteWithoutTransaction(cleanStmt) {
			log.Printf("  ⚠️  Statement requires non-transactional execution")

			// Коммитим текущую транзакцию если она активна
			if m.isTransactionActive(tx) {
				if err := tx.Commit(); err != nil {
					log.Printf("  ⚠️  Failed to commit before non-transactional statement: %v", err)
				}
			}

			// Выполняем statement без транзакции
			if _, err := m.db.ExecContext(ctx, cleanStmt); err != nil {
				return fmt.Errorf("failed to execute non-transactional statement %d: %w", i+1, err)
			}

			// Начинаем новую транзакцию для следующих statements
			tx, err = m.db.BeginTx(ctx, nil)
			if err != nil {
				return fmt.Errorf("failed to begin new transaction: %w", err)
			}
			continue
		}

		// Выполняем statement в транзакции
		if _, err := tx.ExecContext(ctx, cleanStmt); err != nil {
			// Пытаемся откатить если транзакция еще активна
			if m.isTransactionActive(tx) {
				tx.Rollback()
			}
			return fmt.Errorf("failed to execute statement %d: %w", i+1, err)
		}

		// Проверяем состояние транзакции после выполнения
		if !m.isTransactionActive(tx) {
			log.Printf("  ⚠️  Transaction became inactive after statement %d", i+1)

			// Выполняем оставшиеся statements без транзакции
			for j := i + 1; j < len(statements); j++ {
				if strings.TrimSpace(statements[j]) == "" {
					continue
				}
				remainingStmt := strings.TrimSpace(statements[j])
				log.Printf("  [%d/%d] Executing non-transactional: %s", j+1, len(statements), m.truncateStatement(remainingStmt))

				if _, err := m.db.ExecContext(ctx, remainingStmt); err != nil {
					return fmt.Errorf("failed to execute non-transactional statement %d: %w", j+1, err)
				}
			}
			return nil // Транзакция уже не активна
		}
	}

	// Коммитим транзакцию если она все еще активна
	if m.isTransactionActive(tx) {
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit transaction: %w", err)
		}
		log.Printf("  ✅ Transaction committed successfully")
	}

	return nil
}

// splitSQLStatements разбивает SQL на отдельные statements с поддержкой dollar-quoted strings
func (m *PostgresTransactionManager) splitSQLStatements(sql string) []string {
	// Убираем BEGIN/COMMIT если они есть
	cleanedSQL := regexp.MustCompile(`(?i)^\s*BEGIN\s*;`).ReplaceAllString(sql, "")
	cleanedSQL = regexp.MustCompile(`(?i);\s*COMMIT\s*;?\s*$`).ReplaceAllString(cleanedSQL, "")

	var statements []string
	var currentStmt strings.Builder
	inSingleQuote := false
	inDoubleQuote := false
	inDollarQuote := false
	var dollarTag string

	for i := 0; i < len(cleanedSQL); i++ {
		char := cleanedSQL[i]

		switch {
		case !inSingleQuote && !inDoubleQuote && !inDollarQuote && char == '\'':
			// Начало single-quoted string
			inSingleQuote = true
			currentStmt.WriteByte(char)

		case inSingleQuote && char == '\'':
			// Конец single-quoted string
			inSingleQuote = false
			currentStmt.WriteByte(char)

		case !inSingleQuote && !inDoubleQuote && !inDollarQuote && char == '"':
			// Начало double-quoted string
			inDoubleQuote = true
			currentStmt.WriteByte(char)

		case inDoubleQuote && char == '"':
			// Конец double-quoted string
			inDoubleQuote = false
			currentStmt.WriteByte(char)

		case !inSingleQuote && !inDoubleQuote && !inDollarQuote && char == '$':
			// Возможное начало dollar-quoted string
			if i+1 < len(cleanedSQL) {
				// Ищем конец dollar tag
				j := i + 1
				for j < len(cleanedSQL) && (cleanedSQL[j] >= 'a' && cleanedSQL[j] <= 'z' ||
					cleanedSQL[j] >= 'A' && cleanedSQL[j] <= 'Z' ||
					cleanedSQL[j] >= '0' && cleanedSQL[j] <= '9' ||
					cleanedSQL[j] == '_') {
					j++
				}

				if j < len(cleanedSQL) && cleanedSQL[j] == '$' {
					// Нашли dollar-quoted string
					dollarTag = cleanedSQL[i : j+1]
					inDollarQuote = true
					currentStmt.WriteString(dollarTag)
					i = j // Пропускаем обработанные символы
					continue
				}
			}
			currentStmt.WriteByte(char)

		case inDollarQuote && strings.HasPrefix(cleanedSQL[i:], dollarTag):
			// Конец dollar-quoted string
			currentStmt.WriteString(dollarTag)
			inDollarQuote = false
			i += len(dollarTag) - 1 // Пропускаем dollar tag

		case !inSingleQuote && !inDoubleQuote && !inDollarQuote && char == ';':
			// Конец statement (только если не внутри quoted string)
			stmt := strings.TrimSpace(currentStmt.String())
			if stmt != "" {
				statements = append(statements, stmt)
			}
			currentStmt.Reset()

		default:
			currentStmt.WriteByte(char)
		}
	}

	// Добавляем последний statement если он есть
	lastStmt := strings.TrimSpace(currentStmt.String())
	if lastStmt != "" {
		statements = append(statements, lastStmt)
	}

	return statements
}

// shouldExecuteWithoutTransaction определяет, нужно ли выполнять statement без транзакции
func (m *PostgresTransactionManager) shouldExecuteWithoutTransaction(statement string) bool {
	upperStmt := strings.ToUpper(statement)

	// Операции, которые не поддерживают транзакции или могут их сломать
	nonTransactionalPatterns := []string{
		"CREATE INDEX CONCURRENTLY",
		"REINDEX",
		"VACUUM",
		"CLUSTER",
		"CREATE DATABASE",
		"ALTER DATABASE",
		"CREATE TABLESPACE",
		"ALTER TABLESPACE",
	}

	for _, pattern := range nonTransactionalPatterns {
		if strings.Contains(upperStmt, pattern) {
			return true
		}
	}

	return false
}

// isTransactionActive проверяет активна ли транзакция
func (m *PostgresTransactionManager) isTransactionActive(tx *sql.Tx) bool {
	if tx == nil {
		return false
	}

	// Пытаемся выполнить простой запрос в транзакции
	_, err := tx.Exec("SELECT 1")
	return err == nil
}

// truncateStatement обрезает длинный SQL для логов
func (m *PostgresTransactionManager) truncateStatement(stmt string) string {
	if len(stmt) > 100 {
		return stmt[:100] + "..."
	}
	return stmt
}

// ExecuteWithRecord выполняет миграцию и записывает факт применения
func (m *PostgresTransactionManager) ExecuteWithRecord(ctx context.Context, migration migration) error {
	// Выполняем миграцию
	if err := m.ExecuteMigration(ctx, migration.content); err != nil {
		return fmt.Errorf("failed to execute migration %s: %w", migration.version, err)
	}

	// Записываем факт миграции в отдельной транзакции
	return m.recordMigration(ctx, migration)
}

// recordMigration записывает факт применения миграции
func (m *PostgresTransactionManager) recordMigration(ctx context.Context, migration migration) error {
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin record transaction: %w", err)
	}

	insertQuery := `INSERT INTO schema_migrations (version, name, applied_at) VALUES ($1, $2, $3)`
	if _, err := tx.ExecContext(ctx, insertQuery, migration.version, migration.name, time.Now().UTC()); err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to record migration %s: %w", migration.version, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit migration record %s: %w", migration.version, err)
	}

	log.Printf("  ✅ Migration %s recorded successfully", migration.version)
	return nil
}
