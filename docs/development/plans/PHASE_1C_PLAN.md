# План разработки: Email Module - Phase 1C
# Development Plan: Email Module - Phase 1C

## 📋 Метаданные
- **Этап**: Phase 1C - Production Integration & Deployment
- **Статус**: 📋 ЗАПЛАНИРОВАНО
- **Предыдущий этап**: Phase 1B - IMAP Poller & Integration Testing
- **Дата начала**: 2025-10-16
- **Ожидаемая длительность**: 5-7 дней

## 🎯 Цели этапа
Интеграция email модуля в основное приложение URMS-OS и подготовка к production развертыванию.

## ⚠️ Active Issues
| Issue | Priority | Status | Blocked Tasks |
|-------|----------|---------|---------------|
| [#1](https://github.com/audetv/urms/issues/1) - IMAP Hang on Large Mailboxes | HIGH 🔴 | INVESTIGATING | Task 2 |

## 📋 Задачи Phase 1C

### Задача 1: Main Application Integration
- [ ] Интеграция PostgresEmailRepository в основное приложение
- [ ] Обновление dependency injection в cmd/api
- [ ] Настройка конфигурации через environment variables
- [ ] Создание фабрики для выбора репозитория (InMemory/PostgreSQL)

### 🚨 Задача 2: Comprehensive Testing & Validation - UPDATED
- [ ] **Реализация полноценного MIME парсера** (замена заглушки)
- [ ] **Нагрузочное тестирование** обработки 1000+ сообщений
- [ ] **End-to-end тесты** полного цикла от IMAP до БД
- [ ] **Тестирование восстановления** после сетевых сбоев
- [ ] **Бенчмарки производительности** критических операций
- [ ] 🔴 **FIX: IMAP таймауты и пагинация** для больших почтовых ящиков
- [ ] 🔴 **FIX: Structured logging** прогресса обработки
- [ ] 🔴 **FIX: Context cancellation** для длительных операций

### Задача 3: Comprehensive Logging & Observability - UPDATED
- [ ] Интеграция structured logging (zerolog/logrus)
- [ ] Добавление correlation IDs для трассировки запросов
- [ ] Настройка log levels и форматов
- [ ] Добавление метрик производительности и мониторинга
- [ ] 🔴 **FIX: Логирование прогресса IMAP операций**

### Задача 4: Configuration Management
- [ ] Создание централизованной конфигурационной системы
- [ ] Поддержка environment variables и config files
- [ ] Валидация конфигурации при старте приложения
- [ ] Создание конфигурационных шаблонов для разных сред

### Задача 5: HTTP API Development
- [ ] Создание REST API для управления email каналами
- [ ] Реализация endpoints для проверки статуса email провайдеров
- [ ] Добавление API для ручного запуска email обработки
- [ ] Создание документации API (OpenAPI/Swagger)

### Задача 6: Production Deployment & Performance
- [ ] Создание Dockerfile для production
- [ ] Настройка health checks в docker-compose
- [ ] Конфигурация для Kubernetes deployment
- [ ] Оптимизация connection pooling и кэширования

## 🔧 Технические спецификации

### Dependency Injection Structure
```go
// cmd/api/main.go
func main() {
    // Выбор репозитория на основе конфигурации
    var repo ports.EmailRepository
    if config.UsePostgreSQL {
        repo = postgres.NewPostgresEmailRepository(db)
    } else {
        repo = inmemory.NewInMemoryEmailRepo()
    }
    
    // Создание сервисов
    emailService := services.NewEmailService(imapAdapter, repo, ...)
}
```

### Configuration Structure
```yaml
database:
  provider: "postgres"  # or "inmemory"
  postgres:
    dsn: "${DATABASE_URL}"
    max_connections: 20

email:
  imap:
    server: "${IMAP_SERVER}"
    username: "${IMAP_USERNAME}"
    poll_interval: "30s"
    
logging:
  level: "info"
  format: "json"
```

### API Endpoints
```
GET  /api/v1/health          # System health status
GET  /api/v1/email/status    # Email module status
POST /api/v1/email/poll      # Manual email polling
GET  /api/v1/email/channels  # List email channels
```

## 📊 Критерии успеха

### Функциональные требования
- Автоматический запуск email обработки при старте приложения
- Конфигурируемый выбор репозитория (InMemory/PostgreSQL)
- Полная интеграция health checks в основной API
- Structured logging с трассировкой запросов

### Production Readiness
- Готовность к deployment в Kubernetes
- Настроенные health checks и liveness probes
- Production-ready конфигурация
- Оптимизированные настройки производительности

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
backend/cmd/api/main.go
backend/internal/config/config.go
backend/internal/infrastructure/http/api.go
backend/internal/infrastructure/logging/
backend/deployments/docker/Dockerfile
```

### Зависимости:
- Требуется работающая PostgreSQL база данных
- Необходимы тестовые IMAP учетные записи
- Нужен настроенный logging infrastructure

### Связанные документы:
- [Отчет Phase 1B](./2025-10-16_email_module_phase1b_completion.md)
- [Спецификация Email модуля](../../specifications/EMAIL_MODULE_SPEC.md)
- [Архитектурные принципы](../../../ARCHITECTURE_PRINCIPLES.md)

## 📦 Deliverables

### Code Deliverables
- Интегрированное основное приложение URMS-OS
- Production Docker configuration
- Comprehensive API documentation
- Performance optimization patches

### Documentation Deliverables
- Deployment guide
- API reference
- Configuration guide
- Troubleshooting manual

---
**Maintainer**: URMS-OS Architecture Committee  
**Last Updated**: 2025-10-16