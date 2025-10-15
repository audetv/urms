-- backend/internal/infrastructure/persistence/email/postgres_schema.sql

-- Таблица для email сообщений
CREATE TABLE IF NOT EXISTS email_messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    message_id VARCHAR(500) NOT NULL,
    in_reply_to VARCHAR(500),
    thread_id VARCHAR(500),
    from_email VARCHAR(255) NOT NULL,
    to_emails JSONB NOT NULL DEFAULT '[]',
    cc_emails JSONB NOT NULL DEFAULT '[]',
    bcc_emails JSONB NOT NULL DEFAULT '[]',
    subject TEXT,
    body_text TEXT,
    body_html TEXT,
    direction VARCHAR(20) NOT NULL CHECK (direction IN ('incoming', 'outgoing')),
    source VARCHAR(50) NOT NULL,
    headers JSONB NOT NULL DEFAULT '{}',
    processed BOOLEAN NOT NULL DEFAULT FALSE,
    processed_at TIMESTAMP WITH TIME ZONE,
    related_ticket_id VARCHAR(100),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Таблица для вложений
CREATE TABLE IF NOT EXISTS email_attachments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    message_id VARCHAR(500) NOT NULL,
    name VARCHAR(500) NOT NULL,
    content_type VARCHAR(255) NOT NULL,
    size BIGINT NOT NULL,
    content_id VARCHAR(500),
    data BYTEA,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    
    CONSTRAINT fk_email_attachments_message 
        FOREIGN KEY (message_id) 
        REFERENCES email_messages(message_id) 
        ON DELETE CASCADE
);

-- Индексы для оптимизации запросов
CREATE UNIQUE INDEX IF NOT EXISTS idx_email_messages_message_id 
    ON email_messages(message_id);

CREATE INDEX IF NOT EXISTS idx_email_messages_in_reply_to 
    ON email_messages(in_reply_to) 
    WHERE in_reply_to IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_email_messages_thread_id 
    ON email_messages(thread_id) 
    WHERE thread_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_email_messages_processed 
    ON email_messages(processed) 
    WHERE processed = false;

CREATE INDEX IF NOT EXISTS idx_email_messages_created_at 
    ON email_messages(created_at);

CREATE INDEX IF NOT EXISTS idx_email_messages_related_ticket 
    ON email_messages(related_ticket_id) 
    WHERE related_ticket_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_email_messages_from_email 
    ON email_messages(from_email);

CREATE INDEX IF NOT EXISTS idx_email_attachments_message_id 
    ON email_attachments(message_id);

-- Функция для автоматического обновления updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Триггер для автоматического обновления updated_at
CREATE TRIGGER update_email_messages_updated_at 
    BEFORE UPDATE ON email_messages 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();