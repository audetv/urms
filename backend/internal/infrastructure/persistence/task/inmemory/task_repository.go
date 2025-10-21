// internal/infrastructure/persistence/task/inmemory/task_repository.go
package inmemory

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/audetv/urms/internal/core/domain"
	"github.com/audetv/urms/internal/core/ports"
)

type TaskRepository struct {
	tasks  map[string]*domain.Task
	mu     sync.RWMutex
	logger ports.Logger
}

func NewTaskRepository(logger ports.Logger) *TaskRepository {
	return &TaskRepository{
		tasks:  make(map[string]*domain.Task),
		logger: logger,
	}
}

func (r *TaskRepository) Save(ctx context.Context, task *domain.Task) error {
	if task == nil {
		return errors.New("task cannot be nil")
	}
	if task.ID == "" {
		return errors.New("task ID cannot be empty")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.tasks[task.ID] = task
	r.logger.Info(ctx, "task saved", "task_id", task.ID, "type", task.Type)
	return nil
}

func (r *TaskRepository) FindByID(ctx context.Context, id string) (*domain.Task, error) {
	if id == "" {
		return nil, errors.New("task ID cannot be empty")
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	task, exists := r.tasks[id]
	if !exists {
		return nil, fmt.Errorf("task not found: %s", id)
	}

	return task, nil
}

func (r *TaskRepository) FindByQuery(ctx context.Context, query ports.TaskQuery) ([]domain.Task, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var tasks []domain.Task

	for _, task := range r.tasks {
		if r.matchesQuery(task, query) {
			tasks = append(tasks, *task)
		}
	}

	// Применяем сортировку
	r.sortTasks(&tasks, query.SortBy, query.SortOrder)

	r.logger.Debug(ctx, "tasks found by query", "count", len(tasks))
	return tasks, nil
}

func (r *TaskRepository) Update(ctx context.Context, task *domain.Task) error {
	if task == nil {
		return errors.New("task cannot be nil")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tasks[task.ID]; !exists {
		return fmt.Errorf("task not found: %s", task.ID)
	}

	r.tasks[task.ID] = task
	r.logger.Info(ctx, "task updated", "task_id", task.ID)
	return nil
}

func (r *TaskRepository) Delete(ctx context.Context, id string) error {
	if id == "" {
		return errors.New("task ID cannot be empty")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tasks[id]; !exists {
		return fmt.Errorf("task not found: %s", id)
	}

	delete(r.tasks, id)
	r.logger.Info(ctx, "task deleted", "task_id", id)
	return nil
}

func (r *TaskRepository) FindByCustomerID(ctx context.Context, customerID string) ([]domain.Task, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var tasks []domain.Task
	for _, task := range r.tasks {
		if task.CustomerID != nil && *task.CustomerID == customerID {
			tasks = append(tasks, *task)
		}
	}

	r.logger.Debug(ctx, "tasks found by customer", "customer_id", customerID, "count", len(tasks))
	return tasks, nil
}

func (r *TaskRepository) FindByAssigneeID(ctx context.Context, assigneeID string) ([]domain.Task, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var tasks []domain.Task
	for _, task := range r.tasks {
		if task.AssigneeID == assigneeID {
			tasks = append(tasks, *task)
		}
	}

	r.logger.Debug(ctx, "tasks found by assignee", "assignee_id", assigneeID, "count", len(tasks))
	return tasks, nil
}

func (r *TaskRepository) FindByStatus(ctx context.Context, status domain.TaskStatus) ([]domain.Task, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var tasks []domain.Task
	for _, task := range r.tasks {
		if task.Status == status {
			tasks = append(tasks, *task)
		}
	}

	r.logger.Debug(ctx, "tasks found by status", "status", status, "count", len(tasks))
	return tasks, nil
}

func (r *TaskRepository) FindByType(ctx context.Context, taskType domain.TaskType) ([]domain.Task, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var tasks []domain.Task
	for _, task := range r.tasks {
		if task.Type == taskType {
			tasks = append(tasks, *task)
		}
	}

	r.logger.Debug(ctx, "tasks found by type", "type", taskType, "count", len(tasks))
	return tasks, nil
}

func (r *TaskRepository) FindOpenTasks(ctx context.Context) ([]domain.Task, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var tasks []domain.Task
	for _, task := range r.tasks {
		if task.Status == domain.TaskStatusOpen || task.Status == domain.TaskStatusInProgress {
			tasks = append(tasks, *task)
		}
	}

	r.logger.Debug(ctx, "open tasks found", "count", len(tasks))
	return tasks, nil
}

func (r *TaskRepository) FindSubtasks(ctx context.Context, parentID string) ([]domain.Task, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var tasks []domain.Task
	for _, task := range r.tasks {
		if task.ParentID != nil && *task.ParentID == parentID {
			tasks = append(tasks, *task)
		}
	}

	r.logger.Debug(ctx, "subtasks found", "parent_id", parentID, "count", len(tasks))
	return tasks, nil
}

func (r *TaskRepository) FindBySourceMeta(ctx context.Context, meta map[string]interface{}) ([]domain.Task, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var tasks []domain.Task

	for _, task := range r.tasks {
		if r.matchesSourceMeta(task, meta) {
			tasks = append(tasks, *task)
		}
	}

	r.logger.Debug(ctx, "tasks found by source meta",
		"criteria", meta,
		"count", len(tasks))
	return tasks, nil
}

func (r *TaskRepository) GetStats(ctx context.Context, query ports.StatsQuery) (*ports.TaskStats, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	stats := &ports.TaskStats{
		ByPriority: make(map[domain.Priority]int),
		ByCategory: make(map[string]int),
		BySource:   make(map[domain.TaskSource]int),
		ByType:     make(map[domain.TaskType]int),
	}

	var resolutionTimes []float64

	for _, task := range r.tasks {
		if !r.matchesStatsQuery(task, query) {
			continue
		}

		stats.TotalCount++

		// Считаем по статусам
		switch task.Status {
		case domain.TaskStatusOpen:
			stats.OpenCount++
		case domain.TaskStatusInProgress:
			stats.InProgressCount++
		case domain.TaskStatusResolved:
			stats.ResolvedCount++
		case domain.TaskStatusClosed:
			stats.ClosedCount++
		}

		// Распределение
		stats.ByPriority[task.Priority]++
		stats.ByCategory[task.Category]++
		stats.BySource[task.Source]++
		stats.ByType[task.Type]++

		// Время разрешения
		if task.ResolvedAt != nil && task.CreatedAt.Before(*task.ResolvedAt) {
			resolutionTime := task.ResolvedAt.Sub(task.CreatedAt).Hours()
			resolutionTimes = append(resolutionTimes, resolutionTime)
		}
	}

	// Вычисляем статистику по времени разрешения
	if len(resolutionTimes) > 0 {
		stats.AvgResolutionTime = r.calculateAverage(resolutionTimes)
		stats.MaxResolutionTime = r.calculateMax(resolutionTimes)
		stats.MinResolutionTime = r.calculateMin(resolutionTimes)
	}

	// TODO: Реализовать TopAssignees
	stats.TopAssignees = []ports.AssigneeStats{}

	r.logger.Debug(ctx, "stats calculated", "total_tasks", stats.TotalCount)
	return stats, nil
}

func (r *TaskRepository) GetAssigneeWorkload(ctx context.Context) (map[string]int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	workload := make(map[string]int)
	for _, task := range r.tasks {
		if task.AssigneeID != "" && (task.Status == domain.TaskStatusOpen || task.Status == domain.TaskStatusInProgress) {
			workload[task.AssigneeID]++
		}
	}

	r.logger.Debug(ctx, "assignee workload calculated", "assignee_count", len(workload))
	return workload, nil
}

func (r *TaskRepository) BulkUpdateStatus(ctx context.Context, taskIDs []string, status domain.TaskStatus) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	updatedCount := 0
	for _, id := range taskIDs {
		if task, exists := r.tasks[id]; exists {
			task.Status = status
			task.UpdatedAt = time.Now()
			updatedCount++
		}
	}

	r.logger.Info(ctx, "bulk status update completed",
		"task_count", len(taskIDs),
		"updated_count", updatedCount,
		"status", status,
	)
	return nil
}

// Вспомогательные методы

func (r *TaskRepository) matchesQuery(task *domain.Task, query ports.TaskQuery) bool {
	// Фильтр по типам
	if len(query.Types) > 0 && !containsTaskType(query.Types, task.Type) {
		return false
	}

	// Фильтр по статусам
	if len(query.Statuses) > 0 && !containsTaskStatus(query.Statuses, task.Status) {
		return false
	}

	// Фильтр по приоритетам
	if len(query.Priorities) > 0 && !containsPriority(query.Priorities, task.Priority) {
		return false
	}

	// Фильтр по исполнителю
	if query.AssigneeID != "" && task.AssigneeID != query.AssigneeID {
		return false
	}

	// Фильтр по клиенту
	if query.CustomerID != "" && (task.CustomerID == nil || *task.CustomerID != query.CustomerID) {
		return false
	}

	// Фильтр по автору
	if query.ReporterID != "" && task.ReporterID != query.ReporterID {
		return false
	}

	// Фильтр по источнику
	if len(query.Source) > 0 && !containsTaskSource(query.Source, task.Source) {
		return false
	}

	// Фильтр по тегам
	if len(query.Tags) > 0 && !containsAllTags(task.Tags, query.Tags) {
		return false
	}

	// Фильтр по категории
	if query.Category != "" && task.Category != query.Category {
		return false
	}

	// Фильтр по родительской задаче
	if query.ParentID != nil && (task.ParentID == nil || *task.ParentID != *query.ParentID) {
		return false
	}

	// Фильтр по проекту
	if query.ProjectID != nil && (task.ProjectID == nil || *task.ProjectID != *query.ProjectID) {
		return false
	}

	// TODO: Реализовать фильтр по датам и поиск по тексту

	return true
}

// matchesSourceMeta проверяет соответствие задачи критериям поиска по мета-данным
func (r *TaskRepository) matchesSourceMeta(task *domain.Task, meta map[string]interface{}) bool {
	if task.SourceMeta == nil {
		r.logger.Debug(context.Background(), "task has no source meta", "task_id", task.ID)
		return false
	}

	// ✅ ЛОГИРУЕМ СРАВНЕНИЕ
	r.logger.Debug(context.Background(), "comparing source meta",
		"task_id", task.ID,
		"task_source_meta", task.SourceMeta,
		"search_meta", meta)

	// Поиск по message_id
	if messageID, exists := meta["message_id"]; exists {
		if taskMsgID, exists := task.SourceMeta["message_id"]; exists {
			if taskMsgID == messageID {
				r.logger.Debug(context.Background(), "match found by message_id",
					"task_id", task.ID, "message_id", messageID)
				return true
			}
		}
	}

	// Поиск по in_reply_to
	if inReplyTo, exists := meta["in_reply_to"]; exists {
		if taskInReplyTo, exists := task.SourceMeta["in_reply_to"]; exists {
			if taskInReplyTo == inReplyTo {
				r.logger.Debug(context.Background(), "match found by in_reply_to",
					"task_id", task.ID, "in_reply_to", inReplyTo)
				return true
			}
		}
	}

	// Поиск по references (цепочка писем)
	if references, exists := meta["references"]; exists {
		if refs, ok := references.([]string); ok {
			if taskRefs, exists := task.SourceMeta["references"]; exists {
				if taskRefsSlice, ok := taskRefs.([]string); ok {
					// Проверяем пересечение references
					for _, ref := range refs {
						for _, taskRef := range taskRefsSlice {
							if ref == taskRef {
								r.logger.Debug(context.Background(), "match found by references",
									"task_id", task.ID, "reference", ref)
								return true
							}
						}
					}
				}
			}
		}
	}

	r.logger.Debug(context.Background(), "no match found for task",
		"task_id", task.ID)
	return false
}

func (r *TaskRepository) matchesStatsQuery(task *domain.Task, query ports.StatsQuery) bool {
	// Фильтр по клиенту
	if query.CustomerID != "" && (task.CustomerID == nil || *task.CustomerID != query.CustomerID) {
		return false
	}

	// Фильтр по исполнителю
	if query.AssigneeID != "" && task.AssigneeID != query.AssigneeID {
		return false
	}

	// Фильтр по категории
	if query.Category != "" && task.Category != query.Category {
		return false
	}

	// Фильтр по источнику
	if len(query.Source) > 0 && !containsTaskSource(query.Source, task.Source) {
		return false
	}

	// Фильтр по типам
	if len(query.Types) > 0 && !containsTaskType(query.Types, task.Type) {
		return false
	}

	// TODO: Реализовать фильтр по датам

	return true
}

func (r *TaskRepository) sortTasks(tasks *[]domain.Task, sortBy, sortOrder string) {
	if sortBy == "" {
		return
	}

	sort.Slice(*tasks, func(i, j int) bool {
		taskA := (*tasks)[i]
		taskB := (*tasks)[j]

		var less bool
		switch sortBy {
		case "created_at":
			less = taskA.CreatedAt.Before(taskB.CreatedAt)
		case "updated_at":
			less = taskA.UpdatedAt.Before(taskB.UpdatedAt)
		case "priority":
			less = r.priorityValue(taskA.Priority) < r.priorityValue(taskB.Priority)
		case "status":
			less = string(taskA.Status) < string(taskB.Status)
		default:
			return false
		}

		if sortOrder == "desc" {
			return !less
		}
		return less
	})
}

func (r *TaskRepository) priorityValue(priority domain.Priority) int {
	priorityValues := map[domain.Priority]int{
		domain.PriorityLow:      1,
		domain.PriorityMedium:   2,
		domain.PriorityHigh:     3,
		domain.PriorityCritical: 4,
	}
	return priorityValues[priority]
}

func (r *TaskRepository) calculateAverage(values []float64) float64 {
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func (r *TaskRepository) calculateMax(values []float64) float64 {
	max := values[0]
	for _, v := range values {
		if v > max {
			max = v
		}
	}
	return max
}

func (r *TaskRepository) calculateMin(values []float64) float64 {
	min := values[0]
	for _, v := range values {
		if v < min {
			min = v
		}
	}
	return min
}

// Вспомогательные функции для проверки вхождения

func containsTaskType(types []domain.TaskType, target domain.TaskType) bool {
	for _, t := range types {
		if t == target {
			return true
		}
	}
	return false
}

func containsTaskStatus(statuses []domain.TaskStatus, target domain.TaskStatus) bool {
	for _, s := range statuses {
		if s == target {
			return true
		}
	}
	return false
}

func containsPriority(priorities []domain.Priority, target domain.Priority) bool {
	for _, p := range priorities {
		if p == target {
			return true
		}
	}
	return false
}

func containsTaskSource(sources []domain.TaskSource, target domain.TaskSource) bool {
	for _, s := range sources {
		if s == target {
			return true
		}
	}
	return false
}

func containsAllTags(taskTags, queryTags []string) bool {
	for _, queryTag := range queryTags {
		found := false
		for _, taskTag := range taskTags {
			if taskTag == queryTag {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}
