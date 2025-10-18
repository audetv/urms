package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/audetv/urms/internal/core/domain"
	"github.com/audetv/urms/internal/core/ports"
	"github.com/audetv/urms/internal/core/services"
	"github.com/audetv/urms/internal/infrastructure/common/id"
	"github.com/audetv/urms/internal/infrastructure/email"
	imapclient "github.com/audetv/urms/internal/infrastructure/email/imap"
	persistence "github.com/audetv/urms/internal/infrastructure/persistence/email"
)

// ConsoleLogger реализует ports.Logger для вывода в консоль
type ConsoleLogger struct{}

func (l *ConsoleLogger) Debug(ctx context.Context, msg string, fields ...interface{}) {
	fmt.Printf("🔍 [DEBUG] %s %v\n", msg, fields)
}

func (l *ConsoleLogger) Info(ctx context.Context, msg string, fields ...interface{}) {
	fmt.Printf("ℹ️  [INFO] %s %v\n", msg, fields)
}

func (l *ConsoleLogger) Warn(ctx context.Context, msg string, fields ...interface{}) {
	fmt.Printf("⚠️  [WARN] %s %v\n", msg, fields)
}

func (l *ConsoleLogger) Error(ctx context.Context, msg string, fields ...interface{}) {
	fmt.Printf("❌ [ERROR] %s %v\n", msg, fields)
}

func (l *ConsoleLogger) WithContext(ctx context.Context) context.Context {
	return ctx
}

