# План разработки: Ticket Management - Phase 2

## 📋 Метаданные
- **Этап**: Phase 2 - Ticket Management Integration  
- **Статус**: 📋 ЗАПЛАНИРОВАНО
- **Предыдущий этап**: Phase 1C - Email Module Complete ✅
- **Дата начала**: 2025-10-18
- **Ожидаемая длительность**: 7-10 дней

## 🎯 Цели этапа
Создание полной системы управления тикетами с интеграцией email модуля, автоматическим созданием тикетов из входящих сообщений и REST API для управления.

## ⚠️ Активные проблемы
| Issue | Priority | Status | Blocked Tasks |
|-------|----------|---------|---------------|
| MarkAsRead test expectation | MEDIUM | Investigating | - |
| PostgreSQL Migration | LOW | Pending | Production deployment |

## 📋 Задачи этапа

### Задача 1: Domain Model & Core Architecture (2 дня)
- [ ] Design Ticket entity with extensible structure
- [ ] Define Ticket status lifecycle (Открыта, В работе, Решена, Закрыта)
- [ ] Design Customer/Organization hierarchy
- [ ] Create domain validation rules and business logic
- [ ] Implement Priority system (Низкий, Средний, Высокий, Критический)

### Задача 2: Database & Repository Layer (2 дня)  
- [ ] Design PostgreSQL schema for tickets and dictionaries
- [ ] Implement TicketRepository interface in core/ports/
- [ ] Create InMemoryTicketRepository for development
- [ ] Implement dictionary tables for statuses, categories, tags

### Задача 3: Business Logic Integration (2 дня)
- [ ] Extend MessageProcessor for automatic ticket creation
- [ ] Implement email-thread to ticket linking (Message-ID/In-Reply-To)
- [ ] Create TicketService with business operations
- [ ] Add basic assignment rules engine

### Задача 4: REST API Implementation (2 дня)
- [ ] Design REST endpoints for ticket operations
- [ ] Implement HTTP handlers with validation
- [ ] Add search, filtering and pagination
- [ ] Create API documentation

### Задача 5: Email-Ticket Integration (1 день)
- [ ] Automatic ticket creation from incoming emails
- [ ] Thread management and conversation linking
- [ ] Basic assignment logic

## 🔧 Технические спецификации

### Доменная модель Ticket
```go
// internal/core/domain/ticket.go
package domain

type Ticket struct {
    ID           string
    Subject      string
    Description  string
    Status       TicketStatus    // Справочник: Открыта, В работе, Решена, Закрыта
    Priority     Priority        // Справочник: Низкий, Средний, Высокий, Критический
    Category     string          // Справочник (расширяемый)
    Tags         []string        // Произвольные теги
    Assignee     string          // Назначенный исполнитель
    Reporter     string          // Автор заявки
    Participants []Participant   // Участники (исполнители, наблюдатели)
    Source       TicketSource    // Email, Telegram, WebForm, etc.
    SourceMeta   map[string]interface{} // Мета-информация источника
    CustomerID   string          // Связь с клиентом/организацией
    CreatedAt    time.Time
    UpdatedAt    time.Time
}

type TicketStatus string
const (
    StatusOpen      TicketStatus = "open"      // Открыта
    StatusInProgress TicketStatus = "in_progress" // В работе  
    StatusResolved  TicketStatus = "resolved"  // Решена
    StatusClosed    TicketStatus = "closed"    // Закрыта
)

type Priority string
const (
    PriorityLow      Priority = "low"      // Низкий
    PriorityMedium   Priority = "medium"   // Средний
    PriorityHigh     Priority = "high"     // Высокий
    PriorityCritical Priority = "critical" // Критический
)

type TicketSource string
const (
    SourceEmail    TicketSource = "email"
    SourceTelegram TicketSource = "telegram" 
    SourceWebForm  TicketSource = "web_form"
    SourceAPI      TicketSource = "api"
)

type Participant struct {
    UserID    string
    Role      ParticipantRole // Assignee, Reviewer, Watcher
    JoinedAt  time.Time
}

type ParticipantRole string
const (
    RoleAssignee ParticipantRole = "assignee"
    RoleReviewer ParticipantRole = "reviewer" 
    RoleWatcher  ParticipantRole = "watcher"
)
```

