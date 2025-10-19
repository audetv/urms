// internal/core/domain/future_models.go
package domain

import "time"

/*
БАЗА ЗНАНИЙ И ДОКУМЕНТАЦИЯ

KnowledgeBase будет содержать всю структурированную информацию организации:
- Решения (Decisions) - принятые архитектурные и бизнес-решения
- Спецификации (Specifications) - технические и бизнес-спецификации
- Инструкции (Instructions) - руководства и процедуры
- Методики (Methodologies) - методологии работы
- Промпты (Prompts) - шаблоны для AI-агентов
- Отчеты (Reports) - шаблоны и исторические отчеты
- Правила (Rules) - бизнес-правила и политики

Все документы будут доступны для AI RAG системы и помогут:
- Автоматически отвечать на вопросы клиентов
- Формировать предложения и рекомендации
- Генерировать отчеты и аналитику
- Поддерживать единые стандарты работы
*/

// KnowledgeDocumentType тип документа в базе знаний
type KnowledgeDocumentType string

const (
	DocumentTypeDecision      KnowledgeDocumentType = "decision"      // Решение (ADR, бизнес-решения)
	DocumentTypeSpecification KnowledgeDocumentType = "specification" // Спецификация
	DocumentTypeInstruction   KnowledgeDocumentType = "instruction"   // Инструкция
	DocumentTypeMethodology   KnowledgeDocumentType = "methodology"   // Методика
	DocumentTypePrompt        KnowledgeDocumentType = "prompt"        // Промпт для AI
	DocumentTypeReport        KnowledgeDocumentType = "report"        // Отчет
	DocumentTypeRule          KnowledgeDocumentType = "rule"          // Правило
	DocumentTypeArticle       KnowledgeDocumentType = "article"       // Статья базы знаний
	DocumentTypeFAQ           KnowledgeDocumentType = "faq"           // Часто задаваемые вопросы
)

// KnowledgeDocument документ базы знаний
type KnowledgeDocument struct {
	ID          string
	Type        KnowledgeDocumentType
	Title       string
	Content     string
	Summary     string
	Tags        []string
	Category    string
	RelatedTo   []DocumentReference // Связи с задачами, проектами, другими документами
	Authors     []string
	Version     string
	Status      DocumentStatus
	CreatedAt   time.Time
	UpdatedAt   time.Time
	PublishedAt *time.Time
}

// DocumentReference ссылка на связанную сущность
type DocumentReference struct {
	EntityType string // "task", "project", "milestone", "document"
	EntityID   string
	Relation   string // "implements", "references", "related_to"
}

// DocumentStatus статус документа
type DocumentStatus string

const (
	DocumentStatusDraft     DocumentStatus = "draft"     // Черновик
	DocumentStatusReview    DocumentStatus = "review"    // На проверке
	DocumentStatusPublished DocumentStatus = "published" // Опубликован
	DocumentStatusArchived  DocumentStatus = "archived"  // Архивирован
)

// Decision представляет принятое решение (Architectural Decision Record)
type Decision struct {
	ID           string
	Title        string
	Context      string // Контекст проблемы
	Decision     string // Принятое решение
	Consequences string // Последствия
	Status       DecisionStatus
	RelatedTask  *string // Связь с задачей, в рамках которой принято решение
	ApprovedBy   []string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// DecisionStatus статус решения
type DecisionStatus string

const (
	DecisionStatusProposed   DecisionStatus = "proposed"   // Предложено
	DecisionStatusAccepted   DecisionStatus = "accepted"   // Принято
	DecisionStatusSuperseded DecisionStatus = "superseded" // Заменено
	DecisionStatusRejected   DecisionStatus = "rejected"   // Отклонено
)

/*
ПРОЕКТЫ И ЭТАПЫ

Project система будет управлять проектами организации:
- Проекты (Projects) - основные контейнеры работ
- Этапы (Milestones) - ключевые вехи проекта
- Спринты (Sprints) - итерации разработки (опционально)
- Команды (Teams) - группы пользователей
*/

// Project представляет проект
type Project struct {
	ID          string
	Name        string
	Description string
	Status      ProjectStatus
	Category    string
	Tags        []string
	OwnerID     string
	TeamID      *string // Связь с командой
	StartDate   *time.Time
	EndDate     *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// ProjectStatus статус проекта
type ProjectStatus string

const (
	ProjectStatusPlanning  ProjectStatus = "planning"  // Планирование
	ProjectStatusActive    ProjectStatus = "active"    // Активный
	ProjectStatusOnHold    ProjectStatus = "on_hold"   // На паузе
	ProjectStatusCompleted ProjectStatus = "completed" // Завершен
	ProjectStatusCancelled ProjectStatus = "cancelled" // Отменен
)

// Milestone представляет этап проекта
type Milestone struct {
	ID          string
	ProjectID   string
	Name        string
	Description string
	DueDate     *time.Time
	Status      MilestoneStatus
	Tasks       []string // Связанные задачи
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// MilestoneStatus статус этапа
type MilestoneStatus string

const (
	MilestoneStatusPending   MilestoneStatus = "pending"   // Ожидает
	MilestoneStatusActive    MilestoneStatus = "active"    // Активен
	MilestoneStatusCompleted MilestoneStatus = "completed" // Завершен
	MilestoneStatusCancelled MilestoneStatus = "cancelled" // Отменен
)

// Team представляет команду пользователей
type Team struct {
	ID          string
	Name        string
	Description string
	Members     []TeamMember
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// TeamMember участник команды
type TeamMember struct {
	UserID   string
	Role     TeamRole
	JoinedAt time.Time
}

// TeamRole роль в команде
type TeamRole string

const (
	TeamRoleLead   TeamRole = "lead"   // Руководитель
	TeamRoleMember TeamRole = "member" // Участник
	TeamRoleViewer TeamRole = "viewer" // Наблюдатель
)

/*
AI RAG СИСТЕМА

AISystem будет обеспечивать интеллектуальные возможности:
- Векторный поиск по базе знаний
- Семантическое понимание запросов
- Генерация контента и ответов
- Автоматическая классификация
*/

// AIContext контекст для AI-агентов
type AIContext struct {
	UserQuery      string
	ContextDocs    []KnowledgeDocument // Релевантные документы
	UserProfile    *User
	TaskContext    *Task
	ProjectContext *Project
	Timestamp      time.Time
}

// AIResponse ответ AI-системы
type AIResponse struct {
	Content          string
	Sources          []DocumentReference
	Confidence       float64
	Type             AIResponseType
	SuggestedActions []AIAction
}

// AIResponseType тип ответа AI
type AIResponseType string

const (
	AIResponseTypeAnswer         AIResponseType = "answer"         // Ответ на вопрос
	AIResponseTypeSuggestion     AIResponseType = "suggestion"     // Предложение
	AIResponseTypeReport         AIResponseType = "report"         // Отчет
	AIResponseTypeClassification AIResponseType = "classification" // Классификация
)

// AIAction предлагаемое действие
type AIAction struct {
	Type        string
	Description string
	Parameters  map[string]interface{}
}
