# Email Threading Algorithm Documentation

## 🎯 Обзор
Алгоритм группировки связанных email сообщений в единые задачи.

## 🔧 Техническая реализация

### Flow обработки входящего письма:
```
Incoming Email 
    → IMAP Adapter (extractAllHeaders) 
    → Domain EmailMessage (with Thread IDs)
    → MessageProcessor.findExistingTaskByThread()
    → TaskService.FindBySourceMeta()
    → TaskRepository.matchesSourceMeta()
    → Create/Update Task
```

### Ключевые компоненты:

#### 1. Thread Identification
```go
// Критерии поиска существующих задач
searchMeta := map[string]interface{}{
    "message_id":  email.MessageID,    // Высокий приоритет
    "in_reply_to": email.InReplyTo,    // Высокий приоритет  
    "references":  email.References,   // Низкий приоритет
}
```

#### 2. Matching Algorithm
Приоритетность matching:
1. **Message-ID** - точное совпадение
2. **In-Reply-To** - совпадение с message_id существующей задачи
3. **References** - пересечение с references существующей задачи

#### 3. SourceMeta Structure
```json
{
  "message_id": "<unique@message.id>",
  "in_reply_to": "<parent@message.id>", 
  "references": ["<ref1>", "<ref2>", "..."],
  "headers": {"X-IMAP-UID": ["12345"]}
}
```

## 📊 Пример работы

**Цепочка из 5 писем:**
```
Письмо 1: Message-ID: A, References: []
Письмо 2: Message-ID: B, In-Reply-To: A, References: [A]
Письмо 3: Message-ID: C, In-Reply-To: B, References: [A, B]  
Письмо 4: Message-ID: D, In-Reply-To: C, References: [A, B, C]
Письмо 5: Message-ID: E, In-Reply-To: D, References: [A, B, C, D]
```

**Результат:** 1 задача с 5 сообщениями

## 🚀 Следующие улучшения

### Требуется реализовать:
1. **Парсинг тела письма** - извлечение полного текста сообщения
2. **HTML to Text конвертация** - для писем в HTML формате
3. **Attachment handling** - сохранение вложений
4. **Quoted text detection** - удаление цитируемого текста

### Планируемые оптимизации:
- Кэширование результатов поиска
- Batch processing для больших почтовых ящиков
- AI-классификация для автоматического назначения
