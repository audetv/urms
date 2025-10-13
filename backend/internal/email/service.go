// backend/internal/email/service.go
package email

import (
	"github.com/audetv/urms/internal/email/imapclient"
	"github.com/audetv/urms/internal/email/models"
)

// Repository интерфейс для работы с хранилищем
type Repository interface {
	SaveEmailMessage(msg *models.EmailMessage) error
	FindEmailMessageByID(messageID string) (*models.EmailMessage, error)
	FindThreadByID(threadID string) ([]*models.EmailMessage, error)
}

// MessageProcessor интерфейс для обработки сообщений
type MessageProcessor interface {
	ProcessMessage(rawMessage []byte, envelope *imapclient.EnvelopeInfo) error
}

// Service основной сервис email
type Service struct {
	imapClient *imapclient.Client
	repository Repository
	processor  MessageProcessor
}

// NewService создает новый email сервис
func NewService(imapConfig *imapclient.Config, repo Repository, processor MessageProcessor) (*Service, error) {
	client := imapclient.NewClient(imapConfig)

	return &Service{
		imapClient: client,
		repository: repo,
		processor:  processor,
	}, nil
}

// TestConnection тестирует соединение с IMAP сервером
func (s *Service) TestConnection() error {
	return s.imapClient.Connect()
}
