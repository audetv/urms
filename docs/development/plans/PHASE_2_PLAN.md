# План разработки: Task Management - Phase 2

## 📋 Метаданные
- **Этап**: Phase 2 - Task Management Core 
- **Статус**: 🟡 IN PROGRESS (85% завершено)
- **Дата начала**: 2025-10-18
- **Текущий фокус**: Email-Task Integration + REST API

## ⚠️ Активные проблемы
[См. ISSUE_MANAGEMENT.md для деталей](./ISSUE_MANAGEMENT.md)

| Issue | Priority | Status | Blocked Tasks |
|-------|----------|---------|---------------|
| MarkAsRead test expectation | MEDIUM | Investigating | Email integration tests |
| PostgreSQL Migration | LOW | Pending | Production deployment |
| InMemory Repository Message Persistence | LOW | Investigating | Full message testing |

## 📋 ЗАДАЧИ ЭТАПА

### Задача 1: Domain Model & Core Architecture ✅ ВЫПОЛНЕНО
- [x] Design Task entity with extensible structure
- [x] Define Task status lifecycle (Открыта, В работе, Решена, Закрыта)
- [x] Design Customer/Organization hierarchy
- [x] Create domain validation rules and business logic
- [x] Implement Priority system (Низкий, Средний, Высокий, Критический)

### Задача 2: Database & Repository Layer 🟡 ВЫПОЛНЕНО ЧАСТИЧНО
- [x] Design repository interfaces in core/ports/
- [x] Create InMemory repositories for development
- [ ] Design PostgreSQL schema for tasks and dictionaries
- [ ] Implement PostgreSQL repository implementations
- [ ] Create database migration scripts
- [ ] Implement dictionary tables for statuses, categories, tags

### Задача 3: Business Logic Integration 🟡 ВЫПОЛНЕНО ЧАСТИЧНО
- [x] Create TaskService with business operations
- [x] Implement CustomerService with profile management  
- [x] Add validation and business rules
- [x] Extend MessageProcessor for automatic task creation ✅
- [ ] Implement email-thread to task linking (Message-ID/In-Reply-To) ❌
- [x] Add basic assignment rules engine ✅

### Задача 4: REST API Implementation ⏳ НАЧАТЬ
- [ ] Design REST endpoints for task operations
- [ ] Implement HTTP handlers with validation
- [ ] Add search, filtering and pagination
- [ ] Create API documentation

### Задача 5: Email-Task Integration ✅ ВЫПОЛНЕНО
- [x] Automatic task creation from incoming emails ✅
- [x] Thread management and conversation linking ✅ (БАЗОВАЯ)
- [x] Basic assignment logic ✅

## 🔄 ПЕРЕНОС ЧАСТИ ФУНКЦИОНАЛА НА PHASE 3

### Отложенные задачи (перенесены на Phase 3):

**Thread-ID Based Task Linking** 
- **Причина**: Требует изменений в core интерфейсах (TaskQuery) и сложной логики поиска
- **Влияние**: Временное создание новых задач для каждого email (приемлемо для MVP)
- **План реализации**: 
  - Phase 3: Добавить SourceMeta в TaskQuery
  - Phase 3: Реализовать поиск по message_id, in_reply_to в репозиториях
  - Phase 3: Добавить индексацию для производительности

**Полная реализация авто-назначения**
- **Причина**: Требует AI/ML классификации (Phase 4)
- **Текущее решение**: Базовая логика назначения работает

## 🎯 ОБНОВЛЕННЫЙ СТАТУС:

**Phase 2 Core Objectives ДОСТИГНУТЫ:**
- ✅ Универсальная Task архитектура
- ✅ Email-to-Task автоматическое создание  
- ✅ Бизнес-логика и валидация
- ✅ InMemory репозитории для разработки
- ✅ Комплексное тестирование

**Phase 2.5 Objectives (СЕЙЧАС):**
- 🚀 REST API для управления задачами
- 🚀 Frontend-ready интерфейсы

