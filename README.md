## Структура проекта

```text
backend/
├── cmd/
│   └── test-imap/
│       └── main.go
├── internal/
│   └── email/
│       ├── imapclient/     # переименовано из imap
│       │   ├── client.go
│       │   ├── config.go
│       │   └── utils.go
│       ├── models/
│       │   └── message.go
│       └── service.go
├── go.mod
└── go.sum
```

## 🚀 Запускаем тест:

```bash
cd backend

# Устанавливаем credentials через environment variables
export URMS_IMAP_USERNAME="support@yourcompany.com"
export URMS_IMAP_PASSWORD="your_password"
export URMS_IMAP_SERVER="outlook.office365.com"  # опционально

# Запускаем тест
go run cmd/test-imap/main.go
```