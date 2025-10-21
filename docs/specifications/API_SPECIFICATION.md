Запускаем приложение и тестируем! 🚀

```bash
cd /mnt/work/audetv/urms/backend
go run ./cmd/api/
```

В другом терминале тестируем эндпоинты:

## 🧪 Health Checks:
```bash
curl http://localhost:8085/health
curl http://localhost:8085/ready
curl http://localhost:8085/live
```

## 🎯 Task API:
```bash
# Создаем задачу поддержки
curl -X POST http://localhost:8085/api/v1/tasks/support \
  -H "Content-Type: application/json" \
  -d '{
    "subject": "Тест API: Проблема с доступом",
    "description": "Не могу зайти в систему, выдает ошибку аутентификации",
    "customer_id": "test-customer-api",
    "priority": "high",
    "category": "technical"
  }'

# Получаем список задач
curl http://localhost:8085/api/v1/tasks
```

## 👤 Customer API:
```bash
# Создаем клиента
curl -X POST http://localhost:8085/api/v1/customers \
  -H "Content-Type: application/json" \
  -d '{
    "name": "API Тестовый Клиент",
    "email": "api-test@example.com",
    "phone": "+79991234567"
  }'

# Ищем или создаем клиента
curl "http://localhost:8085/api/v1/customers/find-or-create?email=findme@example.com&name=Найденный%20Клиент"
```

## 🔄 Legacy Endpoint:
```bash
curl -X POST http://localhost:8085/test-imap
```
