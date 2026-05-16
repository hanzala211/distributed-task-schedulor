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

func (t *TasksRepo) FetchDueTasks(ctx context.Context) ([]*models.Tasks, error) {
	tasks := []*models.Tasks{}
	query := `UPDATE tasks SET status = 'running' WHERE id IN (
		SELECT id FROM tasks t
		WHERE status = 'pending' AND run_at <= NOW()
		ORDER BY run_at ASC
		FOR UPDATE SKIP LOCKED
		LIMIT 10
	) RETURNING id, target_url, payload;`
	rows, err := t.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		task := &models.Tasks{}
		err := rows.Scan(&task.ID, &task.TargetURL, &task.Payload)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return tasks, nil
}

func (t *TasksRepo) ChangeTaskStatus(ctx context.Context, taskID string, status string) error {
	query := `UPDATE tasks SET status = $1 WHERE id = $2;`
	_, err := t.db.ExecContext(ctx, query, status, taskID)
	if err != nil {
		return err
	}
	return nil
}
