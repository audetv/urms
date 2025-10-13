## 📧 Email Module Specification: Hybrid Architecture

Email модуль с гибридной архитектурой, поддерживающей работу через почтовый клиент и постепенный переход на web-интерфейс.


### 🎯 Business Context

- Текущий процесс: Письма → Exchange группа → Ответы из Outlook
- Целевой процесс: Постепенный переход на web-интерфейс
- Требование: Сохранение существующего workflow на период внедрения

## 🏗️ Architectural Design

### Гибридная архитектура Email обработки

```text
┌─────────────────────────────────────────────────────────────────┐
│                     EXCHANGE/OUTLOOK LAYER                      │
├─────────────────────────────────────────────────────────────────┤
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────────────┐  │
│  │   Exchange  │    │  Outlook    │    │   Other Email       │  │
│  │   Server    │◄──▶│   Clients   │    │   Clients           │  │
│  └─────────────┘    └─────────────┘    └─────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────────┐
│                     URMS EMAIL GATEWAY                          │
├─────────────────────────────────────────────────────────────────┤
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────────────┐  │
│  │  IMAP       │    │  SMTP       │    │   Email Tracking    │  │
│  │  Poller     │    │  Sender     │    │   Service           │  │
│  └─────────────┘    └─────────────┘    └─────────────────────┘  │
│           │               │                      │               │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────────────┐  │
│  │  Email      │    │  Thread     │    │   Attachment        │  │
│  │  Parser     │    │  Manager    │    │   Handler           │  │
│  └─────────────┘    └─────────────┘    └─────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────────┐
│                     URMS CORE SYSTEM                            │
├─────────────────────────────────────────────────────────────────┤
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────────────┐  │
│  │  Ticket     │    │  Unified    │    │   Web Interface     │  │
│  │  Service    │    │  Inbox      │    │   (Vue)             │  │
│  └─────────────┘    └─────────────┘    └─────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
```

## 🔧 Technical Implementation Strategy

### Вариант 1: IMAP Polling + SMTP (Рекомендуемый для начала)

```go
// Core Email Service Structure
package email

type HybridEmailService struct {
    imapClient    *IMAPClient
    smtpClient    *SMTPClient
    tracker       *EmailTracker
    threadManager *ThreadManager
}

// IMAP Configuration для Exchange
type IMAPConfig struct {
    Server   string
    Port     int
    Username string
    Password string
    Mailbox  string  // "Техническая поддержка"
    SSL      bool
}

// SMTP Configuration для отправки
type SMTPConfig struct {
    Server    string
    Port      int
    Username  string
    Password  string
    FromEmail string  // support@company.com
    FromName  string  // "Техническая поддержка"
}
```

### Вариант 2: MTA Integration (Более продвинутый)

```bash
# Postfix как MTA с pipe обработкой
# /etc/postfix/master.cf
urms-support unix  -       n       n       -       -       pipe
  flags=FR user=urms argv=/opt/urms/bin/email-processor ${sender} ${recipient}
```

## 📨 Email Tracking & Thread Management

### Система отслеживания писем

```go
package email

// Email Message Tracking
type EmailMessage struct {
    ID           uuid.UUID
    MessageID    string    // RFC Message-ID header
    InReplyTo    string    // RFC In-Reply-To header  
    References   []string  // RFC References header
    ThreadID     string    // Our internal thread ID
    From         string
    To           []string
    CC           []string
    Subject      string
    BodyHTML     string
    BodyText     string
    Attachments  []Attachment
    Direction    Direction // incoming, outgoing
    Source       string    // imap, smtp, web
    CreatedAt    time.Time
}

// Thread Management
type EmailThread struct {
    ID          string
    TicketID    uuid.UUID
    Subject     string
    Participants []Participant
    Messages    []EmailMessage
    CreatedAt   time.Time
    UpdatedAt   time.Time
}

// Tracking Techniques
type EmailTracker struct {
    methods []TrackingMethod
}

type TrackingMethod interface {
    ApplyTracking(email *OutgoingEmail) error
    DetectTracking(email *IncomingEmail) (*TrackingResult, error)
}

// Конкретные методы трекинга
type MessageIDTracking struct{}
type HeaderTracking struct{}
type EmailAddressTracking struct{}
```

### Методы трекинга писем

1. **Message-ID based** (основной)
   ```go
   // Генерация уникального Message-ID
   func GenerateMessageID(ticketID uuid.UUID, sequence int) string {
       return fmt.Sprintf("<ticket-%s-%d@urms.company.com>", 
           ticketID.String(), sequence)
   }
   ```
