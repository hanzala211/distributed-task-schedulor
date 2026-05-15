package store

import (
	"context"

	"github.com/hanzala211/go-backend-template/internal/models"
)

type Storage struct {
	Tasks interface {
		InsertTask(ctx context.Context, req *models.AddTaskAPIDTO) (*models.Tasks, error)
	}
}

func NewStorage(t *TasksRepo) *Storage {
	return &Storage{
		Tasks: t,
	}
}
