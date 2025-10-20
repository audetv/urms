// internal/infrastructure/http/dto/requests.go
package dto

import (
	"time"

	"github.com/audetv/urms/internal/core/domain"
)

// Task Requests

type CreateTaskRequest struct {
	Type        domain.TaskType `json:"type" binding:"required,oneof=support internal subtask"`
	Subject     string          `json:"subject" binding:"required,min=1,max=255"`
	Description string          `json:"description" binding:"required,min=1,max=5000"`
	CustomerID  *string         `json:"customer_id,omitempty"`
	Priority    domain.Priority `json:"priority" binding:"required,oneof=low medium high critical"`
	Category    string          `json:"category" binding:"required,min=1,max=100"`
	Tags        []string        `json:"tags,omitempty"`
	ParentID    *string         `json:"parent_id,omitempty"`
	ProjectID   *string         `json:"project_id,omitempty"`
	DueDate     *time.Time      `json:"due_date,omitempty"`
}

type CreateSupportTaskRequest struct {
	Subject     string          `json:"subject" binding:"required,min=1,max=255"`
	Description string          `json:"description" binding:"required,min=1,max=5000"`
	CustomerID  string          `json:"customer_id" binding:"required"`
	Priority    domain.Priority `json:"priority" binding:"required,oneof=low medium high critical"`
	Category    string          `json:"category" binding:"required,min=1,max=100"`
	Tags        []string        `json:"tags,omitempty"`
}

type CreateInternalTaskRequest struct {
	Subject     string          `json:"subject" binding:"required,min=1,max=255"`
	Description string          `json:"description" binding:"required,min=1,max=5000"`
	Priority    domain.Priority `json:"priority" binding:"required,oneof=low medium high critical"`
	Category    string          `json:"category" binding:"required,min=1,max=100"`
	Tags        []string        `json:"tags,omitempty"`
	ProjectID   *string         `json:"project_id,omitempty"`
}

type CreateSubTaskRequest struct {
	ParentID    string          `json:"parent_id" binding:"required"`
	Subject     string          `json:"subject" binding:"required,min=1,max=255"`
	Description string          `json:"description" binding:"required,min=1,max=5000"`
	Priority    domain.Priority `json:"priority" binding:"required,oneof=low medium high critical"`
	Category    string          `json:"category" binding:"required,min=1,max=100"`
	Tags        []string        `json:"tags,omitempty"`
}

type UpdateTaskRequest struct {
	Subject     *string          `json:"subject,omitempty" binding:"omitempty,min=1,max=255"`
	Description *string          `json:"description,omitempty" binding:"omitempty,min=1,max=5000"`
	Priority    *domain.Priority `json:"priority,omitempty" binding:"omitempty,oneof=low medium high critical"`
	Category    *string          `json:"category,omitempty" binding:"omitempty,min=1,max=100"`
	Tags        *[]string        `json:"tags,omitempty"`
	DueDate     *time.Time       `json:"due_date,omitempty"`
}

type ChangeStatusRequest struct {
	Status domain.TaskStatus `json:"status" binding:"required,oneof=open in_progress review resolved closed cancelled"`
}

type AssignTaskRequest struct {
	AssigneeID string `json:"assignee_id" binding:"required"`
}

type AddMessageRequest struct {
	Content   string             `json:"content" binding:"required,min=1,max=10000"`
	Type      domain.MessageType `json:"type" binding:"required,oneof=customer internal system"`
	IsPrivate bool               `json:"is_private,omitempty"`
}

type AddInternalNoteRequest struct {
	Content string `json:"content" binding:"required,min=1,max=10000"`
}

// Customer Requests

type CreateCustomerRequest struct {
	Name         string `json:"name" binding:"required,min=1,max=100"`
	Email        string `json:"email" binding:"required,email"`
	Phone        string `json:"phone,omitempty" binding:"omitempty,max=20"`
	Organization string `json:"organization,omitempty" binding:"omitempty,max=100"`
}

type UpdateCustomerRequest struct {
	Name         *string `json:"name,omitempty" binding:"omitempty,min=1,max=100"`
	Email        *string `json:"email,omitempty" binding:"omitempty,email"`
	Phone        *string `json:"phone,omitempty" binding:"omitempty,max=20"`
	Organization *string `json:"organization,omitempty" binding:"omitempty,max=100"`
}

// Search and Filter Requests

type TaskSearchRequest struct {
	Types      []domain.TaskType   `json:"types,omitempty" form:"types"`
	Statuses   []domain.TaskStatus `json:"statuses,omitempty" form:"statuses"`
	Priorities []domain.Priority   `json:"priorities,omitempty" form:"priorities"`
	AssigneeID string              `json:"assignee_id,omitempty" form:"assignee_id"`
	CustomerID string              `json:"customer_id,omitempty" form:"customer_id"`
	ReporterID string              `json:"reporter_id,omitempty" form:"reporter_id"`
	Category   string              `json:"category,omitempty" form:"category"`
	Tags       []string            `json:"tags,omitempty" form:"tags"`
	SearchText string              `json:"search_text,omitempty" form:"search_text"`
	Page       int                 `json:"page,omitempty" form:"page" binding:"min=1"`
	PageSize   int                 `json:"page_size,omitempty" form:"page_size" binding:"min=1,max=100"`
	SortBy     string              `json:"sort_by,omitempty" form:"sort_by"`
	SortOrder  string              `json:"sort_order,omitempty" form:"sort_order" binding:"omitempty,oneof=asc desc"`
}

type CustomerSearchRequest struct {
	SearchText   string `json:"search_text,omitempty" form:"search_text"`
	Organization string `json:"organization,omitempty" form:"organization"`
	Email        string `json:"email,omitempty" form:"email"`
	Page         int    `json:"page,omitempty" form:"page" binding:"min=1"`
	PageSize     int    `json:"page_size,omitempty" form:"page_size" binding:"min=1,max=100"`
}
