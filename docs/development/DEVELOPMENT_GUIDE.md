## Development with Docker

### Quick Start with Docker
```bash
# 1. Start database
make db-up

# 2. Run migrations
make migrate

# 3. Check status
make migrate-status

# 4. Clean up
make dev-clean
```

### Manual Docker Commands
```bash
# Start only PostgreSQL
docker compose -f docker-compose.db.yml up postgres -d

# View logs
docker compose -f docker-compose.db.yml logs -f postgres

# Stop everything
docker compose -f docker-compose.db.yml down
```

## Provider Support

- ✅ **PostgreSQL** - Full support with migrations
- 🔄 MySQL - Architecture ready, implementation planned
- 🔄 SQLite - Architecture ready, implementation planned

**Current implementation uses PostgreSQL-specific SQL syntax.**
