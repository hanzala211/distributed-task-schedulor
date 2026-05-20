package service

import (
	"context"

	"github.com/hanzala211/go-backend-template/internal/models"
)

type Service struct {
	Task interface {
		InsertTask(ctx context.Context, req *models.AddTaskAPIDTO) (*models.Tasks, error)
		HandleTaskFailure(ctx context.Context, task *models.Tasks, maxRetries int) error
		ChangeStatus(ctx context.Context, task *models.Tasks, newStatus string) error
		FetchDueTasks(ctx context.Context, batchSize int) ([]*models.Tasks, error)
	}
}

func NewService(t *TaskService) *Service {
	return &Service{
		Task: t,
	}
}
