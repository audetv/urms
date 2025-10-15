# backend/Makefile

.PHONY: migrate migrate-status migrate-create test

# Environment
export URMS_DATABASE_DSN ?= postgres://urms:urms@localhost:5432/urms?sslmode=disable

# Миграции
migrate:
	@echo "🚀 Applying database migrations..."
	@go run cmd/migrate/main.go -dsn "$(URMS_DATABASE_DSN)" -cmd up

migrate-status:
	@echo "📊 Checking migration status..."
	@go run cmd/migrate/main.go -dsn "$(URMS_DATABASE_DSN)" -cmd status

migrate-create:
	@if [ -z "$(name)" ]; then \
		echo "❌ Usage: make migrate-create name=<migration_name>"; \
		exit 1; \
	fi
	@go run cmd/migrate/main.go -dsn "$(URMS_DATABASE_DSN)" -cmd create -name "$(name)"

# Тесты
test:
	@go test ./... -v

test-short:
	@go test ./... -v -short

# Development
dev-setup: migrate
	@echo "✅ Development environment setup complete"