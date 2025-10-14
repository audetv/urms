# URMS - Unified Request Management System

🌐 **Open Source система управления заявками из различных источников**

## 🎯 О проекте

URMS - это унифицированная система для приема и обработки заявок из email, Telegram, веб-форм и других каналов с AI-классификацией.

## 🎯 Быстрый старт

```bash
git clone https://github.com/audetv/urms.git
cd urms
```

# Текущий статус разработки:
cat docs/development/CURRENT_STATUS.md

## 📚 Документация

### 🏗️ Спецификации

- [Общая спецификация проекта](./docs/specifications/URMS_SPECIFICATION.md)
- [Архитектура системы](./docs/specifications/ARCHITECTURE.md)
- [Email модуль](./docs/specifications/EMAIL_MODULE_SPEC.md)
- [Модели данных](./docs/specifications/DATA_MODELS.md)

### 🔄 Разработка

- [Текущий статус](./docs/development/CURRENT_STATUS.md)
- [Дорожная карта](./docs/development/ROADMAP.md)
- [Отчеты о разработке](./docs/development/reports/INDEX.md)
- [Архитектурные решения](./docs/development/DECISIONS.md)

### 🚀 Деплоймент
- [Руководство по развертыванию](./docs/development/DEVELOPMENT_GUIDE.md) 

## 🏗️ Технологический стек
- Backend: Go (Gin/Fiber)
- Frontend: Vue 3 + TypeScript + Pinia
- Database: PostgreSQL + Redis
- Search: ManticoreSearch (full-text + vector)
- AI: qwen3-4B для классификации
- Email: IMAP/SMTP (Exchange поддержка)

## 📄 Лицензия
### Licensed under the Apache License 2.0