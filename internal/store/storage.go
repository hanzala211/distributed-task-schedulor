package store

import (
	"context"

	"github.com/hanzala211/go-backend-template/internal/models"
)

type Storage struct {
	Tasks interface {
		InsertTask(ctx context.Context, req *models.AddTaskAPIDTO) (*models.Tasks, error)
		FetchDueTasks(ctx context.Context) ([]*models.Tasks, error)
		ChangeTaskStatus(ctx context.Context, taskID string, status string) error
		MarkTaskStatusFailed(ctx context.Context, taskID string) error
	}
}

func NewStorage(t *TasksRepo) *Storage {
	return &Storage{
		Tasks: t,
	}
}
