// backend/internal/infrastructure/persistence/email/postgres_repository.go
package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/audetv/urms/internal/core/domain"
	"github.com/audetv/urms/internal/core/ports"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/rs/zerolog/log"
)

// PostgresEmailRepository реализует ports.EmailRepository для PostgreSQL
type PostgresEmailRepository struct {
	db *sqlx.DB
}

// NewPostgresEmailRepository создает новый PostgreSQL репозиторий
func NewPostgresEmailRepository(db *sqlx.DB) *PostgresEmailRepository {
	return &PostgresEmailRepository{
		db: db,
	}
}

// Save сохраняет email сообщение
func (r *PostgresEmailRepository) Save(ctx context.Context, msg *domain.EmailMessage) error {
	model, err := FromDomain(msg)
	if err != nil {
		return fmt.Errorf("failed to convert domain message to model: %w", err)
	}

	query := `
		INSERT INTO email_messages (
			id, message_id, in_reply_to, thread_id, from_email, to_emails, 
			cc_emails, bcc_emails, subject, body_text, body_html, direction,
			source, headers, processed, processed_at, related_ticket_id,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19
		)
		ON CONFLICT (message_id) 
		DO UPDATE SET
			in_reply_to = EXCLUDED.in_reply_to,
			thread_id = EXCLUDED.thread_id,
			from_email = EXCLUDED.from_email,
			to_emails = EXCLUDED.to_emails,
			cc_emails = EXCLUDED.cc_emails,
			bcc_emails = EXCLUDED.bcc_emails,
			subject = EXCLUDED.subject,
			body_text = EXCLUDED.body_text,
			body_html = EXCLUDED.body_html,
			direction = EXCLUDED.direction,
			source = EXCLUDED.source,
			headers = EXCLUDED.headers,
			processed = EXCLUDED.processed,
			processed_at = EXCLUDED.processed_at,
			related_ticket_id = EXCLUDED.related_ticket_id,
			updated_at = EXCLUDED.updated_at
	`

	_, err = r.db.ExecContext(ctx, query,
		model.ID,
		model.MessageID,
		nullString(model.InReplyTo),
		nullString(model.ThreadID),
		model.FromEmail,
		model.ToEmails,
		model.CcEmails,
		model.BccEmails,
		model.Subject,
		nullString(model.BodyText),
		nullString(model.BodyHTML),
		model.Direction,
		model.Source,
		model.Headers,
		model.Processed,
		model.ProcessedAt, // Просто передаем sql.NullTime
		nullStringPtr(model.RelatedTicketID),
		model.CreatedAt,
		model.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to save email message: %w", err)
	}

	log.Debug().
		Str("message_id", msg.MessageID).
		Str("id", string(msg.ID)).
		Msg("Email message saved to PostgreSQL")

	return nil
}

// FindByID находит сообщение по ID
func (r *PostgresEmailRepository) FindByID(ctx context.Context, id domain.MessageID) (*domain.EmailMessage, error) {
	var model EmailMessageModel

	query := `SELECT * FROM email_messages WHERE id = $1`
	err := r.db.GetContext(ctx, &model, query, string(id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrEmailNotFound
		}
		return nil, fmt.Errorf("failed to find email by ID: %w", err)
	}

	domainMsg, err := model.ToDomain()
	if err != nil {
		return nil, fmt.Errorf("failed to convert model to domain: %w", err)
	}

	return domainMsg, nil
}

// FindByMessageID находит сообщение по MessageID
func (r *PostgresEmailRepository) FindByMessageID(ctx context.Context, messageID string) (*domain.EmailMessage, error) {
	var model EmailMessageModel

	query := `SELECT * FROM email_messages WHERE message_id = $1`
	err := r.db.GetContext(ctx, &model, query, messageID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrEmailNotFound
		}
		return nil, fmt.Errorf("failed to find email by MessageID: %w", err)
	}

	domainMsg, err := model.ToDomain()
	if err != nil {
		return nil, fmt.Errorf("failed to convert model to domain: %w", err)
	}

	return domainMsg, nil
}

