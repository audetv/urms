# URMS-OS Architecture Principles
**Version: 1.0** | **Project: Unified Request Management System**  
**License: Apache 2.0** | **Status: Active**

## 🎯 Core Philosophy: "No Vendor Lock-in"

### 📌 Strategic Decisions (NOT Replaceable)
- **Backend**: Go (strategic choice)
- **Frontend**: Vue 3 + TypeScript + Pinia  
- **AI/ML**: Python + FastAPI
- **Primary DB**: PostgreSQL
- **Architecture**: Hexagonal Architecture (Ports & Adapters)

### 🔄 Replaceable Components (MUST Abstract)
- Email providers (IMAP/SMTP/API)
- AI/ML models (Qwen/OpenAI/Local)
- Search engines (Manticore/Elasticsearch/PostgreSQL)
- Storage providers (Local/S3/Cloud)
- Message queues (Redis/NATS/RabbitMQ)

## 🏗️ Project Structure Convention

```text
urms-os/
├── core/ # PURE BUSINESS LOGIC
│ ├── domain/ # Entities, Value Objects, Aggregates
│ ├── ports/ # INTERFACES (contracts)
│ └── services/ # Business services (use cases)
├── infrastructure/ # EXTERNAL ADAPTERS
│ ├── http/ # Web frameworks (Gin/Fiber)
│ ├── persistence/ # Databases (PostgreSQL/InMemory)
│ └── external/ # External services
└── cmd/ # ENTRY POINTS (dependency wiring)
```

## 📋 Golden Rules

### ✅ DOs
- Define interfaces in `core/ports/` before implementations
- Keep `core/` free from external dependencies
- Use Dependency Injection for all external services
- Write contract tests for all interfaces
- Export/import data in standard formats (JSON/CSV)

### ❌ DON'Ts  
- Import from `infrastructure/` in `core/` 
- Use framework-specific types in domain models
- Put business logic in adapters
- Create vendor-specific database schemas
- Hardcode API keys or endpoints

## 📚 Development Philosophy

### Documentation-First Approach
- **Documentation == Code**: Документация имеет тот же приоритет, что и код
- **Living Documentation**: Документы обновляются параллельно с кодом
- **Session Handover**: Каждая сессия начинается с обновления документации
- **AI Context**: Документация обеспечивает контекст для AI агентов

### Testing-Driven Development  
- **Test Results are Documentation**: Результаты тестов фиксируются в документации
- **Reproduction Steps**: Проблемы документируются с шагами воспроизведения
- **Progress Tracking**: Статус выполнения фиксируется после каждой сессии

## 🔧 Implementation Patterns

### Domain Entities
```go
// ✅ GOOD: Pure Go, no external deps
package domain

type Ticket struct {
    ID          string
    Subject     string
    Content     string
    Category    Category
    Priority    Priority
    CreatedAt   time.Time
}

type Category string
type Priority int

const (
    PriorityLow Priority = iota
    PriorityHigh
)
```

### Ports (Interfaces)
```go
// ✅ GOOD: Interface defines contract
package ports

type TicketRepository interface {
    Save(ticket *domain.Ticket) error
    FindByID(id string) (*domain.Ticket, error)
    FindByQuery(query TicketQuery) ([]domain.Ticket, error)
}

type EmailGateway interface {
    PollMessages() ([]EmailMessage, error) 
    Send(to, subject, body string) error
}
```
### Adapters (Implementations)
```go
// ✅ GOOD: Adapter knows about external world
package infrastructure

type PostgresTicketRepository struct {
    db *sql.DB
}

func (r *PostgresTicketRepository) Save(ticket *domain.Ticket) error {
    // Maps domain entity to database schema
    // Handles PostgreSQL-specific operations
}
```
## 🧪 Testing Strategy
### Unit Tests (Core)
```go
func TestTicketService(t *testing.T) {
    repo := NewInMemoryTicketRepository() // Fake implementation
    service := NewTicketService(repo)
    
    // Test pure business logic
    ticket, err := service.CreateTicket("Test", "Content")
    assert.NoError(t, err)
    assert.Equal(t, "Test", ticket.Subject)
}
```
### Contract Tests
```go
// Tests that ALL implementations satisfy interface
func TestTicketRepositoryContract(t *testing.T, repo ports.TicketRepository) {
    ticket := &domain.Ticket{ID: "test"}
    err := repo.Save(ticket)
    require.NoError(t, err)
    
    found, err := repo.FindByID("test")
    require.NoError(t, err)
    assert.Equal(t, ticket, found)
}
```
## 🔍 Validation & Compliance

### Automated Architecture Checks
The project includes automated scripts to enforce architectural rules:

```bash
# Validate Hexagonal Architecture compliance
./scripts/architecture_audit.sh

# Full validation suite
./scripts/full_validation.sh
```
**These scripts ensure:**
- Core layer has no infrastructure dependencies
- Domain models are pure (no external imports)
- All ports have implementations
- Code compiles without errors

## 📚 Migration & Configuration
### Configuration Structure
```yaml
email:
  provider: "imap"  # or "smtp", "api"
  imap:
    server: "imap.gmail.com"
    username: "support@company.com"
  api:
    base_url: "https://api.resend.com"
    api_key: "${EMAIL_API_KEY}"

ai:
  provider: "qwen"  # or "openai", "local"
  qwen:
    model_path: "./models/qwen3-4b"
  openai:
    base_url: "http://localhost:8080/v1"
```

## 🚨 Common Anti-patterns
### ❌ Business Logic in Adapters
```go
// ❌ BAD: Business logic in infrastructure
func (r *PostgresRepo) CreateTicket(subject, content string) error {
    // Classification logic here - WRONG!
    category := ai.Classify(content) // ❌ AI call in repository
    priority := calculatePriority(content) // ❌ Business logic in adapter
}
```
### ❌ Framework Types in Domain
```go
// ❌ BAD: Gin dependency in domain
type Ticket struct {
    ID      string
    Context *gin.Context // ❌ Framework type in entity
}
```
### ❌ Direct External Calls in Core
```go
// ❌ BAD: Direct API call in service
func (s *TicketService) Process(ticket *Ticket) error {
    resp, err := http.Post("https://api.openai.com/...") // ❌
    // ...
}
```

## 🚨 Performance & Scalability Considerations

### Email Module - Large Mailbox Handling
**Issue:** IMAP операции могут зависать на почтовых ящиках с 2500+ сообщений
**Solution Pattern:**
- Использовать таймауты для всех внешних операций
- Реализовать пагинацию для больших наборов данных
- Добавлять прогресс-логгирование для длительных операций
- Использовать context для cancellation

**Implementation:**
```go
type IMAPConfig struct {
    OperationTimeout time.Duration `yaml:"operation_timeout"`
    PageSize        int           `yaml:"page_size"`
    MaxMessages     int           `yaml:"max_messages"`
}
```

**Maintainer**: URMS-OS Architecture Committee  
**Last Updated**: 2025-10-17