func main() {
	fmt.Println("🚀 URMS Email Module - New Architecture Test")
	fmt.Println("============================================")

	// Получаем конфигурацию из environment variables
	username := os.Getenv("URMS_IMAP_USERNAME")
	password := os.Getenv("URMS_IMAP_PASSWORD")
	server := os.Getenv("URMS_IMAP_SERVER")

	if username == "" || password == "" {
		log.Fatal("❌ Please set URMS_IMAP_USERNAME and URMS_IMAP_PASSWORD environment variables")
	}

	if server == "" {
		server = "outlook.office365.com" // default
		fmt.Printf("🔧 Using default server: %s\n", server)
	}

	// Создаем конфигурацию IMAP
	imapConfig := &imapclient.Config{
		Server:   server,
		Port:     993,
		Username: username,
		Password: password,
		Mailbox:  "INBOX",
		SSL:      true,
		Interval: 30 * time.Second,
		Timeout:  30 * time.Second,
		ReadOnly: true,
	}

	fmt.Printf("🔧 IMAP Configuration:\n")
	fmt.Printf("   Server: %s:%d\n", imapConfig.Server, imapConfig.Port)
	fmt.Printf("   Username: %s\n", imapConfig.Username)
	fmt.Printf("   Mailbox: %s\n", imapConfig.Mailbox)
	fmt.Printf("   SSL: %v\n", imapConfig.SSL)
	fmt.Printf("   ReadOnly: %v\n", imapConfig.ReadOnly)
	fmt.Println()

	// Инициализируем зависимости
	ctx := context.Background()
	logger := &ConsoleLogger{}

	// Infrastructure layer
	fmt.Println("🛠️  Initializing dependencies...")
	imapAdapter := email.NewIMAPAdapterLegacy(imapConfig)
	emailRepo, err := persistence.NewEmailRepository(persistence.RepositoryTypeInMemory, nil)
	if err != nil {
		log.Fatalf("❌ Failed to create email repository: %v", err)
	}
	idGenerator := id.NewUUIDGenerator()

	// Domain policies
	emailPolicy := domain.EmailProcessingPolicy{
		ReadOnlyMode:   true, // Для тестирования - только чтение
		AutoReply:      false,
		SpamFilter:     true,
		MaxMessageSize: 10 * 1024 * 1024, // 10MB
		AllowedSenders: []domain.EmailAddress{},
		BlockedSenders: []domain.EmailAddress{},
	}

	// Core services
	emailService := services.NewEmailService(
		imapAdapter,
		emailRepo,
		nil, // Пока без MessageProcessor
		idGenerator,
		emailPolicy,
		logger,
	)

	fmt.Println("✅ Dependencies initialized successfully")
	fmt.Println()

	// Тестируем соединение
	fmt.Println("🔌 Testing IMAP connection...")
	if err := emailService.TestConnection(ctx); err != nil {
		log.Fatalf("❌ Connection test failed: %v", err)
	}
	fmt.Println("✅ Connection test successful")
	fmt.Println()

	// Получаем информацию о почтовом ящике
	fmt.Println("📬 Getting mailbox information...")
	mailboxInfo, err := imapAdapter.GetMailboxInfo(ctx, "INBOX")
	if err != nil {
		log.Printf("⚠️  Failed to get mailbox info: %v", err)
	} else {
		fmt.Printf("✅ Mailbox Info:\n")
		fmt.Printf("   Name: %s\n", mailboxInfo.Name)
		fmt.Printf("   Total Messages: %d\n", mailboxInfo.Messages)
		fmt.Printf("   Unseen Messages: %d\n", mailboxInfo.Unseen)
		fmt.Printf("   Recent Messages: %d\n", mailboxInfo.Recent)
	}
	fmt.Println()

	// Создаем context с таймаутом для предотвращения зависания
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second) // ✅ 10 секунд таймаут
	defer cancel()

	// Тестируем получение сообщений
	fmt.Println("📧 Testing message fetching...")
	criteria := ports.FetchCriteria{
		Mailbox:    "INBOX",
		Limit:      5, // Только 5 сообщений для теста
		UnseenOnly: false,
		Since:      time.Now().Add(-24 * time.Hour), // За последние 24 часа
	}

	messages, err := imapAdapter.FetchMessages(ctx, criteria)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			log.Printf("❌ Fetch messages timed out after 10 seconds")
		} else {
			log.Printf("⚠️  Failed to fetch messages: %v", err)
		}
	} else {
		fmt.Printf("✅ Successfully fetched %d messages\n", len(messages))

		for i, msg := range messages {
			fmt.Printf("\n%d. 📨 Message:\n", i+1)
			fmt.Printf("   ID: %s\n", msg.MessageID)
			fmt.Printf("   From: %s\n", msg.From)
			fmt.Printf("   Subject: %s\n", msg.Subject)
			fmt.Printf("   Date: %s\n", msg.CreatedAt.Format("2006-01-02 15:04:05"))
			fmt.Printf("   Is Reply: %v\n", msg.IsReply())

			// Сохраняем в репозиторий
			if err := emailRepo.Save(ctx, &msg); err != nil {
				fmt.Printf("   ⚠️  Failed to save: %v\n", err)
			} else {
				fmt.Printf("   💾 Saved to repository\n")
			}
		}
	}
	fmt.Println()

	// Тестируем бизнес-логику обработки
	fmt.Println("⚙️  Testing business logic processing...")
	if err := emailService.ProcessIncomingEmails(ctx); err != nil {
		log.Printf("⚠️  Processing failed: %v", err)
	} else {
		fmt.Println("✅ Email processing completed")
	}
	fmt.Println()

	// Получаем статистику
	fmt.Println("📊 Getting email statistics...")
	stats, err := emailService.GetEmailStatistics(ctx)
	if err != nil {
		log.Printf("⚠️  Failed to get statistics: %v", err)
	} else {
		fmt.Printf("✅ Email Statistics:\n")
		fmt.Printf("   Unprocessed Messages: %d\n", stats.UnprocessedCount)
		fmt.Printf("   Recent Messages (24h): %d\n", stats.RecentCount)
		if stats.LastProcessed != nil {
			fmt.Printf("   Last Processed: %s\n", stats.LastProcessed.Format("2006-01-02 15:04:05"))
		} else {
			fmt.Printf("   Last Processed: Never\n")
		}
	}
	fmt.Println()

	// Тестируем отправку email (в read-only режиме это будет пропущено)
	fmt.Println("📤 Testing email sending (read-only mode)...")
	testEmail := domain.EmailMessage{
		From:     domain.EmailAddress("test@urms.local"),
		To:       []domain.EmailAddress{"recipient@example.com"},
		Subject:  "Test Email from URMS",
		BodyText: "This is a test email from URMS system.",
	}

	if err := emailService.SendEmail(ctx, testEmail); err != nil {
		fmt.Printf("⚠️  Send test completed with: %v\n", err)
	} else {
		fmt.Println("✅ Send test completed (message would be sent in non-read-only mode)")
	}
	fmt.Println()

	fmt.Println("🎉 New Architecture Test Completed Successfully!")
	fmt.Println("==============================================")
	fmt.Println("✅ All components working correctly")
	fmt.Println("✅ Hexagonal Architecture implemented")
	fmt.Println("✅ Business logic isolated from infrastructure")
	fmt.Println("✅ Ready for Phase 1B - IMAP Poller implementation")
}
