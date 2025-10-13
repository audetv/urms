## 📋 Спецификация проекта: Unified Request Management System (URMS) - Open Source Edition

### 🎯 Project ID: URMS-OS-v1

### 📅 Version: 1.0

### 🏷️ Status: In Development

### 📜 License: Apache 2.0 / MIT (to be decided)

### 🌟 Model: qwen3-4B

---

## 📖 Executive Summary

**URMS-OS** - полностью open-source унифицированная система управления заявками. Построена на открытых технологиях с возможностью самостоятельного развертывания и коммерческой поддержки.

## 🎯 Primary Objectives

### Core Value Propositions

1. 100% Open Source - полная прозрачность и возможность самостоятельного использования
2. Unified Inbox - агрегация всех обращений в едином интерфейсе
3. Self-Hosted Friendly - минимальная зависимость от SaaS сервисов
4. AI-Powered Intelligence - автоматическая классификация на основе open-source моделей
5. Vendor Lock-in Free - миграция между провайдерами без потери функциональности

## 🏗️ Architectural Foundation

### Technology Stack (Open Source Only)

| Layer | Technology | Purpose |
| --- | --- | --- |
| Backend | Go (Gin/Fiber) | Core API, business logic |
| Frontend | Vue 3 + TypeScript + Pinia | User interface |
| Database | PostgreSQL + Redis | Primary storage + caching |
| Search | ManticoreSearch 13.13.0+ | Full-text + vector search |
| AI/ML | Python + FastAPI + qwen3-4B | Classification & NLP |
| Message Queue | Redis Streams | Async processing |
| Email | SMTP/IMAP + MTA (Postfix) | Email processing |

### Key Domains & Entities

```go
// CORE DOMAINS
Domain: TicketManagement
  - Ticket (core entity)
  - Message (communication)
  - Thread (conversation grouping)

Domain: CustomerManagement  
  - Customer
  - ContactChannel
  - ProjectMembership

Domain: IntegrationLayer
  - ChannelAdapter (email, telegram, web, api)
  - WebhookHandler
  - SyncService

Domain: IntelligenceLayer
  - ClassifierService (qwen3-4B)
  - VectorSearchService (Manticore)
  - AnalyticsEngine
```

## 🔄 Business Processes Flow

### 1. Request Intake Process

```
Incoming Message → Channel Adapter → Validation → AI Classification (qwen3-4B) → 
Ticket Creation → Unified Inbox → Operator Assignment
```

### 2. Communication Process

```
Operator Reply → Message Processing → Channel-Specific Delivery →
Customer Response → Thread Update → Status Sync
```

### 3. Intelligence Process

```
Text Processing → Vector Embeddings → Semantic Search (Manticore) →
Similar Ticket Matching → Knowledge Base Linking
```

## 📊 Core Features Matrix

### Phase 1: Foundation

- Unified Inbox UI
- **Email Channel (SMTP/IMAP + Postfix)**
- Basic Ticket Management
- Customer Profiles
- PostgreSQL Schema

### Phase 2: Intelligence

- ManticoreSearch Integration (Vector + Full-text)
- AI Classification Service (qwen3-4B)
- Advanced Filtering & Semantic Search
- Git Integration

### Phase 3: Multi-Channel

- Telegram Bot
- Web Forms API
- Application Logs Ingestion
- REST API Gateway

### Phase 4: Analytics & Optimization

- Dashboard & Reporting
- Knowledge Base Integration
- Performance Analytics
- Automated Workflows

## 🔌 Integration Specifications (Open Source)

### Email Channel (Priority 1) - Open Source Approach

**Pattern**: Direct SMTP/IMAP + Optional MTA (Postfix)  
**Architecture**:

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   Email Server  │    │   URMS Email     │    │   Ticket Chat   │
│(Postfix/Dovecot)│◄──▶│   Gateway        │◄──▶│   Interface     │
│  or IMAP Server │    │                  │    │                 │
└─────────────────┘    │ ┌──────────────┐ │    └─────────────────┘
                       │ │  IMAP Poller │ │
                       │ │  + Parser    │ │
                       │ └──────────────┘ │
                       │ ┌──────────────┐ │
                       │ │  SMTP Sender │ │
                       │ │  + Templates │ │
                       │ └──────────────┘ │
                       │ ┌──────────────┐ │
                       │ │  Thread      │ │
                       │ │  Manager     │ │
                       │ └──────────────┘ │
                       └──────────────────┘
