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
	if len(req.Dependencies) == 0 {
		query := `INSERT INTO tasks (target_url, payload, run_at, priority)
		VALUES ($1, $2, $3, $4)
		RETURNING id, target_url, payload, run_at, status, priority;`
		err := t.db.QueryRowContext(ctx, query, req.TargetURL, req.Payload, req.RunAt, req.Priority).
			Scan(&task.ID, &task.TargetURL, &task.Payload, &task.RunAt, &task.Status, &task.Priority)
		if err != nil {
			return nil, err
		}
		return task, nil
	}

	tx, err := t.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	query := `INSERT INTO tasks (target_url, payload, run_at, priority, status)
		VALUES ($1, $2, $3, $4, 'waiting')
		RETURNING id, target_url, payload, run_at, status, priority;`

	err = tx.QueryRowContext(ctx, query, req.TargetURL, req.Payload, req.RunAt, req.Priority).
		Scan(&task.ID, &task.TargetURL, &task.Payload, &task.RunAt, &task.Status, &task.Priority)
	if err != nil {
		return nil, err
	}

	query2 := `INSERT INTO task_dependencies (parent_id, child_id) VALUES ($1, $2)`
	for _, parentID := range req.Dependencies {
		_, err := tx.ExecContext(ctx, query2, parentID, task.ID)
		if err != nil {
			return nil, err
		}
	}
	err = tx.Commit()
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
		ORDER BY priority DESC, run_at ASC
		FOR UPDATE SKIP LOCKED
		LIMIT 10
	) RETURNING id, target_url, payload, priority;`
	rows, err := t.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		task := &models.Tasks{}
		err := rows.Scan(&task.ID, &task.TargetURL, &task.Payload, &task.Priority)
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
	if status != "succeed" {

		query := `UPDATE tasks SET status = $1 WHERE id = $2;`
		_, err := t.db.ExecContext(ctx, query, status, taskID)
		if err != nil {
			return err
		}
		return nil
	}
	tx, err := t.db.BeginTx(ctx, nil)
	defer tx.Rollback()

	query1 := `UPDATE tasks SET status = 'succeed' WHERE id = $1`
	_, err = tx.ExecContext(ctx, query1, taskID)
	if err != nil {
		return err
	}
	query2 := `UPDATE tasks SET status = 'pending'
		WHERE id IN (
			SELECT child_id FROM task_dependencies WHERE parent_id = $1
		)
		AND status = 'waiting'
		AND NOT EXISTS (
			SELECT 1 FROM task_dependencies td
			JOIN tasks t ON t.id = td.parent_id
			WHERE td.child_id = tasks.id AND t.status != 'succeed'
		);
	`
	_, err = tx.ExecContext(ctx, query2, taskID)
	if err != nil {
		return err
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func (t *TasksRepo) MarkTaskStatusFailed(ctx context.Context, taskID string) error {
	query := `UPDATE tasks
		SET
			status = CASE
						WHEN retry_count < max_retries THEN 'pending'
						ELSE 'failed'
					 END,
			run_at = CASE
						WHEN retry_count < max_retries THEN NOW() + INTERVAL '30 seconds'
						ELSE run_at
					 END,
			retry_count = retry_count + 1
		WHERE id = $1;
	`
	_, err := t.db.ExecContext(ctx, query, taskID)
	if err != nil {
		return err
	}
	return nil
}
