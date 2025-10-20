// internal/infrastructure/email/utils.go
package email

import (
	"math/rand"
	"time"
)

func generateShortID() string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	const length = 8

	seededRand := rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}
