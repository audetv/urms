# ADR 002: IMAP Timeout and Pagination Strategy

## Status
**PROPOSED**  
**Created**: 2025-10-16  
**Related Issue**: [IMAP Hang on Large Mailboxes](../issues/2025-10-16_imap_hang_large_mailboxes.md)

## Context
В Phase 1B обнаружена проблема: IMAP операции зависают при работе с почтовыми ящиками, содержащими большое количество сообщений (2545+).

**Проблема**: 
- IMAP операции не имеют таймаутов
- Отсутствует пагинация для больших наборов сообщений
- Нет механизма отмены длительных операций
- Не отслеживается прогресс обработки

**Требования из Phase 1B Completion Report**:
- Нагрузочное тестирование обработки 1000+ сообщений
- Production readiness для больших почтовых ящиков
- Мониторинг и observability

## Decision
Реализовать комплексную стратегию управления IMAP операциями:

### 1. Configurable Timeouts
```go
type IMAPConfig struct {
    // Таймауты для различных операций
    ConnectTimeout    time.Duration `yaml:"connect_timeout"`
    LoginTimeout      time.Duration `yaml:"login_timeout"`
    FetchTimeout      time.Duration `yaml:"fetch_timeout"`
    OperationTimeout  time.Duration `yaml:"operation_timeout"`
    
    // Пагинация для больших почтовых ящиков
    PageSize          int           `yaml:"page_size"`
    MaxMessagesPerPoll int          `yaml:"max_messages_per_poll"`
    
    // Настройки повторных попыток
    MaxRetries        int           `yaml:"max_retries"`
    RetryDelay        time.Duration `yaml:"retry_delay"`
}
```

### 2. UID-based Pagination
```go
// Обработка больших почтовых ящиков частями
func (p *IMAPPoller) pollMessagesPaginated(ctx context.Context, lastUID uint32) error {
    for {
        select {
        case <-ctx.Done():
            return ctx.Err() // Respect cancellation
        default:
            messages, err := p.gateway.FetchMessages(lastUID, p.config.PageSize)
            if err != nil {
                return err
            }
            
            if len(messages) == 0 {
                break // No more messages
            }
            
            // Process batch
            if err := p.processMessageBatch(messages); err != nil {
                return err
            }
            
            lastUID = messages[len(messages)-1].UID
        }
    }
}
```

### 3. Context-Based Cancellation
```go
// Все IMAP операции должны принимать context
type EmailGateway interface {
    FetchMessages(ctx context.Context, sinceUID uint32, limit int) ([]EmailMessage, error)
    Connect(ctx context.Context) error
    // ...
}
```

### 4. Progress Monitoring
```go
// Structured logging прогресса
type PollProgress struct {
    TotalMessages   int       `json:"total_messages"`
    Processed       int       `json:"processed"`
    CurrentBatch    int       `json:"current_batch"`
    EstimatedTime   string    `json:"estimated_time"`
}

func (p *IMAPPoller) logProgress(progress PollProgress) {
    log.Info().
        Int("total", progress.TotalMessages).
        Int("processed", progress.Processed).
        Int("batch", progress.CurrentBatch).
        Str("eta", progress.EstimatedTime).
        Msg("IMAP polling progress")
}
```

## Consequences

### Positive
- ✅ **Устойчивость**: Система не зависает на больших почтовых ящиках
- ✅ **Контроль**: Администраторы могут настраивать таймауты под свои нужды
- ✅ **Мониторинг**: Видимость прогресса через structured logging
- ✅ **Отмена**: Пользователи могут прервать длительные операции
- ✅ **Производительность**: Пагинация снижает потребление памяти

### Negative
- ⚠️ **Сложность**: Усложнение логики IMAP клиента
- ⚠️ **Конфигурация**: Больше параметров для настройки
- ⚠️ **Тестирование**: Требуется тестирование различных сценариев таймаутов

### Neutral
- 🔄 **Производительность**: Небольшой оверхед от проверки контекста
- 🔄 **Память**: Пагинация снижает пиковое использование памяти

## Compliance with Architecture Principles

### Hexagonal Architecture
- ✅ Конфигурация инкапсулирована в инфраструктурном слое
- ✅ Таймауты реализованы в адаптерах, а не в ядре
- ✅ Интерфейсы EmailGateway поддерживают context

### No Vendor Lock-in
- ✅ Стратегия применима к любым IMAP провайдерам
- ✅ Конфигурация позволяет адаптироваться к разным окружениям
- ✅ Легко заменить на другую реализацию с таймаутами

## Implementation Plan

### Phase 1C - Task 2 (Updated)
- [ ] Добавить IMAPConfig с таймаутами и пагинацией
- [ ] Реализовать UID-based пагинацию в IMAPPoller
- [ ] Интегрировать context во все IMAP операции
- [ ] Добавить structured logging прогресса
- [ ] Написать нагрузочные тесты для больших почтовых ящиков

### Testing Strategy
- Unit tests для пагинационной логики
- Integration tests с mock IMAP сервером
- Load tests с 10k+ сообщениями
- Chaos testing для проверки таймаутов

## Alternatives Considered

### Alternative 1: No Timeouts (REJECTED)
- ❌ Система зависает на больших почтовых ящиках
- ❌ Нет контроля над длительными операциями

### Alternative 2: Global Timeout Only (REJECTED)  
- ❌ Недостаточно гибкости для разных операций
- ❌ Не решает проблему пагинации

### Alternative 3: Async Processing (DEFERRED)
- 🔶 Сложнее в реализации и отладке
- 🔶 Может быть рассмотрено в будущих версиях

## References
- [Phase 1B Completion Report](../reports/2025-10-16_email_module_phase1b_completion.md)
- [IMAP Hang Issue](../issues/2025-10-16_imap_hang_large_mailboxes.md)
- [Phase 1C Plan](../plans/PHASE_1C_PLAN.md)
- [Architecture Principles](../../specifications/ARCHITECTURE_PRINCIPLES.md)

---
**Decision Authors**: URMS-OS Architecture Committee  
**Reviewers**: [List of reviewers]  
**Supersedes**: [None]
