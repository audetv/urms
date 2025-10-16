-- backend/internal/infrastructure/persistence/migrations/postgres/001_create_email_tables.sql
-- Убираем BEGIN/COMMIT - теперь этим управляет Go код

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

-- Создаем UNIQUE constraint
CREATE UNIQUE INDEX IF NOT EXISTS idx_email_messages_message_id_unique 
    ON email_messages(message_id);

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