```

**Implementation Options:**

1. **IMAP Polling** - периодическая проверка почтовых ящиков
2. **MDA (Mail Delivery Agent)** - интеграция с Postfix через pipe
3. **LMTP** - Local Mail Transfer Protocol для прямой доставки

### ManticoreSearch Integration

**Capabilities:**

- Full-text search с морфологией
- **Vector search** для semantic matching
- Real-time indexing
- SQL-like query language

**Use Cases:**

- Поиск похожих заявок по semantic similarity
- Автоматическая категоризация на основе vector embeddings
- Поиск в базе знаний по смысловому сходству

### AI Services (qwen3-4B)

**Model:** qwen3-4B - последняя версия с улучшенными возможностями  
**Deployment:**

- Local inference с оптимизацией (GGML/GGUF)
- GPU acceleration при наличии
- Batch processing для эффективности

**Applications:**

- Классификация заявок
- Автоматическое тегирование
- Определение тональности и срочности
- Генерация ответов-заготовок

## 🎨 UI/UX Principles

### Design Tenets

1. **Unified Inbox First** - все обращения в едином интерфейсе
2. **Channel Transparency** - оператор не видит разницы между каналами
3. **Self-Service Configuration** - настройка каналов через UI
4. **Progressive Disclosure** - сложные настройки доступны по требованию

## 📈 Success Metrics

### Technical Excellence

- Zero paid service dependencies
- Comprehensive documentation coverage
- Easy local development setup
- Modular architecture for extensions

### Community Growth

- Contributor-friendly codebase
- Clear extension points documentation
- Demo deployment availability
- Active community support channels

## 🚀 Development Philosophy

### Open Source Principles

1. **No Vendor Lock-in** - все компоненты заменяемы
2. **Transparent Architecture** - понятные и документированные интеграции
3. **Community Driven** - обратная связь определяет roadmap
4. **Enterprise Ready** - возможность коммерческой поддержки и кастомизации

### Deployment Options

1. **Docker Compose** - для быстрого старта
2. **Kubernetes Helm** - для production развертываний
3. **Bare Metal** - с пошаговыми инструкциями

## 🔧 Email Implementation Strategy

### Phase 1A: Basic IMAP/SMTP

```go
package email

type IMAPEmailHandler struct {
    config   IMAPConfig
    poller   *IMAPPoller
    parser   *EmailParser
}

type SMTPService struct {
    client   *smtp.Client
    templates *EmailTemplates
}

// Polling-based approach для начала
```

### Phase 1B: Advanced MTA Integration

```bash
# Postfix integration через pipe
# /etc/postfix/master.cf
urms    unix  -       n       n       -       -       pipe
  flags=FR user=urms argv=/opt/urms/bin/email-handler ${sender} ${recipient}
```

### Phase 1C: Webmail Bridge (Optional)

- Roundcube integration
- или собственный простой webmail интерфейс

---

## 🔄 How to Use This Specification

For AI Context: "Reference URMS-OS Project Spec v1.0 - [Section Name]"  
For Development: "Implementing according to URMS-OS Spec - [Feature]"  
For Open Source:"Following open-source principles of URMS-OS..."  

## 📝 Next Steps Tracking

### Current Phase: Foundation Development

- Project scaffolding (Go + Vue)
- Database schema implementation
- Basic Ticket CRUD API
- Unified Inbox UI skeleton

### Next Phase: Email Channel (Open Source)

- IMAP/SMTP client implementation
- Email parsing and normalization
- Thread management system
- Attachment handling

---

**Document Maintainer**: Project Architect  
**License**: Open Source (To be finalized)  
**Last Updated**: ${current_date}  
