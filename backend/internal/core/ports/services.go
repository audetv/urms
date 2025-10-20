// internal/core/ports/services.go
package ports

import (
	"context"

	"github.com/audetv/urms/internal/core/domain"
)

// TaskService определяет бизнес-операции с задачами
type TaskService interface {
	// Core operations
	CreateTask(ctx context.Context, req CreateTaskRequest) (*domain.Task, error)
	GetTask(ctx context.Context, id string) (*domain.Task, error)
	UpdateTask(ctx context.Context, id string, req UpdateTaskRequest) (*domain.Task, error)
	DeleteTask(ctx context.Context, id string) error

	// Task type specific creation
	CreateSupportTask(ctx context.Context, req CreateSupportTaskRequest) (*domain.Task, error)
	CreateInternalTask(ctx context.Context, req CreateInternalTaskRequest) (*domain.Task, error)
	CreateSubTask(ctx context.Context, req CreateSubTaskRequest) (*domain.Task, error)

	// Status management
	ChangeStatus(ctx context.Context, id string, status domain.TaskStatus, userID string) (*domain.Task, error)
	AssignTask(ctx context.Context, id string, assigneeID string, userID string) (*domain.Task, error)
	AddParticipant(ctx context.Context, id string, userID string, role domain.ParticipantRole) (*domain.Task, error)

	// Communication
	AddMessage(ctx context.Context, id string, req AddMessageRequest) (*domain.Task, error)
	AddInternalNote(ctx context.Context, id string, authorID, content string) (*domain.Task, error)

	// Email threading support
	FindBySourceMeta(ctx context.Context, meta map[string]interface{}) ([]domain.Task, error)

	// Search and lists
	SearchTasks(ctx context.Context, query TaskQuery) (*TaskSearchResult, error)
	GetCustomerTasks(ctx context.Context, customerID string) ([]domain.Task, error)
	GetUserTasks(ctx context.Context, userID string, userRole domain.UserRole) (*UserTasks, error)
	GetSubtasks(ctx context.Context, parentID string) ([]domain.Task, error)

	// Analytics
	GetStats(ctx context.Context, query StatsQuery) (*TaskStats, error)
	GetDashboard(ctx context.Context, userID string) (*UserDashboard, error)

	// Automation
	AutoAssignTasks(ctx context.Context) ([]AutoAssignmentResult, error)
	ProcessEscalations(ctx context.Context) ([]EscalationResult, error)

	// Bulk operations
	BulkUpdateStatus(ctx context.Context, taskIDs []string, status domain.TaskStatus, userID string) ([]BulkOperationResult, error)
	BulkAssign(ctx context.Context, taskIDs []string, assigneeID string, userID string) ([]BulkOperationResult, error)
}

// CustomerService определяет бизнес-операции с клиентами
type CustomerService interface {
	CreateCustomer(ctx context.Context, req CreateCustomerRequest) (*domain.Customer, error)
	FindOrCreateByEmail(ctx context.Context, email, name string) (*domain.Customer, error)
	GetCustomerProfile(ctx context.Context, id string) (*CustomerProfile, error)
	UpdateCustomer(ctx context.Context, id string, req UpdateCustomerRequest) (*domain.Customer, error)
	DeleteCustomer(ctx context.Context, id string) error
	ListCustomers(ctx context.Context, query CustomerQuery) (*CustomerSearchResult, error)
}

// KnowledgeService определяет сервис для работы с базой знаний
type KnowledgeService interface {
	CreateDocument(ctx context.Context, req CreateDocumentRequest) (*domain.KnowledgeDocument, error)
	GetDocument(ctx context.Context, id string) (*domain.KnowledgeDocument, error)
	UpdateDocument(ctx context.Context, id string, req UpdateDocumentRequest) (*domain.KnowledgeDocument, error)
	SearchDocuments(ctx context.Context, query KnowledgeQuery) (*KnowledgeSearchResult, error)
	PublishDocument(ctx context.Context, id string) (*domain.KnowledgeDocument, error)
	ArchiveDocument(ctx context.Context, id string) (*domain.KnowledgeDocument, error)

	// AI-enhanced operations
	FindRelevantDocuments(ctx context.Context, query string, taskContext *domain.Task) ([]domain.KnowledgeDocument, error)
	GenerateResponse(ctx context.Context, task *domain.Task, userQuery string) (*domain.AIResponse, error)
}

// ProjectService определяет сервис для управления проектами
type ProjectService interface {
	CreateProject(ctx context.Context, req CreateProjectRequest) (*domain.Project, error)
	GetProject(ctx context.Context, id string) (*domain.Project, error)
	UpdateProject(ctx context.Context, id string, req UpdateProjectRequest) (*domain.Project, error)
	DeleteProject(ctx context.Context, id string) error
	ListProjects(ctx context.Context, query ProjectQuery) (*ProjectSearchResult, error)

	// Milestone management
	CreateMilestone(ctx context.Context, req CreateMilestoneRequest) (*domain.Milestone, error)
	UpdateMilestone(ctx context.Context, id string, req UpdateMilestoneRequest) (*domain.Milestone, error)
	GetProjectMilestones(ctx context.Context, projectID string) ([]domain.Milestone, error)

	// Team management
	AddTeamMember(ctx context.Context, projectID string, req AddTeamMemberRequest) (*domain.Project, error)
	RemoveTeamMember(ctx context.Context, projectID string, userID string) (*domain.Project, error)
}

// ClassificationService определяет сервис AI-классификации
type ClassificationService interface {
	ClassifyTask(ctx context.Context, content string) (*ClassificationResult, error)
	SuggestAssignee(ctx context.Context, task *domain.Task) ([]AssigneeSuggestion, error)
	ExtractEntities(ctx context.Context, text string) (*EntityExtractionResult, error)
	SuggestCategory(ctx context.Context, task *domain.Task) ([]CategorySuggestion, error)
}

