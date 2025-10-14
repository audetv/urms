package id

import (
	"fmt"

	"github.com/google/uuid"
)

// UUIDGenerator реализует domain.IDGenerator используя google/uuid
type UUIDGenerator struct{}

func NewUUIDGenerator() *UUIDGenerator {
	return &UUIDGenerator{}
}

func (g *UUIDGenerator) GenerateID() string {
	return uuid.New().String()
}

func (g *UUIDGenerator) GenerateMessageID() string {
	return fmt.Sprintf("<%s@urms.local>", uuid.New().String())
}

func (g *UUIDGenerator) GenerateThreadID() string {
	return fmt.Sprintf("thread_%s", uuid.New().String())
}
