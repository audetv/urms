# URMS Database Migrations

## Overview
Database migration system for URMS-OS following Hexagonal Architecture principles.

**Current Provider**: PostgreSQL  
**Architecture**: Supports multiple databases via ports/interfaces

## Quick Start

### Option 1: PostgreSQL with Docker (Recommended for Development)

```bash
# Start PostgreSQL container
docker run -d \
  --name urms-postgres \
  -e POSTGRES_DB=urms \
  -e POSTGRES_USER=urms \
  -e POSTGRES_PASSWORD=urms \
  -p 5432:5432 \
  postgres:15

# Set environment
export URMS_DATABASE_DSN="postgres://urms:urms@localhost:5432/urms?sslmode=disable"
```

### Option 2: Native PostgreSQL Installation

```bash
# Install PostgreSQL (Ubuntu/Debian)
sudo apt update && sudo apt install postgresql postgresql-contrib

# Create database and user
sudo -u postgres psql -c "CREATE DATABASE urms;"
sudo -u postgres psql -c "CREATE USER urms WITH PASSWORD 'urms';"
sudo -u postgres psql -c "GRANT ALL PRIVILEGES ON DATABASE urms TO urms;"

# Set environment
export URMS_DATABASE_DSN="postgres://urms:urms@localhost:5432/urms?sslmode=disable"
```

### Apply Migrations
```bash
# Using Makefile
make migrate

# Using Go directly  
go run cmd/migrate/main.go -dsn "$URMS_DATABASE_DSN" -cmd up
```

## Docker Compose для разработки

Создаем `docker-compose.yml` для удобства:

```yaml
# backend/docker-compose.db.yml
services:
  postgres:
    image: postgres:18
    environment:
      POSTGRES_DB: urms
      POSTGRES_USER: urms
      POSTGRES_PASSWORD: urms
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U urms -d urms"]
      interval: 10s
      timeout: 5s
      retries: 5

volumes:
  postgres_data:
```

Использование:
```bash
# Запуск БД
docker compose -f docker-compose.db.yml up -d

# Остановка
docker compose -f docker-compose.db.yml down
```