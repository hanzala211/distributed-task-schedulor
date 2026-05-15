package store

import (
	"context"
	"database/sql"

	"github.com/hanzala211/go-backend-template/internal/models"
)

type TasksRepo struct {
	db *sql.DB
}

func NewTaskRepo(db *sql.DB) *TasksRepo {
	return &TasksRepo{
		db: db,
	}
}

func (t *TasksRepo) InsertTask(ctx context.Context, req *models.AddTaskAPIDTO) (*models.Tasks, error) {
	task := &models.Tasks{}
	query := `INSERT INTO tasks (target_url, payload, run_at)
	VALUES ($1, $2, $3)
	RETURNING id, target_url, payload, run_at, status;`
	err := t.db.QueryRowContext(ctx, query, req.TargetURL, req.Payload, req.RunAt).
		Scan(&task.ID, &task.TargetURL, &task.Payload, &task.RunAt, &task.Status)
	if err != nil {
		return nil, err
	}
	return task, nil
}
