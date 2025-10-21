# Phase 3B Plan - Email Optimization & Customer Service

## 🎯 Цели
Завершить оптимизацию email обработки и реализовать базовый Customer Service

## 📋 Приоритетные задачи

### Задача 1: IMAP Search Optimization (ВЫСОКИЙ ПРИОРИТЕТ)
- [ ] Диагностика проблемы с поиском 5-го письма в цепочке
- [ ] Оптимизация IMAP search criteria для полного покрытия
- [ ] Реализация UID-based пагинации для больших почтовых ящиков
- [ ] Fallback стратегии для разных IMAP провайдеров

### Задача 2: Code Quality & Testing (ВЫСОКИЙ ПРИОРИТЕТ)  
- [ ] Написание unit tests для email threading
- [ ] Написание unit tests для MIME парсера
- [ ] Удаление устаревших методов (fallback functions)
- [ ] Консолидация дублирующей логики

### Задача 3: Customer Service Implementation (СРЕДНИЙ ПРИОРИТЕТ)
- [ ] Исправить CustomerService.ListCustomers
- [ ] Реализовать поиск клиентов по email/имени
- [ ] Добавить базовые операции CRUD для клиентов
- [ ] Интеграция клиентов с задачами

### Задача 4: Email→Message Mapping Optimization (НИЗКИЙ ПРИОРИТЕТ)
- [ ] Анализ текущей логики 1:1 mapping
- [ ] Оптимизация для полного соответствия Jira/Zendesk model
- [ ] Улучшение хронологического упорядочивания