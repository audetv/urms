# Phase 3C Plan - IMAP Search Optimization & 5/5 Email Threading

## 📋 Метаданные
- **Этап**: Phase 3C - IMAP Search Optimization & Complete Email Threading
- **Статус**: 🏗️ ACTIVE DEVELOPMENT (0% завершено)
- **Дата начала**: 2025-10-22
- **Предыдущий этап**: Phase 3B - Headers Optimization & Architecture Refactoring ✅

## 🎯 Цели этапа
Решить проблему поиска всех 5 писем в цепочке и достичь полной группировки email переписки в одной задаче.

## 🎯 КРИТИЧЕСКАЯ ПРОБЛЕМА
**Текущее состояние**: 4/5 писем в цепочке → 1 задача  
**Целевое состояние**: 5/5 писем в цепочке → 1 задача

## 📋 ПРИОРИТЕТНЫЕ ЗАДАЧИ

### 🔥 Задача 1: IMAP Search Optimization (ВЫСОКИЙ ПРИОРИТЕТ)

#### 1.1 Диагностика проблемы поиска
- [ ] Анализ текущих IMAP search criteria
- [ ] Логирование полного процесса поиска
- [ ] Определение почему 5-е письмо не находится
- [ ] Тестирование с различными IMAP провайдерами

#### 1.2 Enhanced Search Strategies
- [ ] Расширение временного диапазона поиска (90+ дней)
- [ ] Реализация UID-based пагинации как fallback
- [ ] Комбинированные критерии поиска (Subject + Message-ID + References)
- [ ] Thread-aware поиск по normalized subject

#### 1.3 Provider-Specific Optimizations
- [ ] Стратегии для Gmail IMAP
- [ ] Стратегии для Yandex Mail
- [ ] Стратегии для Outlook/Exchange
- [ ] Универсальные fallback механизмы

### 🔥 Задача 2: Complete Email→Message Mapping (ВЫСОКИЙ ПРИОРИТЕТ)

#### 2.1 1:1 Message Mapping
- [ ] Исправление логики 5 писем → 4 сообщения
- [ ] Обеспечение полного соответствия email→message
- [ ] Тестирование с различными сценариями переписки

#### 2.2 Message History Preservation
- [ ] Сохранение полной хронологии переписки
- [ ] Правильное упорядочивание сообщений по времени
- [ ] Отображение полной истории в UI

### 🔧 Задача 3: Technical Debt Cleanup (СРЕДНИЙ ПРИОРИТЕТ)

#### 3.1 Code Quality Improvements
- [ ] Удаление convertToDomainMessageWithBodyFallback
- [ ] Удаление других устаревших методов
- [ ] Консолидация дублирующей логики парсинга
- [ ] Рефакторинг унаследованного кода

#### 3.2 Enhanced Testing
- [ ] Интеграционные тесты для IMAP search
- [ ] Тесты для различных email провайдеров
- [ ] Тесты для edge cases threading scenarios

### 🔧 Задача 4: Customer Service Completion (СРЕДНИЙ ПРИОРИТЕТ)

#### 4.1 Customer Management
- [ ] Исправление CustomerService.ListCustomers
- [ ] Реализация полноценного поиска клиентов
- [ ] Базовые CRUD операции для клиентов
- [ ] Интеграция клиентов с задачами

## 🏗️ АРХИТЕКТУРНЫЕ КОМПОНЕНТЫ ДЛЯ РЕАЛИЗАЦИИ

### 1. Enhanced IMAP Search Engine
```go
// Расширенная логика поиска в IMAPAdapter
type EnhancedIMAPSearch struct {
    TimeframeExtensions time.Duration
    UIDBasedFallback    bool
    CombinedCriteria    bool
    ProviderSpecific    map[string]SearchStrategy
}
```

### 2. Thread Detection Service
```go
// Сервис для обнаружения полных цепочек писем
type ThreadDetectionService struct {
    SearchStrategies []SearchStrategy
    FallbackMechanisms []FallbackStrategy
}
```

### 3. Provider-Specific Adapters
```go
// Адаптеры для различных email провайдеров
type ProviderAdapter interface {
    OptimizeSearch(criteria *imap.SearchCriteria)
    GetSearchLimits() SearchLimits
}
```

## 🎯 КРИТЕРИИ УСПЕХА PHASE 3C

### Функциональные требования
- [ ] 5/5 писем в цепочке группируются в одной задаче
- [ ] Полная история переписки сохраняется и отображается
- [ ] 1:1 соответствие email→message
- [ ] Работа с различными IMAP провайдерами

### Качественные требования  
- [ ] Comprehensive IMAP search стратегии
- [ ] Полное тестовое покрытие новой функциональности
- [ ] Удаление technical debt из предыдущих этапов
- [ ] Customer Service полностью функционален

### Performance Requirements
- [ ] Поиск всех писем цепочки за < 30 секунд
- [ ] Поддержка почтовых ящиков с 10,000+ сообщений
- [ ] Стабильная работа при network issues

## 🔧 ТЕХНИЧЕСКИЕ СПЕЦИФИКАЦИИ

### IMAP Search Optimizations:
- **Временной диапазон**: 90+ дней вместо 3
- **Критерии поиска**: Message-ID + In-Reply-To + References + Subject
- **Fallback стратегии**: UID-based pagination, provider-specific
- **Кэширование**: Thread detection results

### Thread Detection Logic:
- **Primary**: Message-ID/References matching
- **Secondary**: Subject-based threading (normalized)
- **Tertiary**: Temporal proximity + participant matching
- **Confidence scoring** для ambiguous cases

## 🧪 ТЕСТИРОВАНИЕ И ВАЛИДАЦИЯ

### Test Scenarios:
1. **Simple Thread** - 5 последовательных писем
2. **Complex Thread** - ветвления, ответы разным людям
3. **Cross-Provider** - письма с разных email провайдеров
4. **Large Mailbox** - поиск в ящике с 10,000+ сообщений
5. **Network Issues** - таймауты, reconnection логика

### Validation Metrics:
```json
{
  "thread_completion_rate": "100% (5/5 emails)",
  "search_performance": "< 30 seconds",
  "provider_coverage": ["gmail", "yandex", "outlook", "generic"],
  "test_coverage": "> 80%"
}
```

## ⚠️ РИСКИ И МИТИГАЦИЯ

### Высокие риски:
- **IMAP provider limitations** - разные реализации IMAP
- **Performance degradation** - расширенные поисковые запросы
- **False positives** в thread detection

### Стратегии митигации:
- Постепенное внедрение с feature flags
- Extensive logging и мониторинг
- Fallback к существующей функциональности
- A/B testing поисковых стратегий

---
**Maintainer**: URMS-OS Architecture Committee  
**Created**: 2025-10-22  
**Estimated Completion**: 2025-10-25  
**Dependencies**: Phase 3B Headers Optimization ✅
