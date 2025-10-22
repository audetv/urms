
# Phase 3C - Next Steps Plan

## 🎯 ПРИОРИТЕТ 1: Критические исправления

### 1.1 Удалить/Исправить /test-imap endpoint
- **Задача**: Удалить зависающий endpoint
- **Альтернатива**: Создать безопасную тестовую endpoint
- **Срок**: Немедленно

### 1.2 Анализ Initial IMAP Errors
- **Задача**: Исследовать причину SEARCH Backend errors
- **Подход**: Добавить детальное логирование ошибок IMAP
- **Цель**: Понимание почему происходят initial failures

## 🎯 ПРИОРИТЕТ 2: 5/5 Threading Solution

### 2.1 Диагностика Thread Detection
- **Задача**: Проанализировать почему 5-е письмо не находится
- **Методы**: 
  - Детальное логирование критериев поиска
  - Проверка threading данных в source_meta
  - Анализ IMAP search результатов

### 2.2 Enhanced Search Strategies
- **Задача**: Реализовать дополнительные стратегии поиска
- **Варианты**:
  - Temporal proximity matching
  - Participant-based threading  
  - Subject normalization improvements

## 🎯 ПРИОРИТЕТ 3: Оптимизация логирования

### 3.1 Уровни логирования
- **Задача**: Перевести избыточные логи в Debug уровень
- **Критерий**: Info только для бизнес-событий, Debug для технических

### 3.2 Structured Logging улучшения
- **Задача**: Улучшить читаемость логов
- **Методы**: Группировка related logs, сокращение дублирования

## 🎯 ПРИОРИТЕТ 4: Performance & Stability

### 4.1 Connection Management
- **Задача**: Улучшить обработку reconnection
- **Цель**: Уменьшить initial errors

### 4.2 Memory & Resource Optimization
- **Задача**: Профилирование использования ресурсов
- **Фокус**: Large mailbox handling