// Update обновляет сообщение
func (r *PostgresEmailRepository) Update(ctx context.Context, msg *domain.EmailMessage) error {
	model, err := FromDomain(msg)
	if err != nil {
		return fmt.Errorf("failed to convert domain message to model: %w", err)
	}

	query := `
		UPDATE email_messages SET
			in_reply_to = $2,
			thread_id = $3,
			from_email = $4,
			to_emails = $5,
			cc_emails = $6,
			bcc_emails = $7,
			subject = $8,
			body_text = $9,
			body_html = $10,
			direction = $11,
			source = $12,
			headers = $13,
			processed = $14,
			processed_at = $15,
			related_ticket_id = $16,
			updated_at = $17
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query,
		model.ID,
		nullString(model.InReplyTo),
		nullString(model.ThreadID),
		model.FromEmail,
		model.ToEmails,
		model.CcEmails,
		model.BccEmails,
		model.Subject,
		nullString(model.BodyText),
		nullString(model.BodyHTML),
		model.Direction,
		model.Source,
		model.Headers,
		model.Processed,
		model.ProcessedAt, // Просто передаем sql.NullTime
		nullStringPtr(model.RelatedTicketID),
		model.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to update email message: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return domain.ErrEmailNotFound
	}

	log.Debug().
		Str("message_id", msg.MessageID).
		Msg("Email message updated in PostgreSQL")

	return nil
}

// Delete удаляет сообщение
func (r *PostgresEmailRepository) Delete(ctx context.Context, id domain.MessageID) error {
	query := `DELETE FROM email_messages WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, string(id))
	if err != nil {
		return fmt.Errorf("failed to delete email message: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return domain.ErrEmailNotFound
	}

	log.Debug().
		Str("id", string(id)).
		Msg("Email message deleted from PostgreSQL")

	return nil
}

// FindUnprocessed находит необработанные сообщения
func (r *PostgresEmailRepository) FindUnprocessed(ctx context.Context) ([]domain.EmailMessage, error) {
	var models []EmailMessageModel

	query := `SELECT * FROM email_messages WHERE processed = false ORDER BY created_at ASC`
	err := r.db.SelectContext(ctx, &models, query)
	if err != nil {
		return nil, fmt.Errorf("failed to find unprocessed emails: %w", err)
	}

	return r.convertModelsToDomain(models)
}

// FindByPeriod находит сообщения за период
func (r *PostgresEmailRepository) FindByPeriod(ctx context.Context, from, to time.Time) ([]domain.EmailMessage, error) {
	var models []EmailMessageModel

	query := `SELECT * FROM email_messages WHERE created_at BETWEEN $1 AND $2 ORDER BY created_at ASC`
	err := r.db.SelectContext(ctx, &models, query, from, to)
	if err != nil {
		return nil, fmt.Errorf("failed to find emails by period: %w", err)
	}

	return r.convertModelsToDomain(models)
}

// FindByInReplyTo находит сообщения по In-Reply-To
func (r *PostgresEmailRepository) FindByInReplyTo(ctx context.Context, inReplyTo string) ([]domain.EmailMessage, error) {
	var models []EmailMessageModel

	query := `SELECT * FROM email_messages WHERE in_reply_to = $1 ORDER BY created_at ASC`
	err := r.db.SelectContext(ctx, &models, query, inReplyTo)
	if err != nil {
		return nil, fmt.Errorf("failed to find emails by In-Reply-To: %w", err)
	}

	return r.convertModelsToDomain(models)
}

// FindByReferences находит сообщения по References
func (r *PostgresEmailRepository) FindByReferences(ctx context.Context, references []string) ([]domain.EmailMessage, error) {
	if len(references) == 0 {
		return []domain.EmailMessage{}, nil
	}

	var models []EmailMessageModel

	query := `SELECT * FROM email_messages WHERE in_reply_to = ANY($1) OR message_id = ANY($1) ORDER BY created_at ASC`
	err := r.db.SelectContext(ctx, &models, query, pq.Array(references)) // Теперь pq доступен
	if err != nil {
		return nil, fmt.Errorf("failed to find emails by references: %w", err)
	}

	return r.convertModelsToDomain(models)
}

// FindByRelatedTicket находит сообщения по связанному тикету
func (r *PostgresEmailRepository) FindByRelatedTicket(ctx context.Context, ticketID string) ([]domain.EmailMessage, error) {
	var models []EmailMessageModel

	query := `SELECT * FROM email_messages WHERE related_ticket_id = $1 ORDER BY created_at ASC`
	err := r.db.SelectContext(ctx, &models, query, ticketID)
	if err != nil {
		return nil, fmt.Errorf("failed to find emails by related ticket: %w", err)
	}

	return r.convertModelsToDomain(models)
}

// Helper methods

// convertModelsToDomain конвертирует слайс моделей в domain сущности
func (r *PostgresEmailRepository) convertModelsToDomain(models []EmailMessageModel) ([]domain.EmailMessage, error) {
	result := make([]domain.EmailMessage, len(models))
	for i, model := range models {
		domainMsg, err := model.ToDomain()
		if err != nil {
			return nil, fmt.Errorf("failed to convert model at index %d: %w", i, err)
		}
		result[i] = *domainMsg
	}
	return result, nil
}

// nullString возвращает sql.NullString для пустых строк
func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: s, Valid: true}
}

// nullStringPtr возвращает sql.NullString для string указателя
func nullStringPtr(s *string) sql.NullString {
	if s == nil {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: *s, Valid: true}
}

// Ensure interface compliance
var _ ports.EmailRepository = (*PostgresEmailRepository)(nil)
