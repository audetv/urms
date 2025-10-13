// backend/cmd/test-imap/main.go
package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/audetv/urms/internal/email/imapclient"
	"github.com/emersion/go-imap"
)

func main() {
	// Для тестирования можно установить credentials через environment variables
	username := os.Getenv("URMS_IMAP_USERNAME")
	password := os.Getenv("URMS_IMAP_PASSWORD")
	server := os.Getenv("URMS_IMAP_SERVER")

	if username == "" || password == "" {
		log.Fatal("Please set URMS_IMAP_USERNAME and URMS_IMAP_PASSWORD environment variables")
	}

	if server == "" {
		server = "outlook.office365.com" // default
	}

	config := &imapclient.Config{
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

	fmt.Printf("🔧 Testing IMAP connection to %s...\n", config.Addr())

	client := imapclient.NewClient(config)

	// Тестируем подключение
	if err := client.Connect(); err != nil {
		log.Fatalf("❌ Failed to connect: %v", err)
	}
	defer func() {
		if err := client.Logout(); err != nil {
			log.Printf("⚠️  Error during logout: %v", err)
		}
	}()

	fmt.Println("✅ Successfully connected to IMAP server")
	fmt.Printf("⏱️  Connection established at: %s\n", client.GetConnectedAt().Format("2006-01-02 15:04:05"))

	// Получаем информацию о почтовом ящике без его выбора
	mailboxInfo, err := client.GetMailboxInfo(config.Mailbox)
	if err != nil {
		log.Printf("⚠️  Failed to get mailbox info: %v", err)
	} else {
		fmt.Printf("📬 Mailbox: %s\n", config.Mailbox)
		fmt.Printf("📧 Total messages: %d\n", mailboxInfo.Messages)
		fmt.Printf("🔍 Unseen messages: %d\n", mailboxInfo.Unseen)
	}

	// Выбираем почтовый ящик для более детальной информации
	mailbox, err := client.SelectMailbox(config.Mailbox, true)
	if err != nil {
		log.Fatalf("❌ Failed to select mailbox: %v", err)
	}

	fmt.Printf("📁 Selected mailbox: %s\n", mailbox.Name)
	fmt.Printf("💬 Mailbox flags: %v\n", mailbox.Flags)

	// Получаем информацию о последних 5 сообщениях
	if mailbox.Messages > 0 {
		seqset := new(imap.SeqSet)
		start := uint32(1)
		end := mailbox.Messages
		if mailbox.Messages > 5 {
			start = mailbox.Messages - 4
		}
		seqset.AddRange(start, end)

		fmt.Printf("\n📋 Fetching last %d messages...\n", end-start+1)

		messages, err := client.FetchMessages(seqset, imapclient.CreateFetchItems(false))
		if err != nil {
			log.Printf("⚠️  Failed to fetch messages: %v", err)
		} else {
			messageCount := 0
			for msg := range messages {
				envelope := imapclient.GetMessageEnvelopeInfo(msg)
				if envelope != nil {
					messageCount++
					from := "Unknown"
					if len(envelope.From) > 0 {
						from = envelope.From[0]
					}
					date := envelope.Date.Format("2006-01-02 15:04")
					if envelope.Date.IsZero() {
						date = "Unknown date"
					}

					fmt.Printf("  %d. [%s] %s\n", messageCount, date, envelope.Subject)
					fmt.Printf("     From: %s\n", from)
					if envelope.InReplyTo != "" {
						fmt.Printf("     In-Reply-To: %s\n", envelope.InReplyTo)
					}
					fmt.Println()
				}
			}
			fmt.Printf("📊 Successfully processed %d messages\n", messageCount)
		}
	} else {
		fmt.Println("📭 No messages in mailbox")
	}

	fmt.Printf("\n🎉 IMAP client test completed successfully!\n")
	fmt.Printf("⏱️  Connection uptime: %v\n", client.GetConnectionUptime())
}
