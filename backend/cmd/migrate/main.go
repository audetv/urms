// backend/cmd/migrate/main.go
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/audetv/urms/internal/core/ports"
	"github.com/audetv/urms/internal/infrastructure/persistence/migrations"
	_ "github.com/lib/pq"
)

func main() {
	// Парсим аргументы командной строки
	var (
		dsn      = flag.String("dsn", "", "PostgreSQL DSN (e.g., postgres://user:pass@localhost:5432/dbname)")
		provider = flag.String("provider", "postgres", "Database provider: postgres, mysql, sqlite")
		command  = flag.String("cmd", "up", "Migration command: up, status, create")
		name     = flag.String("name", "", "Migration name (for create command)")
		timeout  = flag.Duration("timeout", 30*time.Second, "Operation timeout")
	)
	flag.Parse()

	// Проверяем обязательные параметры
	if *dsn == "" {
		fmt.Println("❌ Error: DSN is required")
		flag.PrintDefaults()
		os.Exit(1)
	}

	// Определяем providerType ДО использования
	providerType := ports.MigrationProviderType(*provider)

	// Создаем контекст с таймаутом
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	// Создаем DB connection на основе провайдера
	db, err := createDBConnection(providerType, *dsn)
	if err != nil {
		log.Fatalf("❌ Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Проверяем соединение
	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("❌ Failed to ping database: %v", err)
	}

	// Создаем мигратор через фабрику
	migrator, err := migrations.NewMigrationGateway(providerType, db)
	if err != nil {
		log.Fatalf("❌ Failed to create migrator: %v", err)
	}

	// Выполняем команду
	switch *command {
	case "up":
		fmt.Println("🚀 Applying database migrations...")
		if err := migrator.Migrate(ctx); err != nil {
			log.Fatalf("❌ Migration failed: %v", err)
		}
		fmt.Println("✅ All migrations applied successfully")

	case "status":
		fmt.Println("📊 Checking migration status...")
		status, err := migrator.Status(ctx)
		if err != nil {
			log.Fatalf("❌ Failed to get migration status: %v", err)
		}
		printMigrationStatus(status)

	case "create":
		if *name == "" {
			log.Fatal("❌ Migration name is required for create command")
		}
		fmt.Printf("📝 Creating new migration: %s\n", *name)
		if err := migrator.CreateMigration(ctx, *name); err != nil {
			log.Fatalf("❌ Failed to create migration: %v", err)
		}
		fmt.Println("✅ Migration template created successfully")

	default:
		log.Fatalf("❌ Unknown command: %s. Use: up, status, create", *command)
	}
}

// Добавляем функцию для создания DB connection на основе провайдера
func createDBConnection(provider ports.MigrationProviderType, dsn string) (*sql.DB, error) {
	switch provider {
	case ports.PostgreSQLProvider:
		return sql.Open("postgres", dsn)
	case ports.MySQLProvider:
		// return sql.Open("mysql", dsn) - когда добавим MySQL
		return nil, fmt.Errorf("MySQL not implemented yet")
	case ports.SQLiteProvider:
		// return sql.Open("sqlite3", dsn) - когда добавим SQLite
		return nil, fmt.Errorf("SQLite not implemented yet")
	default:
		return nil, fmt.Errorf("unsupported database provider: %s", provider)
	}
}

// printMigrationStatus выводит статус миграций в читаемом формате
func printMigrationStatus(status *ports.MigrationStatus) {
	fmt.Printf("Database: %s\n", status.DatabaseType)
	fmt.Printf("Total Migrations: %d\n", status.TotalCount)
	fmt.Println("")

	if len(status.AppliedMigrations) > 0 {
		fmt.Println("✅ APPLIED MIGRATIONS:")
		for _, migration := range status.AppliedMigrations {
			fmt.Printf("  %s: %s", migration.Version, migration.Name)
			if migration.AppliedAt != nil {
				fmt.Printf(" (applied at: %s)", migration.AppliedAt.Format("2006-01-02 15:04:05"))
			}
			fmt.Println()
		}
		fmt.Println()
	}

	if len(status.PendingMigrations) > 0 {
		fmt.Println("⏳ PENDING MIGRATIONS:")
		for _, migration := range status.PendingMigrations {
			fmt.Printf("  %s: %s\n", migration.Version, migration.Name)
		}
		fmt.Println()
	} else {
		fmt.Println("🎉 All migrations are applied!")
	}

	fmt.Printf("Summary: %d applied, %d pending\n",
		len(status.AppliedMigrations), len(status.PendingMigrations))
}
