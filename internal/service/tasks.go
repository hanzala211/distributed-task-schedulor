package service

import (
	"context"
	"fmt"
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
	res, err := t.store.Tasks.InsertTask(ctx, task, req.Dependencies)
	if err != nil {
		return nil, &AppError{
			Message: "failed to insert task",
			Err:     fmt.Errorf("store error: %w", err),
		}
	}
	return res, nil
}

func (t *TaskService) HandleTaskFailure(ctx context.Context, task *models.Tasks, maxRetries int) error {
	task.RetryCount++
	if task.RetryCount >= maxRetries {
		task.Status = "failed"
	} else {
		task.Status = "pending"
		task.RunAt = time.Now().UTC().Add(30 * time.Second)
	}
	err := t.store.Tasks.MarkTaskStatusFailed(ctx, task)
	if err != nil {
		return &AppError{
			Message: fmt.Sprintf("failed to handle task failure for task %s", task.ID),
			Err:     fmt.Errorf("store error: %w", err),
		}
	}
	return nil
}

func (t *TaskService) ChangeStatus(ctx context.Context, task *models.Tasks, newStatus string) error {
	var err error
	if newStatus == "succeed" {
		err = t.store.Tasks.MarkTaskSucceed(ctx, task.ID)
	} else {
		err = t.store.Tasks.UpdateTaskStatus(ctx, newStatus, task.ID)
	}

	if err != nil {
		return &AppError{
			Message: fmt.Sprintf("failed to change status to %s for task %s", newStatus, task.ID),
			Err:     fmt.Errorf("store error: %w", err),
		}
	}
	return nil
}

func (t *TaskService) FetchDueTasks(ctx context.Context, batchSize int) ([]*models.Tasks, error) {
	tasks, err := t.store.Tasks.FetchDueTasks(ctx, batchSize)
	if err != nil {
		return nil, &AppError{
			Message: "failed to fetch due tasks",
			Err:     fmt.Errorf("store error: %w", err),
		}
	}
	return tasks, nil
}
