package service

import (
	"context"
	"time"

	"github.com/hanzala211/go-backend-template/internal/models"
	"github.com/hanzala211/go-backend-template/internal/store"
)

type TaskService struct {
	store *store.Storage
}

func NewTaskService(store *store.Storage) *TaskService {
	return &TaskService{
		store: store,
	}
}

func (t *TaskService) InsertTask(ctx context.Context, req *models.AddTaskAPIDTO) (*models.Tasks, error) {
	task := &models.Tasks{
		Priority:  req.Priority,
		Payload:   req.Payload,
		Status:    "pending",
		RunAt:     req.RunAt,
		TargetURL: req.TargetURL,
	}
	if len(req.Dependencies) > 0 {
		task.Status = "waiting"
	}
	return t.store.Tasks.InsertTask(ctx, task, req.Dependencies)
}

func (t *TaskService) HandleTaskFailure(ctx context.Context, task *models.Tasks, maxRetries int) error {
	task.RetryCount++
	if task.RetryCount >= maxRetries {
		task.Status = "failed"
	} else {
		task.Status = "pending"
		task.RunAt = time.Now().Add(30 * time.Second)
	}
	return t.store.Tasks.MarkTaskStatusFailed(ctx, task)
}

func (t *TaskService) ChangeStatus(ctx context.Context, task *models.Tasks, newStatus string) error {
	if newStatus == "succeed" {
		return t.store.Tasks.MarkTaskSucceed(ctx, task.ID)
	}
	return t.store.Tasks.UpdateTaskStatus(ctx, newStatus, task.ID)
}

func (t *TaskService) FetchDueTasks(ctx context.Context, batchSize int) ([]*models.Tasks, error) {
	return t.store.Tasks.FetchDueTasks(ctx, batchSize)
}
