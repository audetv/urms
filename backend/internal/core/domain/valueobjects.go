package domain

import (
	"fmt"
	"strings"
)

// EmailHeader - value object для email заголовков
type EmailHeader struct {
	Key   string
	Value string
}

func (h EmailHeader) String() string {
	return fmt.Sprintf("%s: %s", h.Key, h.Value)
}

// MessagePriority - value object для приоритета сообщения
type MessagePriority int

// const (
// 	PriorityLow MessagePriority = iota
// 	PriorityNormal
// 	PriorityHigh
// 	PriorityUrgent
// )

// func (p MessagePriority) String() string {
// 	switch p {
// 	case PriorityLow:
// 		return "low"
// 	case PriorityNormal:
// 		return "normal"
// 	case PriorityHigh:
// 		return "high"
// 	case PriorityUrgent:
// 		return "urgent"
// 	default:
// 		return "normal"
// 	}
// }

// EmailContent - value object для содержимого email
type EmailContent struct {
	Text string
	HTML string
}

func (c EmailContent) IsEmpty() bool {
	return strings.TrimSpace(c.Text) == "" && strings.TrimSpace(c.HTML) == ""
}

func (c EmailContent) Preferred() string {
	if strings.TrimSpace(c.HTML) != "" {
		return c.HTML
	}
	return c.Text
}
