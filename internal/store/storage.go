package store

import (
	"context"

	"github.com/hanzala211/go-backend-template/internal/models"
)

type Storage struct {
	Tasks interface {
		InsertTask(ctx context.Context, tasksModel *models.Tasks, dependencies []string) (*models.Tasks, error)
		FetchDueTasks(ctx context.Context, batchSize int) ([]*models.Tasks, error)
		MarkTaskSucceed(ctx context.Context, taskID string) error
		MarkTaskStatusFailed(ctx context.Context, task *models.Tasks) error
		UpdateTaskStatus(ctx context.Context, status string, taskID string) error
	}
}

func NewStorage(t *TasksRepo) *Storage {
	return &Storage{
		Tasks: t,
	}
}
