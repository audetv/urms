# План разработки: Email Module - Phase 1C
# Development Plan: Email Module - Phase 1C

## 📋 Метаданные
- **Этап**: Phase 1C - Production Integration & Deployment
- **Статус**: 🔄 В ПРОЦЕССЕ
- **Предыдущий этап**: Phase 1B - IMAP Poller & Integration Testing
- **Дата начала**: 2025-10-16
- **Ожидаемая длительность**: 5-7 дней

## 🎯 Цели этапа
Интеграция email модуля в основное приложение URMS-OS и подготовка к production развертыванию.

## 🧪 Результаты тестирования API Server

### ✅ Успешный запуск (2025-10-17)
```bash
# API Server running on :8085
curl http://localhost:8085/health
# Response: {"status":"UP","services":{"imap_email_gateway":...}}

# IMAP Connection successful
# Mailbox: 2562 messages, 210 unseen
```

### 🚨 Выявленные проблемы
1. **IMAP Hanging Risk**: Подтвержден сценарий больших почтовых ящиков
2. **Message Processing**: Poller подключен но не обрабатывает сообщения
3. **No Timeout Strategy**: Критический риск для production

## ⚠️ Active Issues
| Issue | Priority | Status | Blocked Tasks |
|-------|----------|---------|---------------|
| [#1](https://github.com/audetv/urms/issues/1) - IMAP Hang on Large Mailboxes | CRITICAL 🔴 | CONFIRMED | Task 2.1, 2.2, 2.3 |
| Message Processing Inactive | HIGH 🟡 | INVESTIGATING | Task 2.2 |
| No Structured Logging | MEDIUM 🟠 | PLANNED | Task 3.1 |

## 🎯 Обновленные приоритеты Phase 1C

### 🔴 Критические (Blocking)
- [ ] **Задача 2.1**: Реализация IMAP Timeout Strategy (ADR-002)
- [ ] **Задача 2.2**: Активация обработки сообщений в IMAP Poller
- [ ] **Задача 2.3**: Context integration для cancellation

### 🟡 Высокий приоритет  
- [ ] **Задача 3.1**: Structured logging (zerolog integration)
- [ ] **Задача 3.2**: Message persistence verification
- [ ] **Задача 3.3**: PostgreSQL migration integration

### 🟠 Средний приоритет
- [ ] **Задача 4.1**: Comprehensive Testing & Validation
- [ ] **Задача 4.2**: Configuration Management
- [ ] **Задача 4.3**: HTTP API Development

## 📋 Детализация критических задач

### Задача 2.1: IMAP Timeout Strategy (ADR-002)
- [ ] Обновление IMAPConfig с таймаутами
- [ ] Реализация UID-based пагинации
- [ ] Context integration во все IMAP операции
- [ ] Structured logging прогресса обработки

### Задача 2.2: Активация обработки сообщений
- [ ] Активировать FetchMessages в IMAP Poller
- [ ] Интегрировать MessageProcessor для бизнес-логики
- [ ] Добавить сохранение в репозиторий
- [ ] Реализовать обработку вложений

### Задача 2.3: Context Integration
- [ ] Добавить context во все IMAP операции
- [ ] Реализовать cancellation для длительных операций
- [ ] Добавить timeout handling в EmailService

## 🔧 Технические спецификации

### IMAP Timeout Configuration
```yaml
email:
  imap:
    connect_timeout: "30s"
    login_timeout: "15s" 
    fetch_timeout: "60s"
    operation_timeout: "120s"
    page_size: 100
    max_messages_per_poll: 500
```

### Context Integration Pattern
```go
type EmailGateway interface {
    FetchMessages(ctx context.Context, criteria FetchCriteria) ([]domain.EmailMessage, error)
    Connect(ctx context.Context) error
    HealthCheck(ctx context.Context) error
}
```

## 📊 Критерии успеха

### Функциональные требования
- Автоматический запуск email обработки при старте приложения
- Безопасная обработка почтовых ящиков с 5000+ сообщений
- Конфигурируемые таймауты для всех IMAP операций
- Structured logging с трассировкой прогресса

### Production Readiness
- Готовность к deployment в Kubernetes
- Настроенные health checks и liveness probes
- Production-ready конфигурация таймаутов
- Мониторинг длительных операций

## 🚀 Следующие этапы

### Phase 2: Ticket Management Integration
- Интеграция email сообщений с системой тикетов
- Автоматическое создание тикетов из email
- Связывание ответов с существующими тикетами

### Phase 3: Multi-Channel Support
- Реализация Telegram Bot адаптера
- Добавление Web Forms API
- Поддержка Application Logs ingestion

## 📝 Примечания для разработки

### Ключевые файлы для реализации:
```text
backend/internal/config/config.go
backend/internal/infrastructure/email/imap_adapter.go
backend/internal/infrastructure/email/imap_poller.go
backend/internal/core/services/email_service.go
```

### Зависимости:
- Требуется работающая IMAP учетная запись с большим почтовым ящиком
- Необходимо тестирование с 5000+ сообщениями
- Нужна настройка мониторинга прогресса обработки

### Связанные документы:
- [Отчет Phase 1B](./2025-10-16_email_module_phase1b_completion.md)
- [ADR-002: IMAP Timeout Strategy](../decisions/ADR-002-imap-timeout-strategy.md)
- [Тестовый отчет API Server](./2025-10-17_api_server_testing.md)
- [Архитектурные принципы](../../../ARCHITECTURE_PRINCIPLES.md)

## 📦 Deliverables

### Code Deliverables
- IMAP Timeout Strategy implementation
- Activated message processing pipeline
- Context-integrated email operations
- Production-ready configuration

### Documentation Deliverables
- Updated ADR-002 with implementation details
- Performance testing results
- Production deployment guide
- Monitoring and troubleshooting manual

---
**Maintainer**: URMS-OS Architecture Committee  
**Last Updated**: 2025-10-17