// backend/internal/email/imapclient/client.go
package imapclient

import (
	"fmt"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/rs/zerolog/log"
)

// Client обертка вокруг go-imap клиента с переподключением
type Client struct {
	config      *Config
	client      *client.Client
	isConnected bool
	lastError   error
	connectedAt time.Time
}

// NewClient создает новый IMAP клиент
func NewClient(config *Config) *Client {
	return &Client{
		config: config,
	}
}

// Connect устанавливает соединение с IMAP сервером
func (c *Client) Connect() error {
	log.Info().
		Str("server", c.config.Addr()).
		Bool("ssl", c.config.SSL).
		Msg("Connecting to IMAP server")

	var cl *client.Client
	var err error

	// Устанавливаем соединение (убрали context, так как go-imap не использует его напрямую)
	if c.config.SSL {
		cl, err = client.DialTLS(c.config.Addr(), nil)
	} else {
		cl, err = client.Dial(c.config.Addr())
	}

	if err != nil {
		c.lastError = err
		return fmt.Errorf("failed to connect to IMAP server: %w", err)
	}

	// Настраиваем таймауты
	cl.Timeout = c.config.Timeout

	// Логин
	if err := cl.Login(c.config.Username, c.config.Password); err != nil {
		cl.Logout()
		c.lastError = err
		return fmt.Errorf("failed to login to IMAP server: %w", err)
	}

	c.client = cl
	c.isConnected = true
	c.connectedAt = time.Now()
	c.lastError = nil

	log.Info().
		Str("server", c.config.Addr()).
		Str("username", c.config.Username).
		Msg("Successfully connected to IMAP server")

	return nil
}

// CheckConnection проверяет соединение и переподключается при необходимости
func (c *Client) CheckConnection() error {
	if !c.isConnected || c.client == nil {
		return c.Connect()
	}

	// Проверяем соединение отправкой NOOP команды
	if err := c.client.Noop(); err != nil {
		log.Warn().
			Err(err).
			Msg("IMAP connection lost, reconnecting...")

		c.isConnected = false
		if c.client != nil {
			c.client.Logout()
			c.client = nil
		}
		return c.Connect()
	}

	return nil
}

// SelectMailbox выбирает почтовый ящик
func (c *Client) SelectMailbox(name string, readOnly bool) (*imap.MailboxStatus, error) {
	if err := c.CheckConnection(); err != nil {
		return nil, err
	}

	mailbox, err := c.client.Select(name, readOnly)
	if err != nil {
		c.isConnected = false
		return nil, fmt.Errorf("failed to select mailbox %s: %w", name, err)
	}

	log.Debug().
		Str("mailbox", name).
		Int("total_messages", int(mailbox.Messages)).
		Msg("Mailbox selected")

	return mailbox, nil
}

// SearchMessages выполняет поиск сообщений по критериям
func (c *Client) SearchMessages(criteria *imap.SearchCriteria) ([]uint32, error) {
	if err := c.CheckConnection(); err != nil {
		return nil, err
	}

	return c.client.Search(criteria)
}

// FetchMessages получает сообщения по их UID с улучшенной обработкой ошибок
func (c *Client) FetchMessages(seqset *imap.SeqSet, items []imap.FetchItem) (chan *imap.Message, error) {
	if err := c.CheckConnection(); err != nil {
		return nil, err
	}

	if seqset == nil || seqset.Empty() {
		return nil, fmt.Errorf("empty sequence set")
	}

	messages := make(chan *imap.Message, 10)
	err := c.client.Fetch(seqset, items, messages)
	if err != nil {
		close(messages) // Важно закрыть канал при ошибке
		return nil, fmt.Errorf("failed to fetch messages: %w", err)
	}

	return messages, nil
}

// GetMailboxInfo получает информацию о почтовом ящике без его выбора
func (c *Client) GetMailboxInfo(name string) (*imap.MailboxStatus, error) {
	if err := c.CheckConnection(); err != nil {
		return nil, err
	}

	mailbox, err := c.client.Status(name, []imap.StatusItem{imap.StatusMessages, imap.StatusUnseen})
	if err != nil {
		return nil, fmt.Errorf("failed to get mailbox status for %s: %w", name, err)
	}

	return mailbox, nil
}

// Logout закрывает соединение
func (c *Client) Logout() error {
	if c.client != nil {
		err := c.client.Logout()
		c.client = nil
		c.isConnected = false
		return err
	}
	return nil
}

// IsConnected возвращает статус соединения
func (c *Client) IsConnected() bool {
	return c.isConnected
}

// GetLastError возвращает последнюю ошибку
func (c *Client) GetLastError() error {
	return c.lastError
}

// GetConnectedAt возвращает время установления соединения
func (c *Client) GetConnectedAt() time.Time {
	return c.connectedAt
}

// GetConnectionUptime возвращает время с последнего подключения
func (c *Client) GetConnectionUptime() time.Duration {
	if !c.isConnected {
		return 0
	}
	return time.Since(c.connectedAt)
}

// GetConnectionStatus возвращает статус соединения в виде строки
func (c *Client) GetConnectionStatus() string {
	if !c.isConnected {
		return "disconnected"
	}

	uptime := time.Since(c.connectedAt)
	if uptime < time.Minute {
		return fmt.Sprintf("connected (%v)", uptime.Round(time.Second))
	} else if uptime < time.Hour {
		return fmt.Sprintf("connected (%v)", uptime.Round(time.Minute))
	} else {
		return fmt.Sprintf("connected (%v)", uptime.Round(time.Hour))
	}
}

// GetStats возвращает статистику соединения
func (c *Client) GetStats() map[string]interface{} {
	stats := map[string]interface{}{
		"is_connected": c.isConnected,
		"last_error":   nil,
	}

	if c.lastError != nil {
		stats["last_error"] = c.lastError.Error()
	}

	if c.isConnected {
		stats["connected_at"] = c.connectedAt
		stats["uptime"] = time.Since(c.connectedAt).String()
	}

	return stats
}
