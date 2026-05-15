package service

import (
	"context"

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
	return t.store.Tasks.InsertTask(ctx, req)
}
