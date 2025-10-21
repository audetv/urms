# План разработки: Phase 3A - Email Threading & Bug Fixes

## 📋 Метаданные
- **Этап**: Phase 3A - Email Threading & Production Readiness
- **Статус**: 🔄 В РАБОТЕ (85% завершено)
- **Ожидаемая дата начала**: 2025-10-21
- **Предыдущий этап**: Phase 2.5 - REST API & Email Integration ✅

## 🎯 Цели этапа
Исправление критических проблем email threading и подготовка системы к production развертыванию с полнофункциональной цепочкой обработки писем.

## ✅ РЕШЕННЫЕ ПРОБЛЕМЫ:

### Проблема 1: Email Threading Not Working ❌ → ✅ РЕШЕНО
**Решение**: Исправлено сохранение SourceMeta в domain.Task и работа matching алгоритма
**Результат**: 5 связанных писем создают 1 задачу с полной историей переписки

### Проблема 0: Background Tasks Not Starting ❌ → ✅ РЕШЕНО
**Решение**: BackgroundTaskManager с os.Stdout.Sync()
**Результат**: Email poller запускается ДО HTTP сервера

## 🔄 ТЕКУЩИЙ СТАТУС PHASE 3A

| Проблема | Статус | Решение |
|----------|---------|---------|
| Email Threading Not Working | ✅ **РЕШЕНО** | SourceMeta + matching алгоритм |
| Message Content Parsing | 🔄 **В РАБОТЕ** | Парсинг тела письма |
| CustomerService.ListCustomers Empty | ⏳ **ОЖИДАЕТ** | После message parsing |
| InMemory Message Persistence | ⏳ **ОЖИДАЕТ** | После message parsing |

## 📋 ОСТАЮЩИЕСЯ ЗАДАЧИ

### Задача 2: Message Content Parsing (1 день)
- [ ] Исправить парсинг тела письма в MessageProcessor
- [ ] Реализовать извлечение полного текста сообщений
- [ ] Обновить `buildMessageContent` для использования email.BodyText
- [ ] Протестировать с реальным контентом писем

### Задача 3: CustomerService Bug Fixes (1 день)  
- [ ] Исправить CustomerService.ListCustomers
- [ ] Реализовать полноценный поиск клиентов
- [ ] Протестировать API endpoints

## 🎯 КРИТЕРИИ УСПЕХА PHASE 3A

### Функциональные требования
- [x] Email цепочки группируются в одной задаче
- [ ] Полный текст сообщений сохраняется и отображается
- [ ] Customer list API возвращает корректные данные
- [ ] Все endpoints работают без validation errors

### Качественные требования
- [ ] 100% тестовое покрытие нового функционала
- [ ] Backward compatibility сохранена
- [ ] Производительность поиска не деградирует

---
**Maintainer**: URMS-OS Architecture Committee  
**Created**: 2025-10-20  
**Last Updated**: 2025-10-21