2. **Headers injection**
   ```go
   // Добавление служебных headers
   X-URMS-Ticket-ID: ticket_uuid
   X-URMS-Thread-ID: thread_uuid  
   List-ID: "Техническая поддержка" <support.company.com>
   ```
3. **Email address aliases** (опционально)
   ```
   ticket-{ticket_id}@support.company.com
   ```

## 🔄 Workflow Processing

### Обработка входящих писем

```go
// Incoming Email Processing Pipeline
type EmailProcessor struct {
    steps []ProcessingStep
}

type ProcessingStep interface {
    Process(email *EmailMessage) error
}

// Конкретные шаги обработки
type StepValidation struct{}
type StepDeduplication struct{} 
type StepThreadMatching struct{}
type StepTicketCreation struct{}
type StepNotification struct{}

func (p *EmailProcessor) ProcessIncoming(email *EmailMessage) error {
    for _, step := range p.steps {
        if err := step.Process(email); err != nil {
            return fmt.Errorf("step failed: %w", err)
        }
    }
    return nil
}
```

### Сопоставление цепочек писем

```go
// Thread Matching Algorithm
type ThreadMatcher struct {
    strategies []MatchingStrategy
}

type MatchingStrategy interface {
    Match(email *EmailMessage) (*EmailThread, error)
    Priority() int
}

// Стратегии сопоставления:
// 1. По Message-ID/In-Reply-To (наиболее надежный)
// 2. По специальным headers (X-URMS-Thread-ID)
// 3. По subject line (с учетом Re:, Fwd: и локализации)
// 4. По участникам переписки
// 5. По временным меткам
```

## 📊 Database Schema for Email

```sql
-- Таблица для трекинга email сообщений
CREATE TABLE email_messages (
    id UUID PRIMARY KEY,
    message_id VARCHAR(512) UNIQUE,  -- RFC Message-ID
    in_reply_to VARCHAR(512),
    references TEXT[],               -- Array of reference Message-IDs
    thread_id VARCHAR(255),
    ticket_id UUID REFERENCES tickets(id),
    
    from_email VARCHAR(255) NOT NULL,
    to_emails VARCHAR(255)[],
    cc_emails VARCHAR(255)[],
    subject VARCHAR(1024),
    
    body_html TEXT,
    body_text TEXT,
    
    direction VARCHAR(20) CHECK (direction IN ('incoming', 'outgoing')),
    source VARCHAR(50),              -- imap, smtp, web
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Индексы для быстрого поиска цепочек
CREATE INDEX idx_email_messages_message_id ON email_messages(message_id);
CREATE INDEX idx_email_messages_thread_id ON email_messages(thread_id);
CREATE INDEX idx_email_messages_ticket_id ON email_messages(ticket_id);
CREATE INDEX idx_email_messages_in_reply_to ON email_messages(in_reply_to);
```

## 🚀 Implementation Phases

### Phase 1: Basic IMAP Integration

```go
// IMAP Poller Implementation
type IMAPPoller struct {
    config     IMAPConfig
    interval   time.Duration
    lastUID    uint32
    processor  *EmailProcessor
}

func (p *IMAPPoller) Start() {
    ticker := time.NewTicker(p.interval)
    for range ticker.C {
        p.pollNewMessages()
    }
}

func (p *IMAPPoller) pollNewMessages() error {
    // Connect to IMAP
    // Fetch messages since last UID
    // Process each message
    // Update last UID
}
```

### Phase 2: Thread Management & Tracking

- Реализация системы трекинга Message-ID
- Алгоритмы сопоставления цепочек
- Обработка вложений

### Phase 3: Advanced Features

- Email templates
- Автоматические ответы
- Умная маршрутизация
- Analytics и reporting

## 🔍 Best Practices from Industry

### Jira-like Email Handling

1. Message-ID based threading - золотой стандарт
2. Special headers for metadata - не нарушают RFC
3. Fallback strategies - когда Message-ID отсутствует
4. Deduplication - защита от дубликатов

### Exchange Specific Considerations

```go
// Exchange IMAP особенности
type ExchangeIMAPConfig struct {
    IMAPConfig
    AuthMethod    string // NTLM, OAuth2
    Domain        string // AD domain
    ExchangeURL   string // EWS endpoint (опционально)
}
```

## 🎯 Immediate Next Steps

1. Прототип IMAP клиента - подключение к Exchange
2. Базовый парсер писем - извлечение Message-ID, subject, body
3. Простая база для email сообщений - сохранение входящих
4. SMTP отправка - ответы через систему с трекингом
