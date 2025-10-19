// backend/internal/core/domain/types.go
package domain

import "time"

// ВАЖНО: Этот файл содержит фундаментальные типы системы управления на основе МИП.
// НЕ УДАЛЯТЬ КОММЕНТАРИИ - они объясняют онтологические принципы системы.

// EntityType - тип сущности в системе управления
// Сущности представляют собой объекты управления, которые существуют
// в триединстве Мера-Информация-Процесс и участвуют в иерархии до ИНВОУ
type EntityType string

const (
	// EntityTypeTicket - заявка или обращение пользователя
	// Представляет собой фактор среды, требующий управления
	EntityTypeTicket EntityType = "ticket"

	// EntityTypeNews - новость или информационное сообщение
	// Может быть фактором среды, влияющим на процессы управления
	EntityTypeNews EntityType = "news"

	// EntityTypeResource - ресурс системы (оборудование, специалисты)
	// Представляет собой средства для осуществления процессов управления
	EntityTypeResource EntityType = "resource"

	// EntityTypeProcess - процесс управления
	// Динамическая составляющая триединства МИП
	EntityTypeProcess EntityType = "process"

	// EntityTypeUser - пользователь системы
	// Субъект управления, осуществляющий ПФУ
	EntityTypeUser EntityType = "user"

	// EntityTypeSystem - системная сущность
	// Автономные компоненты, реализующие этапы ПФУ
	EntityTypeSystem EntityType = "system"
)

// FactorType - тип фактора среды в ПФУ
// Факторы среды - это явления, которые "давят на психику" и вызывают
// потребность в управлении (этап 1 ПФУ)
type FactorType string

const (
	// FactorTypeTechnical - технические факторы (сбои, ошибки, нагрузки)
	FactorTypeTechnical FactorType = "technical"

	// FactorTypeBusiness - бизнес-факторы (требования, изменения, возможности)
	FactorTypeBusiness FactorType = "business"

	// FactorTypeEnvironmental - внешние факторы среды
	FactorTypeEnvironmental FactorType = "environmental"

	// FactorTypeSocial - социальные факторы (взаимодействия, коммуникации)
	FactorTypeSocial FactorType = "social"

	// FactorTypeEconomic - экономические факторы (бюджеты, затраты, эффективность)
	FactorTypeEconomic FactorType = "economic"
)

// Priority - приоритет в системе управления
// Определяет очередность обработки факторов и распределения ресурсов
type Priority int

const (
	PriorityLow      Priority = iota // Низкий приоритет - фоновые процессы
	PriorityMedium                   // Средний приоритет - обычные операции
	PriorityHigh                     // Высокий приоритет - срочные задачи
	PriorityCritical                 // Критический приоритет - немедленное реагирование
)

// EntityReference - ссылка на связанную сущность
// Отражает структурные отношения (Мера) между сущностями в системе
type EntityReference struct {
	EntityID     string     // Идентификатор связанной сущности
	EntityType   EntityType // Тип связанной сущности
	RelationType string     // Тип отношения (родитель, ребенок, зависимость и т.д.)
}

// StateSnapshot - снимок состояния сущности
// Представляет собой информационный срез (Информация) процесса в конкретный момент времени
// Квантование процесса через меру различения состояний
type StateSnapshot struct {
	Timestamp time.Time              // Момент времени среза
	Metrics   map[string]interface{} // Числовые метрики состояния
	Status    string                 // Текстовое описание статуса
}

// Pattern - паттерн распознавания в системе
// Относится к Мере - структурным шаблонам для идентификации факторов и состояний
type Pattern struct {
	ID          string      // Уникальный идентификатор паттерна
	Name        string      // Человеко-читаемое название
	Description string      // Подробное описание паттерна
	Conditions  []Condition // Условия применения паттерна
}

// Category - категория классификации сущности
// Элемент таксономии (Мера) для организации сущностей в системе
type Category struct {
	ID   string // Идентификатор категории
	Name string // Название категории
	Type string // Тип категории (функциональная, техническая, бизнесовая)
}

// Relation - отношение между сущностями
// Структурный элемент Меры, определяющий связи в системе
type Relation struct {
	FromEntity string  // Исходная сущность отношения
	ToEntity   string  // Целевая сущность отношения
	Type       string  // Тип отношения (наследование, композиция, зависимость)
	Strength   float64 // Сила связи (0-1)
}

// Attribute - атрибут сущности
// Структурная характеристика (Мера), определяющая свойства сущности
type Attribute struct {
	Key   string      // Ключ атрибута
	Value interface{} // Значение атрибута
	Type  string      // Тип значения (string, number, boolean, object)
}

// Metric - метрика системы
// Информационный элемент (Информация) для количественной оценки состояний и процессов
type Metric struct {
	Name  string  // Название метрики
	Value float64 // Числовое значение
	Unit  string  // Единица измерения
}

// Event - событие в системе
// Информация о значимом изменении состояния или процессе
type Event struct {
	ID        string                 // Идентификатор события
	Type      string                 // Тип события
	Timestamp time.Time              // Время возникновения
	Data      map[string]interface{} // Дополнительные данные события
}

// ProcessPhase - фаза процесса управления
// Динамическая характеристика (Процесс) текущего этапа развития
type ProcessPhase string

// ProcessFlow - поток процесса
// Динамическая модель (Процесс) последовательности состояний и переходов
type ProcessFlow string

// ProcessStatus - статус процесса
// Информационная характеристика (Информация) текущего состояния процесса
type ProcessStatus string

const (
	ProcessStatusActive    ProcessStatus = "active"    // Процесс активен и выполняется
	ProcessStatusPaused    ProcessStatus = "paused"    // Процесс приостановлен
	ProcessStatusCompleted ProcessStatus = "completed" // Процесс завершен успешно
	ProcessStatusFailed    ProcessStatus = "failed"    // Процесс завершен с ошибкой
)