// Request/Response structures

type CreateTaskRequest struct {
	Type        domain.TaskType
	Subject     string
	Description string
	CustomerID  *string // Optional для internal tasks
	ReporterID  string
	Source      domain.TaskSource
	SourceMeta  map[string]interface{}
	Priority    domain.Priority
	Category    string
	Tags        []string
	ParentID    *string // Для подзадач
	ProjectID   *string // Для привязки к проекту
}

type CreateSupportTaskRequest struct {
	Subject     string
	Description string
	CustomerID  string
	ReporterID  string
	Source      domain.TaskSource
	SourceMeta  map[string]interface{}
	Priority    domain.Priority
	Category    string
	Tags        []string
}

type CreateInternalTaskRequest struct {
	Subject     string
	Description string
	ReporterID  string
	Priority    domain.Priority
	Category    string
	Tags        []string
	ProjectID   *string
}

type CreateSubTaskRequest struct {
	ParentID    string
	Subject     string
	Description string
	ReporterID  string
	Priority    domain.Priority
	Category    string
	Tags        []string
}

type UpdateTaskRequest struct {
	Subject     *string
	Description *string
	Priority    *domain.Priority
	Category    *string
	Tags        *[]string
	DueDate     *string
}

type AddMessageRequest struct {
	AuthorID  string
	Content   string
	Type      domain.MessageType
	IsPrivate bool
}

type CreateCustomerRequest struct {
	Name         string
	Email        string
	Phone        string
	Organization string
}

type UpdateCustomerRequest struct {
	Name         *string
	Email        *string
	Phone        *string
	Organization *string
}

type CustomerQuery struct {
	SearchText   string
	Organization string
	Email        string
	Offset       int
	Limit        int
}

type CreateDocumentRequest struct {
	Type     domain.KnowledgeDocumentType
	Title    string
	Content  string
	Summary  string
	Tags     []string
	Category string
	Authors  []string
}

type UpdateDocumentRequest struct {
	Title    *string
	Content  *string
	Summary  *string
	Tags     *[]string
	Category *string
	Status   *domain.DocumentStatus
}

type CreateProjectRequest struct {
	Name        string
	Description string
	Category    string
	Tags        []string
	OwnerID     string
	TeamID      *string
	StartDate   *string
	EndDate     *string
}

type UpdateProjectRequest struct {
	Name        *string
	Description *string
	Category    *string
	Tags        *[]string
	Status      *domain.ProjectStatus
	StartDate   *string
	EndDate     *string
}

type CreateMilestoneRequest struct {
	ProjectID   string
	Name        string
	Description string
	DueDate     *string
}

type UpdateMilestoneRequest struct {
	Name        *string
	Description *string
	DueDate     *string
	Status      *domain.MilestoneStatus
}

type AddTeamMemberRequest struct {
	UserID string
	Role   domain.TeamRole
}

type ClassificationResult struct {
	Category    string
	Priority    domain.Priority
	Tags        []string
	Confidence  float64
	Entities    []Entity
	Suggestions []string
}

type CategorySuggestion struct {
	Category   string
	Confidence float64
	Reason     string
}

type Entity struct {
	Type  string
	Value string
	Start int
	End   int
}

type AssigneeSuggestion struct {
	UserID     string
	Confidence float64
	Reason     string
}

type EntityExtractionResult struct {
	Technologies []string
	Products     []string
	Errors       []string
	Features     []string
	UrgencyLevel string
}

type TaskSearchResult struct {
	Tasks      []domain.Task
	TotalCount int
	Page       int
	PageSize   int
	TotalPages int
}

type KnowledgeSearchResult struct {
	Documents  []domain.KnowledgeDocument
	TotalCount int
	Page       int
	PageSize   int
	TotalPages int
}

type ProjectSearchResult struct {
	Projects   []domain.Project
	TotalCount int
	Page       int
	PageSize   int
	TotalPages int
}

type CustomerSearchResult struct {
	Customers  []domain.Customer
	TotalCount int
	Page       int
	PageSize   int
	TotalPages int
}

type UserTasks struct {
	AssignedTasks   []domain.Task
	ReportedTasks   []domain.Task
	ParticipatingIn []domain.Task
	Watching        []domain.Task
}

type UserDashboard struct {
	MyTasks         []domain.Task
	AssignedCount   int
	UnassignedCount int
	OverdueCount    int
	RecentActivity  []DashboardActivity
	Stats           *UserStats
}

type DashboardActivity struct {
	Type      string
	TaskID    string
	Subject   string
	UserID    string
	Timestamp string
}

type UserStats struct {
	OpenTasks         int
	ResolvedToday     int
	AvgResolutionTime float64
	SatisfactionRate  float64
}

type AutoAssignmentResult struct {
	TaskID     string
	AssigneeID string
	Success    bool
	Error      string
}

type EscalationResult struct {
	TaskID  string
	Action  string
	Success bool
	Error   string
}

type BulkOperationResult struct {
	TaskID  string
	Success bool
	Error   string
}

type CustomerProfile struct {
	Customer     *domain.Customer
	Tasks        []domain.Task
	Stats        *CustomerStats
	RecentTasks  []domain.Task
	Organization *domain.Organization
}

type CustomerStats struct {
	TotalTasks      int
	OpenTasks       int
	AvgResponseTime float64
	Satisfaction    float64
	ByPriority      map[domain.Priority]int
	ByCategory      map[string]int
}

// Backward compatibility types for email module
type CreateTicketRequest = CreateSupportTaskRequest
type UpdateTicketRequest = UpdateTaskRequest
type TicketSearchResult = TaskSearchResult
