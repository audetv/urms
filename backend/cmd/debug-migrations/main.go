// backend/cmd/debug-migrations/main.go
package main

import (
	"context"
	"database/sql"
	"log"
	"os"
	"time"

	"github.com/audetv/urms/internal/infrastructure/persistence/migrations"
	_ "github.com/lib/pq"
)

func main() {
	dsn := os.Getenv("URMS_DATABASE_DSN")
	if dsn == "" {
		dsn = "postgres://urms:urms@localhost:5432/urms?sslmode=disable"
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("❌ Failed to connect: %v", err)
	}
	defer db.Close()

	// Проверяем соединение
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("❌ Database ping failed: %v", err)
	}

	log.Println("✅ Database connection successful")

	// Создаем отладочный мигратор
	debugMigrator := migrations.NewDebugMigrator(db)

	// Тест 1: Базовые транзакции
	log.Println("=== TEST 1: Basic Transactions ===")
	if err := debugMigrator.TestTransaction(ctx); err != nil {
		log.Fatalf("❌ Basic transaction tests failed: %v", err)
	}

	// Тест 2: Полный цикл миграции
	log.Println("\n=== TEST 2: Full Migration Flow ===")
	if err := debugMigrator.TestMigrationFlow(ctx); err != nil {
		log.Fatalf("❌ Migration flow tests failed: %v", err)
	}

	log.Println("🎉 All debug tests completed successfully!")
}
