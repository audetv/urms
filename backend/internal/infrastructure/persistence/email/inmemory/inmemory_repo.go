package inmemory

import (
	"context"
	"sync"
	"time"

	"github.com/audetv/urms/internal/core/domain"
)

// InMemoryEmailRepo реализует ports.EmailRepository для тестирования
type InMemoryEmailRepo struct {
	mu       sync.RWMutex
	messages map[domain.MessageID]*domain.EmailMessage
	byMsgID  map[string]*domain.EmailMessage // Индекс по MessageID
}

// NewInMemoryEmailRepo создает новый in-memory репозиторий
func NewInMemoryEmailRepo() *InMemoryEmailRepo {
	return &InMemoryEmailRepo{
		messages: make(map[domain.MessageID]*domain.EmailMessage),
		byMsgID:  make(map[string]*domain.EmailMessage),
	}
}

// Save сохраняет email сообщение
func (r *InMemoryEmailRepo) Save(ctx context.Context, msg *domain.EmailMessage) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.messages[msg.ID] = msg
	r.byMsgID[msg.MessageID] = msg

	return nil
}

// FindByID находит сообщение по ID
func (r *InMemoryEmailRepo) FindByID(ctx context.Context, id domain.MessageID) (*domain.EmailMessage, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	msg, exists := r.messages[id]
	if !exists {
		return nil, domain.ErrEmailNotFound
	}

	return msg, nil
}

// FindByMessageID находит сообщение по MessageID
func (r *InMemoryEmailRepo) FindByMessageID(ctx context.Context, messageID string) (*domain.EmailMessage, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	msg, exists := r.byMsgID[messageID]
	if !exists {
		return nil, domain.ErrEmailNotFound
	}

	return msg, nil
}

// Update обновляет сообщение
func (r *InMemoryEmailRepo) Update(ctx context.Context, msg *domain.EmailMessage) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.messages[msg.ID]; !exists {
		return domain.ErrEmailNotFound
	}

	r.messages[msg.ID] = msg
	r.byMsgID[msg.MessageID] = msg

	return nil
}

// Delete удаляет сообщение
func (r *InMemoryEmailRepo) Delete(ctx context.Context, id domain.MessageID) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	msg, exists := r.messages[id]
	if !exists {
		return domain.ErrEmailNotFound
	}

	delete(r.messages, id)
	delete(r.byMsgID, msg.MessageID)

	return nil
}

// FindUnprocessed находит необработанные сообщения
func (r *InMemoryEmailRepo) FindUnprocessed(ctx context.Context) ([]domain.EmailMessage, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []domain.EmailMessage
	for _, msg := range r.messages {
		if !msg.Processed {
			result = append(result, *msg)
		}
	}

	return result, nil
}

// FindByPeriod находит сообщения за период
func (r *InMemoryEmailRepo) FindByPeriod(ctx context.Context, from, to time.Time) ([]domain.EmailMessage, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []domain.EmailMessage
	for _, msg := range r.messages {
		if !msg.CreatedAt.Before(from) && !msg.CreatedAt.After(to) {
			result = append(result, *msg)
		}
	}

	return result, nil
}

// FindByInReplyTo находит сообщения по In-Reply-To
func (r *InMemoryEmailRepo) FindByInReplyTo(ctx context.Context, inReplyTo string) ([]domain.EmailMessage, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []domain.EmailMessage
	for _, msg := range r.messages {
		if msg.InReplyTo == inReplyTo {
			result = append(result, *msg)
		}
	}

	return result, nil
}

// FindByReferences находит сообщения по References
func (r *InMemoryEmailRepo) FindByReferences(ctx context.Context, references []string) ([]domain.EmailMessage, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []domain.EmailMessage
	for _, msg := range r.messages {
		for _, ref := range references {
			for _, msgRef := range msg.References {
				if msgRef == ref {
					result = append(result, *msg)
					break
				}
			}
		}
	}

	return result, nil
}

// FindByRelatedTicket находит сообщения по связанному тикету
func (r *InMemoryEmailRepo) FindByRelatedTicket(ctx context.Context, ticketID string) ([]domain.EmailMessage, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []domain.EmailMessage
	for _, msg := range r.messages {
		if msg.RelatedTicketID != nil && *msg.RelatedTicketID == ticketID {
			result = append(result, *msg)
		}
	}

	return result, nil
}
