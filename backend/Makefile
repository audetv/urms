# backend/Makefile

.PHONY: db-up db-down db-logs migrate migrate-status test

# Database
db-up:
	@echo "🐘 Starting PostgreSQL database..."
	@docker compose -f docker-compose.db.yml up -d

db-down:
	@echo "🛑 Stopping database..."
	@docker compose -f docker-compose.db.yml down

db-logs:
	@docker compose -f docker-compose.db.yml logs -f

# Environment
export URMS_DATABASE_DSN ?= postgres://urms:urms@localhost:5432/urms?sslmode=disable

# Migrations
migrate: db-up
	@echo "🚀 Waiting for database to be ready..."
	@sleep 3
	@echo "Applying database migrations..."
	@go run cmd/migrate/main.go -dsn "$(URMS_DATABASE_DSN)" -cmd up

migrate-status:
	@go run cmd/migrate/main.go -dsn "$(URMS_DATABASE_DSN)" -cmd status

# Development
dev-setup: db-up migrate
	@echo "✅ Development environment setup complete"

dev-clean: db-down
	@echo "🧹 Development environment cleaned up"

# Tests
test:
	@go test ./... -v

test-short:
	@go test ./... -v -short