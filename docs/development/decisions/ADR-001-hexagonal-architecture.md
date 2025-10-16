# ADR 001: Выбор Hexagonal Architecture

## Status
ACCEPTED

## Context
Нужна архитектура, которая обеспечит "No Vendor Lock-in" и легкую тестируемость.

## Decision
Использовать Hexagonal Architecture (Ports & Adapters) с четким разделением:
- Core слой (бизнес-логика)
- Infrastructure слой (адаптеры)
- Порты (интерфейсы)

## Consequences
### Положительные
- ✅ Легкая замена внешних сервисов
- ✅ Упрощенное тестирование
- ✅ Четкое разделение ответственности

### Отрицательные  
- ⚠️ Усложнение начальной настройки
- ⚠️ Больше boilerplate кода