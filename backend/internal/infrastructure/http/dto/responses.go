// internal/infrastructure/http/dto/responses.go
package dto

import (
	"time"

	"github.com/audetv/urms/internal/core/domain"
)

// Common Responses

type BaseResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *ErrorInfo  `json:"error,omitempty"`
	Meta    *MetaInfo   `json:"meta,omitempty"`
}

type ErrorInfo struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

type MetaInfo struct {
	Timestamp time.Time `json:"timestamp"`
	RequestID string    `json:"request_id,omitempty"`
	Page      *PageInfo `json:"page,omitempty"`
}

type PageInfo struct {
	Page       int `json:"page"`
	PageSize   int `json:"page_size"`
	TotalCount int `json:"total_count"`
	TotalPages int `json:"total_pages"`
}

// Task Responses

type TaskResponse struct {
	ID          string            `json:"id"`
	Type        domain.TaskType   `json:"type"`
	Subject     string            `json:"subject"`
	Description string            `json:"description"`
	Status      domain.TaskStatus `json:"status"`
	Priority    domain.Priority   `json:"priority"`
	Category    string            `json:"category"`
	Tags        []string          `json:"tags"`

	// Relationships
	ParentID   *string `json:"parent_id,omitempty"`
	ProjectID  *string `json:"project_id,omitempty"`
	AssigneeID string  `json:"assignee_id,omitempty"`
	ReporterID string  `json:"reporter_id"`
	CustomerID *string `json:"customer_id,omitempty"`

	// Source information
	Source     domain.TaskSource      `json:"source"`
	SourceMeta map[string]interface{} `json:"source_meta,omitempty"`

	// Participants
	Participants []ParticipantResponse `json:"participants,omitempty"`

	// Messages (only in detailed responses)
	Messages []MessageResponse `json:"messages,omitempty"`

	// History (only in detailed responses)
	History []TaskEventResponse `json:"history,omitempty"`

	// Timestamps
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	DueDate    *time.Time `json:"due_date,omitempty"`
	ResolvedAt *time.Time `json:"resolved_at,omitempty"`
	ClosedAt   *time.Time `json:"closed_at,omitempty"`
}

type ParticipantResponse struct {
	UserID   string                 `json:"user_id"`
	Role     domain.ParticipantRole `json:"role"`
	JoinedAt time.Time              `json:"joined_at"`
}

type MessageResponse struct {
	ID        string             `json:"id"`
	Content   string             `json:"content"`
	AuthorID  string             `json:"author_id"`
	Type      domain.MessageType `json:"type"`
	CreatedAt time.Time          `json:"created_at"`
}

type TaskEventResponse struct {
	ID        string      `json:"id"`
	Type      string      `json:"type"`
	UserID    string      `json:"user_id"`
	OldValue  interface{} `json:"old_value,omitempty"`
	NewValue  interface{} `json:"new_value,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
	Message   string      `json:"message"`
}

type TaskListResponse struct {
	Tasks      []TaskResponse `json:"tasks"`
	Pagination PageInfo       `json:"pagination"`
}

// Customer Responses

type CustomerResponse struct {
	ID           string                `json:"id"`
	Name         string                `json:"name"`
	Email        string                `json:"email"`
	Phone        string                `json:"phone,omitempty"`
	Organization *OrganizationResponse `json:"organization,omitempty"`
	CreatedAt    time.Time             `json:"created_at"`
	UpdatedAt    time.Time             `json:"updated_at"`
}

type OrganizationResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type CustomerProfileResponse struct {
	Customer     CustomerResponse      `json:"customer"`
	Tasks        []TaskResponse        `json:"tasks"`
	Stats        CustomerStats         `json:"stats"`
	RecentTasks  []TaskResponse        `json:"recent_tasks,omitempty"`
	Organization *OrganizationResponse `json:"organization,omitempty"`
}

type CustomerStats struct {
	TotalTasks      int                     `json:"total_tasks"`
	OpenTasks       int                     `json:"open_tasks"`
	AvgResponseTime float64                 `json:"avg_response_time"`
	Satisfaction    float64                 `json:"satisfaction"`
	ByPriority      map[domain.Priority]int `json:"by_priority"`
	ByCategory      map[string]int          `json:"by_category"`
}

type CustomerListResponse struct {
	Customers  []CustomerResponse `json:"customers"`
	Pagination PageInfo           `json:"pagination"`
}

// User Responses

type UserResponse struct {
	ID    string          `json:"id"`
	Email string          `json:"email"`
	Name  string          `json:"name"`
	Role  domain.UserRole `json:"role"`
}

// Health and System Responses

type HealthResponse struct {
	Status    string            `json:"status"`
	Timestamp time.Time         `json:"timestamp"`
	Services  map[string]string `json:"services,omitempty"`
}

type StatsResponse struct {
	TotalTasks        int                       `json:"total_tasks"`
	OpenTasks         int                       `json:"open_tasks"`
	InProgressTasks   int                       `json:"in_progress_tasks"`
	ResolvedTasks     int                       `json:"resolved_tasks"`
	ClosedTasks       int                       `json:"closed_tasks"`
	AvgResolutionTime float64                   `json:"avg_resolution_time"`
	ByPriority        map[domain.Priority]int   `json:"by_priority"`
	ByCategory        map[string]int            `json:"by_category"`
	BySource          map[domain.TaskSource]int `json:"by_source"`
	ByType            map[domain.TaskType]int   `json:"by_type"`
}

// Helper functions

func NewSuccessResponse(data interface{}) BaseResponse {
	return BaseResponse{
		Success: true,
		Data:    data,
		Meta: &MetaInfo{
			Timestamp: time.Now(),
		},
	}
}

func NewPaginatedResponse(data interface{}, pageInfo PageInfo) BaseResponse {
	return BaseResponse{
		Success: true,
		Data:    data,
		Meta: &MetaInfo{
			Timestamp: time.Now(),
			Page:      &pageInfo,
		},
	}
}

func NewErrorResponse(code, message, details string) BaseResponse {
	return BaseResponse{
		Success: false,
		Error: &ErrorInfo{
			Code:    code,
			Message: message,
			Details: details,
		},
		Meta: &MetaInfo{
			Timestamp: time.Now(),
		},
	}
}