### Customer/Organization модель
```go
// internal/core/domain/customer.go
package domain

type Customer struct {
    ID           string
    Name         string
    Email        string
    Organization *Organization
    Projects     []ProjectMembership
    CreatedAt    time.Time
}

type Organization struct {
    ID   string
    Name string
}

type ProjectMembership struct {
    ProjectID string
    Role      string
}
```

### Repository интерфейсы
```go
// internal/core/ports/repositories.go
package ports

type TicketRepository interface {
    Save(ctx context.Context, ticket *domain.Ticket) error
    FindByID(ctx context.Context, id string) (*domain.Ticket, error)
    FindByQuery(ctx context.Context, query TicketQuery) ([]domain.Ticket, error)
    Update(ctx context.Context, ticket *domain.Ticket) error
    Delete(ctx context.Context, id string) error
}

type TicketQuery struct {
    Status    []domain.TicketStatus
    Priority  []domain.Priority  
    Assignee  string
    CustomerID string
    Source    []domain.TicketSource
    Tags      []string
    Offset    int
    Limit     int
}
```

### Расширенный MessageProcessor
```go
// internal/infrastructure/email/advanced_message_processor.go
type AdvancedMessageProcessor struct {
    ticketService ports.TicketService
    logger        ports.Logger
}

func (p *AdvancedMessageProcessor) ProcessIncomingEmail(ctx context.Context, email domain.EmailMessage) error {
    // 1. Поиск существующего тикета по ThreadID
    // 2. Если не найден - создание нового тикета
    // 3. Добавление сообщения в тикет
    // 4. Автоматическое назначение по правилам
    // 5. Обновление статуса
}
```

### REST API Endpoints
```go
// internal/infrastructure/http/handlers/ticket_handler.go
package handlers

// GET /api/tickets - список тикетов с фильтрацией
// POST /api/tickets - создание тикета
// GET /api/tickets/{id} - получение тикета
// PUT /api/tickets/{id} - обновление тикета  
// GET /api/tickets/{id}/messages - сообщения тикета
// POST /api/tickets/{id}/messages - добавление сообщения
// PUT /api/tickets/{id}/status - изменение статуса
// PUT /api/tickets/{id}/assignee - назначение исполнителя
```

## 📊 Критерии успеха

### Функциональные требования
- [ ] Автоматическое создание тикетов из входящих email
- [ ] Ручное создание тикетов через API
- [ ] Поиск и фильтрация тикетов по статусам/приоритетам
- [ ] Базовое назначение исполнителей
- [ ] Управление статусами жизненного цикла

### Качественные требования  
- [ ] 100% покрытие domain моделей тестами
- [ ] InMemory репозиторий для разработки
- [ ] Структурированное логирование всех операций
- [ ] Архитектурное соответствие Hexagonal Principles

## 🚀 Следующие этапы

### Phase 3: Frontend & UI
- [ ] Unified Inbox интерфейс
- [ ] Ticket Management UI
- [ ] Customer profiles
- [ ] Real-time updates

### Phase 4: AI Integration
- [ ] Автоматическая классификация
- [ ] Умное назначение
- [ ] Semantic search

## 📝 Примечания для разработки

### Ключевые файлы для реализации:
```text
internal/core/domain/ticket.go
internal/core/domain/customer.go  
internal/core/ports/repositories.go
internal/core/services/ticket_service.go
internal/infrastructure/persistence/inmemory/ticket_repository.go
internal/infrastructure/http/handlers/ticket_handler.go
internal/infrastructure/email/advanced_message_processor.go
```

### Зависимости:
- Существующий Email Module
- Structured logging система
- Configuration management

### Стратегия базы данных:
**Рекомендация**: Начинаем с InMemory для быстрой разработки бизнес-логики, затем добавляем PostgreSQL.

**Преимущества:**
- Быстрый feedback цикл при разработке
- Легкое тестирование
- Можно отложить миграции до Phase 2.5
- Фокус на бизнес-логике, а не на инфраструктуре

## 📦 Deliverables

### Code Deliverables
- [ ] Complete Ticket domain model
- [ ] TicketService with business logic
- [ ] InMemoryTicketRepository
- [ ] REST API endpoints
- [ ] Email-Ticket integration

### Documentation Deliverables  
- [ ] API specification
- [ ] Domain model documentation
- [ ] Integration guide

---
**Maintainer**: URMS-OS Architecture Committee  
**Last Updated**: 2025-10-18