**Phase 3 Objectives (БУДУЩЕЕ):**
- 🔄 Thread-ID поиск и linking
- 🔄 Улучшенное авто-назначение
- 🔄 PostgreSQL миграция

## 🎯 ТЕКУЩИЙ ФОКУС: Phase 2.5 - Email Integration & REST API

### Приоритет 1: Email-Task Integration (1-2 дня)
```go
// Расширение существующего MessageProcessor
type AdvancedMessageProcessor struct {
    taskService ports.TaskService
    customerService ports.CustomerService
    logger ports.Logger
}

func (p *AdvancedMessageProcessor) ProcessIncomingEmail(ctx context.Context, email domain.EmailMessage) error {
    // 1. Поиск существующего таска по ThreadID/In-Reply-To
    // 2. Если не найден - создание нового таска через TaskService
    // 3. Добавление сообщения в существующий таск
    // 4. Автоматическое назначение по правилам
}
```

### Приоритет 2: REST API Implementation (2-3 дня)
```go
// Новые файлы для создания:
internal/infrastructure/http/handlers/task_handler.go
internal/infrastructure/http/handlers/customer_handler.go
internal/infrastructure/http/middleware/
internal/infrastructure/http/dto/
```

### Приоритет 3: PostgreSQL Preparation (ОТЛОЖЕНО)
- Реализовать когда будем готовы к production
- InMemory достаточно для текущей разработки

## 📊 КРИТЕРИИ УСПЕХА

### Функциональные требования
- [x] Ручное создание задач через сервисы
- [ ] Автоматическое создание задач из входящих email
- [x] Поиск и фильтрация задач по статусам/приоритетам
- [x] Базовое назначение исполнителей
- [x] Управление статусами жизненного цикла
- [ ] REST API для всех операций

### Качественные требования  
- [x] 100% покрытие domain моделей тестами
- [x] InMemory репозитории для разработки
- [x] Структурированное логирование всех операций
- [x] Архитектурное соответствие Hexagonal Principles
- [ ] PostgreSQL репозитории для production

## 🔧 ТЕХНИЧЕСКИЕ СПЕЦИФИКАЦИИ

### Email-Task Integration Flow
```
Incoming Email → MessageProcessor → 
    Find Existing Task (by Thread-ID) → 
        If Found: Add Message to Task
        If Not Found: Create New Task → 
            Auto-assign based on rules → 
                Update Email status
```

### REST API Endpoints
```go
// Task Management
GET    /api/tasks              # List tasks with filtering
POST   /api/tasks              # Create task
GET    /api/tasks/{id}         # Get task details
PUT    /api/tasks/{id}         # Update task
DELETE /api/tasks/{id}         # Delete task

// Task Operations
PUT    /api/tasks/{id}/status  # Change status
PUT    /api/tasks/{id}/assign  # Assign task
POST   /api/tasks/{id}/messages     # Add message
POST   /api/tasks/{id}/internal-note # Add internal note
```

## 🚀 СЛЕДУЮЩИЕ ЭТАПЫ

### Phase 3: Frontend & UI
- [ ] Unified Inbox interface
- [ ] Task Management UI
- [ ] Customer profiles
- [ ] Real-time updates

### Phase 4: AI Integration & PostgreSQL
- [ ] PostgreSQL migration and repositories
- [ ] Automatic classification
- [ ] Smart assignment
- [ ] Semantic search

## 📝 ПРИМЕЧАНИЯ ДЛЯ РАЗРАБОТКИ

### Стратегия базы данных:
**Текущая**: InMemory для быстрой разработки  
**Будущая**: PostgreSQL при готовности к production

### Архитектурные решения:
- Email модуль уже работает и готов к интеграции
- TaskService полностью реализован и протестирован
- InMemory репозитории достаточно для MVP
- PostgreSQL можно добавить без изменения бизнес-логики

---
**Maintainer**: URMS-OS Architecture Committee  
**Last Updated**: 2025-10-18
**Next Task**: Email-Task Integration
