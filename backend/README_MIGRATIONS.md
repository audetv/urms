## Database Migrations

### Architecture
- **Port**: `core/ports/migration_gateway.go`
- **Implementation**: `infrastructure/persistence/migrations/`
- **CLI**: `cmd/migrate/main.go`

### Quick Start

1. Set environment variable:
```bash
export URMS_DATABASE_DSN="postgres://user:pass@localhost:5432/urms?sslmode=disable"
```

2. Apply migrations:
```bash
# Using Makefile
make migrate

# Using Go directly
go run cmd/migrate/main.go -dsn "$URMS_DATABASE_DSN" -cmd up
```

3. Check status:
```bash
make migrate-status
```

### Available Commands

- `up` - Apply all pending migrations
- `status` - Show migration status
- `create` - Create new migration template

### Provider Support

- ✅ PostgreSQL
- 🔄 MySQL (planned)
- 🔄 SQLite (planned)

## 🚀 Тестируем миграции

```bash
# Создаем тестовую БД (если еще не создана)
createdb urms

# Запускаем миграции
cd backend
make migrate

# Проверяем статус
make migrate-status
```