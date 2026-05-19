package store

import (
	"context"
	"database/sql"
	"strings"

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

func (t *TasksRepo) InsertTask(ctx context.Context, tasksModel *models.Tasks, dependencies []string) (*models.Tasks, error) {
	task := &models.Tasks{}
	if len(dependencies) == 0 {
		query := `INSERT INTO tasks (target_url, payload, run_at, priority, status)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, target_url, payload, run_at, status, priority;`
		err := t.db.QueryRowContext(ctx, query, tasksModel.TargetURL, tasksModel.Payload, tasksModel.RunAt, tasksModel.Priority, tasksModel.Status).
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
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, target_url, payload, run_at, status, priority;`

	err = tx.QueryRowContext(ctx, query, tasksModel.TargetURL, tasksModel.Payload, tasksModel.RunAt, tasksModel.Priority, tasksModel.Status).
		Scan(&task.ID, &task.TargetURL, &task.Payload, &task.RunAt, &task.Status, &task.Priority)
	if err != nil {
		return nil, err
	}

	query2 := `INSERT INTO task_dependencies (parent_id, child_id) VALUES ($1, $2)`
	for _, parentID := range dependencies {
		_, err := tx.ExecContext(ctx, query2, parentID, task.ID)
		if err != nil {
			if strings.Contains(err.Error(), "foreign key constraint") {
				return nil, ErrNotFound
			} else if strings.Contains(err.Error(), "unique constraint") {
				return nil, ErrConflict
			}
			return nil, err
		}
	}
	err = tx.Commit()
	if err != nil {
		return nil, err
	}
	return task, nil
}

func (t *TasksRepo) FetchDueTasks(ctx context.Context, batchSize int) ([]*models.Tasks, error) {
	tasks := []*models.Tasks{}
	query := `UPDATE tasks SET status = 'running', started_at = NOW() WHERE id IN (
		SELECT id FROM tasks t
		WHERE
		(status = 'pending' AND run_at <= NOW())
		 OR (status = 'running' AND started_at < NOW() - INTERVAL '5 minutes')
		ORDER BY priority DESC, run_at ASC
		FOR UPDATE SKIP LOCKED
		LIMIT $1
	) RETURNING id, target_url, payload, priority;`
	rows, err := t.db.QueryContext(ctx, query, batchSize)
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

func (t *TasksRepo) UpdateTaskStatus(ctx context.Context, status string, taskID string) error {
	query := `UPDATE tasks SET status = $1 WHERE id = $2;`
	res, err := t.db.ExecContext(ctx, query, status, taskID)
	if err != nil {
		return err
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (t *TasksRepo) MarkTaskSucceed(ctx context.Context, taskID string) error {
	tx, err := t.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query1 := `UPDATE tasks SET status = 'succeed' WHERE id = $1`
	res, err := tx.ExecContext(ctx, query1, taskID)
	if err != nil {
		return err
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrNotFound
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

func (t *TasksRepo) MarkTaskStatusFailed(ctx context.Context, task *models.Tasks) error {
	query := `UPDATE tasks
		SET status = $2, run_at = $3, retry_count = $4
		WHERE id = $1;
	`
	res, err := t.db.ExecContext(ctx, query, task.ID, task.Status, task.RunAt, task.RetryCount)
	if err != nil {
		return err
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
