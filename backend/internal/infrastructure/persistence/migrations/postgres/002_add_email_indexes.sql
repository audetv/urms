-- backend/internal/infrastructure/persistence/migrations/postgres/002_add_email_indexes.sql

-- Migration: 002_add_email_indexes  
-- Description: Add performance indexes for email queries
-- Created at: 2025-10-15

BEGIN;

-- Убираем CREATE UNIQUE INDEX т.к. он уже создан в первой миграции
-- Остальные индексы оставляем
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

-- Функция и триггер оставляем без изменений
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_email_messages_updated_at 
    BEFORE UPDATE ON email_messages 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();

COMMIT;