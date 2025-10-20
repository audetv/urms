// internal/core/ports/repositories.go
package ports

import (
	"context"

	"github.com/audetv/urms/internal/core/domain"
)

// TaskRepository определяет контракт для работы с задачами
type TaskRepository interface {
	// Basic CRUD operations
	Save(ctx context.Context, task *domain.Task) error
	FindByID(ctx context.Context, id string) (*domain.Task, error)
	FindByQuery(ctx context.Context, query TaskQuery) ([]domain.Task, error)
	Update(ctx context.Context, task *domain.Task) error
	Delete(ctx context.Context, id string) error

	// Specific queries
	FindByCustomerID(ctx context.Context, customerID string) ([]domain.Task, error)
	FindByAssigneeID(ctx context.Context, assigneeID string) ([]domain.Task, error)
	FindByStatus(ctx context.Context, status domain.TaskStatus) ([]domain.Task, error)
	FindByType(ctx context.Context, taskType domain.TaskType) ([]domain.Task, error)
	FindOpenTasks(ctx context.Context) ([]domain.Task, error)
	FindSubtasks(ctx context.Context, parentID string) ([]domain.Task, error)
	// Email threading support
	FindBySourceMeta(ctx context.Context, meta map[string]interface{}) ([]domain.Task, error)

	// Statistics
	GetStats(ctx context.Context, query StatsQuery) (*TaskStats, error)
	GetAssigneeWorkload(ctx context.Context) (map[string]int, error)

	// Bulk operations
	BulkUpdateStatus(ctx context.Context, taskIDs []string, status domain.TaskStatus) error
}

// CustomerRepository определяет контракт для работы с клиентами
type CustomerRepository interface {
	Save(ctx context.Context, customer *domain.Customer) error
	FindByID(ctx context.Context, id string) (*domain.Customer, error)
	FindByEmail(ctx context.Context, email string) (*domain.Customer, error)
	FindByOrganization(ctx context.Context, orgID string) ([]domain.Customer, error)
	Update(ctx context.Context, customer *domain.Customer) error
	Delete(ctx context.Context, id string) error
}

// UserRepository определяет контракт для работы с пользователями системы
type UserRepository interface {
	FindByID(ctx context.Context, id string) (*domain.User, error)
	FindByEmail(ctx context.Context, email string) (*domain.User, error)
	FindAssignees(ctx context.Context) ([]domain.User, error)
	Save(ctx context.Context, user *domain.User) error
	Update(ctx context.Context, user *domain.User) error
	Delete(ctx context.Context, id string) error
}

// KnowledgeRepository определяет контракт для работы с базой знаний
type KnowledgeRepository interface {
	SaveDocument(ctx context.Context, doc *domain.KnowledgeDocument) error
	FindDocumentByID(ctx context.Context, id string) (*domain.KnowledgeDocument, error)
	FindDocumentsByQuery(ctx context.Context, query KnowledgeQuery) ([]domain.KnowledgeDocument, error)
	FindDocumentsByType(ctx context.Context, docType domain.KnowledgeDocumentType) ([]domain.KnowledgeDocument, error)
	UpdateDocument(ctx context.Context, doc *domain.KnowledgeDocument) error
	DeleteDocument(ctx context.Context, id string) error
}

// ProjectRepository определяет контракт для работы с проектами
type ProjectRepository interface {
	SaveProject(ctx context.Context, project *domain.Project) error
	FindProjectByID(ctx context.Context, id string) (*domain.Project, error)
	FindProjectsByQuery(ctx context.Context, query ProjectQuery) ([]domain.Project, error)
	UpdateProject(ctx context.Context, project *domain.Project) error
	DeleteProject(ctx context.Context, id string) error

	// Milestone operations
	SaveMilestone(ctx context.Context, milestone *domain.Milestone) error
	FindMilestoneByID(ctx context.Context, id string) (*domain.Milestone, error)
	FindMilestonesByProject(ctx context.Context, projectID string) ([]domain.Milestone, error)
	UpdateMilestone(ctx context.Context, milestone *domain.Milestone) error
}

// Query structures

// TaskQuery представляет критерии поиска задач
type TaskQuery struct {
	Types      []domain.TaskType
	Statuses   []domain.TaskStatus
	Priorities []domain.Priority
	AssigneeID string
	CustomerID string
	ReporterID string
	Source     []domain.TaskSource
	Tags       []string
	Category   string
	ParentID   *string // Для поиска подзадач
	ProjectID  *string // Для поиска по проектам
	DateFrom   *string
	DateTo     *string
	SearchText string
	Offset     int
	Limit      int
	SortBy     string
	SortOrder  string // "asc" or "desc"
}

// KnowledgeQuery представляет критерии поиска в базе знаний
type KnowledgeQuery struct {
	Types      []domain.KnowledgeDocumentType
	Statuses   []domain.DocumentStatus
	Tags       []string
	Category   string
	SearchText string
	Authors    []string
	Offset     int
	Limit      int
	SortBy     string
	SortOrder  string
}

// ProjectQuery представляет критерии поиска проектов
type ProjectQuery struct {
	Statuses   []domain.ProjectStatus
	Category   string
	Tags       []string
	OwnerID    string
	TeamID     string
	SearchText string
	Offset     int
	Limit      int
	SortBy     string
	SortOrder  string
}

// StatsQuery представляет критерии для статистики
type StatsQuery struct {
	DateFrom   *string
	DateTo     *string
	CustomerID string
	AssigneeID string
	Category   string
	Source     []domain.TaskSource
	Types      []domain.TaskType
}

// TaskStats представляет статистику по задачам
type TaskStats struct {
	TotalCount      int
	OpenCount       int
	InProgressCount int
	ResolvedCount   int
	ClosedCount     int

	// Время разрешения (в часах)
	AvgResolutionTime float64
	MaxResolutionTime float64
	MinResolutionTime float64

	// Распределение
	ByPriority map[domain.Priority]int
	ByCategory map[string]int
	BySource   map[domain.TaskSource]int
	ByType     map[domain.TaskType]int

	// Статистика по исполнителям
	TopAssignees []AssigneeStats
}

// AssigneeStats представляет статистику по исполнителям
type AssigneeStats struct {
	AssigneeID  string
	TicketCount int
	TaskCount   int // Новое поле для совместимости
	AvgTime     float64
}
