package domain

// IDGenerator определяет контракт для генерации идентификаторов
// Находится в DOMAIN слое, так как это доменная концепция
type IDGenerator interface {
	GenerateID() string
	GenerateMessageID() string
	GenerateThreadID() string
}
