# URMS - Unified Request Management System

🌐 **Open Source система управления заявками из различных источников**

## 🎯 О проекте

URMS - это унифицированная система для приема и обработки заявок из email, Telegram, веб-форм и других каналов с AI-классификацией.

## 🏗️ Архитектура

- **Backend**: Go (Gin/Fiber)
- **Frontend**: Vue 3 + TypeScript  
- **Database**: PostgreSQL + Redis
- **Search**: ManticoreSearch (full-text + vector)
- **AI**: qwen3-4B для классификации

[URMS-OS Architecture Principles — архитектурные принципы ](./ARCHITECTURE_PRINCIPLES.md)

## 📚 Документация

- [Спецификация проекта](./docs/specifications/URMS_SPECIFICATION.md)
- [Отчеты о разработке](./docs/development/DEVELOPMENT_REPORTS.md)
- [Дорожная карта](./docs/development/ROADMAP.md)
- [Email модуль](./docs/specifications/EMAIL_MODULE_SPEC.md)

## 🚀 Быстрый старт

```bash
# Клонирование репозитория
git clone https://github.com/audetv/urms.git
cd urms/backend

# Запуск тестового IMAP клиента
export URMS_IMAP_USERNAME="your_email"
export URMS_IMAP_PASSWORD="your_password"
go run cmd/test-imap/main.go
```

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

# Например устанавливаем соединение с яндекс почтой
export URMS_IMAP_USERNAME="you-support-email@yandex.ru"
export URMS_IMAP_PASSWORD="your_app_password" # пароль приложения https://yandex.ru/support/id/ru/authorization/app-passwords.html
export URMS_IMAP_SERVER="imap.yandex.ru"

# Запускаем тест
go run cmd/test-imap/main.go
```
## 📄 Лицензия
### Licensed under the Apache License 2.0