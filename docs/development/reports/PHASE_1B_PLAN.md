# План разработки: Email Module - Phase 1B

## 📋 Метаданные
- **Этап**: Phase 1B - IMAP Poller & Integration Testing
- **Статус**: 📋 ЗАПЛАНИРОВАНО
- **Предыдущий этап**: Phase 1A - Hexagonal Architecture Refactoring
- **Дата начала**: 2025-10-15
- **Ожидаемая длительность**: 3-5 дней

## 🎯 Цели этапа
Создать полнофункциональный IMAP Poller с автоматическим опросом почтовых ящиков и полной обработкой RFC 5322 сообщений.

## 📋 Задачи Phase 1B

### Задача 1: IMAP Poller Implementation
- [ ] Реализовать `IMAPPoller` с UID-based polling
- [ ] Добавить обработку новых сообщений с отслеживанием последнего UID
- [ ] Реализовать механизм восстановления после сбоев
- [ ] Добавить конфигурацию интервала опроса

### Задача 2: Complete Message Parsing
- [ ] Расширить парсинг для получения тела сообщения (text/HTML)
- [ ] Реализовать обработку MIME частей и вложений
- [ ] Добавить извлечение всех RFC заголовков
- [ ] Реализовать нормализацию email адресов

### Задача 3: Contract Tests
- [ ] Создать контрактные тесты для `EmailGateway`
- [ ] Создать контрактные тесты для `EmailRepository` 
- [ ] Реализовать тесты для `MessageProcessor`
- [ ] Добавить интеграционные тесты полного цикла

### Задача 4: PostgreSQL Integration
- [ ] Создать миграции базы данных для email моделей
- [ ] Реализовать `PostgresEmailRepository`
- [ ] Добавить индексы для поиска по MessageID и ThreadID
- [ ] Реализовать мягкое удаление сообщений

### Задача 5: Error Handling & Monitoring
- [ ] Добавить исчерпывающую обработку ошибок IMAP
- [ ] Реализовать retry логику для временных сбоев
- [ ] Добавить метрики и мониторинг обработки
- [ ] Создать health checks для email модуля

## 🔧 Технические спецификации

### IMAP Poller Architecture
```go
type IMAPPoller struct {
    gateway     ports.EmailGateway
    repo        ports.EmailRepository
    lastUID     uint32
    pollInterval time.Duration
}

func (p *IMAPPoller) Start(ctx context.Context) error
func (p *IMAPPoller) pollNewMessages(ctx context.Context) error
func (p *Poller) processMessageBatch(messages []domain.EmailMessage) error
```
### Database Schema
```sql
-- Таблица email_messages
CREATE TABLE email_messages (
    id UUID PRIMARY KEY,
    message_id VARCHAR(500) UNIQUE NOT NULL,
    in_reply_to VARCHAR(500),
    thread_id VARCHAR(500),
    from_email VARCHAR(255) NOT NULL,
    to_emails JSONB,
    subject TEXT,
    body_text TEXT,
    body_html TEXT,
    direction VARCHAR(20) NOT NULL,
    source VARCHAR(50) NOT NULL,
    processed BOOLEAN DEFAULT FALSE,
    processed_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);

-- Индексы
CREATE INDEX idx_email_messages_message_id ON email_messages(message_id);
CREATE INDEX idx_email_messages_thread_id ON email_messages(thread_id);
CREATE INDEX idx_email_messages_processed ON email_messages(processed);
```
### Message Processing Flow
```text
IMAP Poller → Fetch Messages → Parse RFC 5322 → 
Save to Repository → Process Business Logic → 
Update Message Status → Mark as Read (optional)
```
## 📊 Критерии успеха
### Функциональные требования
- Автоматический опрос почтового ящика каждые 30 секунд
- Обработка 1000+ сообщений без потери производительности
- Полный парсинг RFC 5322 сообщений с вложениями
- Сохранение всей информации в PostgreSQL
- Восстановление после перезапуска службы
### Качественные требования
- 90%+ покрытие кода тестами
- Полная обработка ошибок IMAP протокола
- Конфигурируемые параметры опроса
- Логирование всех критических операций
## 🚀 Следующие этапы
### Phase 1C: SMTP Integration & Email Sending
- Реализация SMTP адаптера для отправки email
- Система шаблонов ответов
- Очередь исходящих сообщений
### Phase 2: Ticket Management Integration
- Интеграция email сообщений с системой тикетов
- Автоматическое создание тикетов из email
- Связывание ответов с существующими тикетами
## 📝 Примечания для разработки
### Ключевые файлы для реализации:
```text
backend/internal/infrastructure/email/imap_poller.go
backend/internal/infrastructure/persistence/email/postgres_repo.go
backend/internal/core/ports/email_contract_test.go
backend/migrations/001_create_email_tables.sql
```
### Зависимости:
- Требуется работающая PostgreSQL база данных
- Необходимы тестовые IMAP учетные записи
- Нужны mock серверы для тестирования
### Связанные документы:

- [Отчет Phase 1A](./2025-10-14_email_module_phase1a_refactoring.md)
- [Спецификация Email модуля](../../specifications/EMAIL_MODULE_SPEC.md)
- [Архитектурные принципы](../../../ARCHITECTURE_PRINCIPLES.md)


## 📦 Список файлов для передачи в следующий чат:
Обязательные архитектурные файлы:  
📄 ARCHITECTURE_PRINCIPLES.md  
📄 AI_CODING_GUIDELINES.md  
📄 PROJECT_STRUCTURE.md  
📄 URMS_SPECIFICATION.md  

Отчеты и планы:  
📄 docs/reports/2024-01-20_email_module_phase1a_refactoring.md  
📄 docs/reports/PHASE_1B_PLAN.md  

Ключевые исходные файлы (выборочно):  
📄 internal/core/domain/email.go  
📄 internal/core/ports/email_gateway.go  
📄 internal/core/services/email_service.go  
📄 internal/infrastructure/email/imap_adapter.go  
📄 internal/infrastructure/email/imap/client.go  
📄 cmd/test-imap/main.go  


## 🎯 Готово к передаче!

**Следующий шаг:** При начале нового чата предоставить эти файлы и указать:
- "Продолжаем разработку URMS-OS Email Module"
- "Текущий этап: Phase 1B - IMAP Poller & Integration Testing"  
- "Архитектура: Hexagonal Architecture, No Vendor Lock-in"
- "Ссылаться на PHASE_1B_PLAN.md для деталей"

Теперь можно плавно передать проект! 🚀