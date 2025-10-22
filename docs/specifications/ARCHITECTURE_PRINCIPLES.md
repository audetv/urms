# URMS-OS Architecture Principles
**Version: 1.1** | **Project: Unified Request Management System**  
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

## 🎯 DEVELOPMENT PHILOSOPHY: QUALITY OVER SPEED

### 🏗️ Active Development Mindset
- **NO BACKWARD COMPATIBILITY** - APIs and architecture WILL change
- **NO PRODUCTION DEPLOYMENT** - We can make breaking changes freely
- **COMPREHENSIVE SOLUTIONS** - No quick fixes or temporary workarounds
- **ARCHITECTURAL PURITY** - Quality of design over development speed
- **UNLIMITED ITERATIONS** - As many sessions as needed to get it right

### 🔧 Implementation Strategy
- **Refactor aggressively** when architecture violations are found
- **Delete and rewrite** instead of patching problematic code  
- **Take time for proper design** - no rushing to "working state"
- **Document architectural decisions** thoroughly
- **Write tests for new patterns** before widespread implementation

## 🏗️ ARCHITECTURAL PATTERNS FROM PHASE 3B

### Pattern: EmailHeaders Value Object
```go
// Domain-centric representation of essential data only
type EmailHeaders struct {
    MessageID  string          // Business identifier
    InReplyTo  string          // Threading data
    References []string        // Threading data  
    Subject    string          // Business content
    From       EmailAddress    // Business entity
    To         []EmailAddress  // Business entities
    // ONLY business-significant headers - no technical metadata
}
```
**Principles:**
- Value Objects contain ONLY business-significant data
- No technical headers (Received, DKIM, etc.)
- Immutable and validated on creation
- Domain layer purity maintained

### Pattern: HeaderFilter Service
```go
// Infrastructure service for data filtering and sanitization
type HeaderFilter struct {
    logger ports.Logger
}

func (f *HeaderFilter) FilterEssentialHeaders(
    rawHeaders map[string][]string,
) (*domain.EmailHeaders, error) {
    // Extracts only business-essential headers
    // Removes sensitive information (IP, tracking, auth)
    // Preserves threading data integrity
}
```
**Principles:**
- Separation of concerns: filtering logic in infrastructure
- Security: automatic removal of sensitive data
- Performance: 70-80% data reduction achieved
- Threading integrity: all business data preserved

### Principle: Systematic Interface Updates
**When modifying interfaces:**
1. **Update ALL implementations** simultaneously
2. **Update ALL test mocks** and contracts
3. **Update ALL constructors** and dependency injections
4. **Verify compilation** across entire codebase

**Example from Phase 3B:**
```go
// Adding SearchThreadMessages to EmailGateway required:
- IMAPAdapter implementation
- HealthCheckAdapter delegation  
- 7+ test mock updates
- MessageProcessor constructor updates
- Main.go dependency wiring
```

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
- **Update ALL implementations** when interfaces change
- **Create Value Objects** for domain data representations

### ❌ DON'Ts  
- Import from `infrastructure/` in `core/` 
- Use framework-specific types in domain models
- Put business logic in adapters
- Create vendor-specific database schemas
- Hardcode API keys or endpoints
- **Make partial interface updates** - update ALL or NONE
- **Store technical metadata** in domain models

## 📚 Development Philosophy

### Documentation-First Approach
- **Documentation == Code**: Документация имеет тот же приоритет, что и код
- **Living Documentation**: Документы обновляются параллельно с кодом
- **Session Handover**: Каждая сессия начинается с обновления документации
- **AI Context**: Документация обеспечивает контекст для AI агентов
- **Architectural Decisions**: Фиксировать принятые паттерны и принципы

### Testing-Driven Development  
- **Test Results are Documentation**: Результаты тестов фиксируются в документации
- **Reproduction Steps**: Проблемы документируются с шагами воспроизведения
- **Progress Tracking**: Статус выполнения фиксируется после каждой сессии
- **Systematic Test Updates**: Все тесты обновляются при architectural changes

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

### Value Objects (NEW PATTERN)
```go
// ✅ GOOD: Domain-centric data representation
package domain

type EmailHeaders struct {
    MessageID  string
    InReplyTo  string  
    References []string
    Subject    string
    From       EmailAddress
    // ONLY business data - no technical headers
}

func NewEmailHeaders(email *EmailMessage) (*EmailHeaders, error) {
    // Validation and business logic only
    // No external dependencies
}
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

### Filter Services (NEW PATTERN)
```go
// ✅ GOOD: Infrastructure service for data processing
package infrastructure

type HeaderFilter struct {
    logger ports.Logger
}

func (f *HeaderFilter) FilterEssentialHeaders(
    email *domain.EmailMessage,
) (*domain.EmailHeaders, error) {
    // Transforms external data to domain representation
    // Removes sensitive/technical information
    // Maintains data integrity for business operations
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

### Systematic Update Testing
```go
// When interfaces change, verify ALL implementations
func TestEmailGatewaySystematicUpdate(t *testing.T) {
    implementations := []ports.EmailGateway{
        &IMAPAdapter{},
        &HealthCheckAdapter{},
        &TestEmailGateway{},
        // ALL implementations must be tested
    }
    
    for _, impl := range implementations {
        t.Run(fmt.Sprintf("%T", impl), func(t *testing.T) {
            // Verify new method exists and doesn't panic
            _, err := impl.SearchThreadMessages(ctx, criteria)
            // Basic contract verification
        })
    }
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

# Verify systematic interface updates
./scripts/interface_consistency.sh
```

**These scripts ensure:**
- Core layer has no infrastructure dependencies
- Domain models are pure (no external imports)
- All ports have implementations
- Code compiles without errors
- **All interface implementations are consistent**

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

### ❌ Partial Interface Updates
```go
// ❌ BAD: Only some implementations updated
type EmailGateway interface {
    FetchMessages(ctx context.Context, criteria FetchCriteria) ([]EmailMessage, error)
    SearchThreadMessages(ctx context.Context, criteria ThreadSearchCriteria) ([]EmailMessage, error) // NEW
}

// ❌ Some adapters missing new method
type OldAdapter struct {} // Missing SearchThreadMessages
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

### Data Optimization Patterns
**Headers Optimization (Phase 3B):**
- 70-80% data reduction in source_meta
- Automatic sensitive data removal
- Business data integrity preserved
- No performance degradation

**Maintainer**: URMS-OS Architecture Committee  
**Last Updated**: 2025-10-22
**Version Notes**: Added architectural patterns from Phase 3B - EmailHeaders Value Object, HeaderFilter Service, Systematic Interface Updates