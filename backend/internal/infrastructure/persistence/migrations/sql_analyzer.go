// backend/internal/infrastructure/persistence/migrations/sql_analyzer.go
package migrations

import (
	"strings"

	"github.com/audetv/urms/internal/core/ports"
)

// SQLAnalyzer анализирует SQL запросы для определения стратегии выполнения
type SQLAnalyzer struct {
	provider ports.MigrationProviderType
}

// NewSQLAnalyzer создает новый анализатор SQL
func NewSQLAnalyzer(provider ports.MigrationProviderType) *SQLAnalyzer {
	return &SQLAnalyzer{
		provider: provider,
	}
}

// AnalyzeMigration анализирует миграцию и возвращает рекомендации
func (a *SQLAnalyzer) AnalyzeMigration(sqlContent string) ports.MigrationAnalysis {
	analysis := ports.MigrationAnalysis{
		UseTransaction: true, // По умолчанию используем транзакции
		Reason:         "Default strategy",
		Warnings:       []string{},
	}

	upperSQL := strings.ToUpper(strings.TrimSpace(sqlContent))

	// Определяем тип SQL операций
	sqlType := a.detectSQLType(upperSQL)

	// Провайдер-специфичная логика
	switch a.provider {
	case ports.PostgreSQLProvider:
		analysis = a.analyzeForPostgreSQL(upperSQL, sqlType)
	case ports.MySQLProvider:
		analysis = a.analyzeForMySQL(upperSQL, sqlType)
	case ports.SQLiteProvider:
		analysis = a.analyzeForSQLite(upperSQL, sqlType)
	}

	return analysis
}

// detectSQLType определяет тип SQL операций в миграции
func (a *SQLAnalyzer) detectSQLType(sql string) string {
	if a.containsAny(sql, []string{"CREATE ", "ALTER ", "DROP ", "TRUNCATE "}) {
		return ports.SQLTypeDDL
	}
	if a.containsAny(sql, []string{"INSERT", "UPDATE", "DELETE", "MERGE"}) {
		return ports.SQLTypeDML
	}
	if a.containsAny(sql, []string{"GRANT", "REVOKE"}) {
		return ports.SQLTypeDCL
	}
	return ports.SQLTypeDDL // По умолчанию считаем DDL
}

// analyzeForPostgreSQL анализирует миграции для PostgreSQL
func (a *SQLAnalyzer) analyzeForPostgreSQL(sql string, sqlType string) ports.MigrationAnalysis {
	analysis := ports.MigrationAnalysis{
		UseTransaction: true, // Всегда используем транзакции с безопасным менеджером
		Reason:         "PostgreSQL with safe transaction manager",
	}

	// Предупреждения для потенциально проблемных операций
	upperSQL := strings.ToUpper(sql)

	warningPatterns := []struct {
		pattern string
		message string
	}{
		{"FOREIGN KEY", "Foreign key operations may affect transaction stability"},
		{"CREATE INDEX CONCURRENTLY", "This operation will be executed without transaction"},
		{"ALTER TABLE.*ADD CONSTRAINT.*FOREIGN KEY", "Foreign key constraint may affect transaction"},
	}

	for _, wp := range warningPatterns {
		if a.containsRegex(upperSQL, wp.pattern) {
			analysis.Warnings = append(analysis.Warnings, wp.message)
		}
	}

	return analysis
}

// analyzeForMySQL анализирует миграции для MySQL
func (a *SQLAnalyzer) analyzeForMySQL(sql string, sqlType string) ports.MigrationAnalysis {
	// MySQL не поддерживает DDL транзакции для большинства операций
	analysis := ports.MigrationAnalysis{
		UseTransaction: sqlType == ports.SQLTypeDML, // Только DML в транзакциях
		Reason:         "MySQL has limited DDL transaction support",
	}

	if sqlType == ports.SQLTypeDDL {
		analysis.Warnings = append(analysis.Warnings,
			"DDL operations in MySQL may cause implicit commits")
	}

	return analysis
}

// analyzeForSQLite анализирует миграции для SQLite
func (a *SQLAnalyzer) analyzeForSQLite(sql string, sqlType string) ports.MigrationAnalysis {
	// SQLite поддерживает большинство операций в транзакциях
	return ports.MigrationAnalysis{
		UseTransaction: true,
		Reason:         "SQLite has good transaction support",
	}
}

// containsAny проверяет содержит ли строка любую из подстрок
func (a *SQLAnalyzer) containsAny(str string, substrs []string) bool {
	for _, substr := range substrs {
		if strings.Contains(str, substr) {
			return true
		}
	}
	return false
}

// containsRegex проверяет совпадение с простыми regex паттернами
func (a *SQLAnalyzer) containsRegex(str, pattern string) bool {
	// Простая реализация для базовых паттернов
	if strings.Contains(pattern, ".*") {
		parts := strings.Split(pattern, ".*")
		for i, part := range parts {
			if !strings.Contains(str, part) {
				return false
			}
			if i > 0 {
				// Проверяем порядок частей
				prevIndex := strings.Index(str, parts[i-1])
				currentIndex := strings.Index(str, part)
				if currentIndex <= prevIndex {
					return false
				}
			}
		}
		return true
	}
	return strings.Contains(str, pattern)
}
