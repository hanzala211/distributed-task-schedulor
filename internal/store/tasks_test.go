package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/hanzala211/go-backend-template/internal/models"
	"github.com/hanzala211/go-backend-template/internal/testutils/dbtest"
	"github.com/stretchr/testify/require"
)

var (
	testUtils *dbtest.PostgresTestSuite
	testDB    *sql.DB
)

func TestMain(m *testing.M) {
	testUtils = dbtest.SetupSuite()
	if testUtils.DB == nil {
		panic("dbtest.SetupSuite failed")
	}

	err := testUtils.RunMigrations()
	if err != nil {
		panic(err)
	}
	testDB = testUtils.DB
	code := m.Run()
	if err := testUtils.Close(); err != nil {
		panic(err)
	}
	os.Exit(code)
}

func newRepo() *TasksRepo {
	return NewTaskRepo(testDB)
}

func TestTasksRepo_InsertTask_WithoutDependencies(t *testing.T) {
	testUtils.ResetDB(t)

	repo := newRepo()
	ctx := context.Background()
	runAt := time.Now().Add(1 * time.Minute)

	input := &models.Tasks{
		TargetURL: "https://example.com",
		Payload:   json.RawMessage(`{"a":1}`),
		RunAt:     runAt,
		Priority:  10,
		Status:    "pending",
	}

	got, err := repo.InsertTask(ctx, input, nil)
	require.NoError(t, err)
	require.NotNil(t, got)
	require.NotEmpty(t, got.ID)
	require.Equal(t, input.TargetURL, got.TargetURL)
	require.JSONEq(t, string(input.Payload), string(got.Payload))
	require.Equal(t, input.Status, got.Status)
	require.Equal(t, input.Priority, got.Priority)
}

func TestTasksRepo_InsertTask_WithDependencies(t *testing.T) {
	testUtils.ResetDB(t)

	repo := newRepo()
	ctx := context.Background()

	parent1, err := repo.InsertTask(ctx, &models.Tasks{
		TargetURL: "https://parent1.com",
		Payload:   json.RawMessage(`{}`),
		RunAt:     time.Now(),
		Priority:  5,
		Status:    "succeed",
	}, nil)
	require.NoError(t, err)

	parent2, err := repo.InsertTask(ctx, &models.Tasks{
		TargetURL: "https://parent2.com",
		Payload:   json.RawMessage(`{}`),
		RunAt:     time.Now(),
		Priority:  5,
		Status:    "succeed",
	}, nil)
	require.NoError(t, err)

	child, err := repo.InsertTask(ctx, &models.Tasks{
		TargetURL: "https://child.com",
		Payload:   json.RawMessage(`{"job":"x"}`),
		RunAt:     time.Now(),
		Priority:  1,
		Status:    "waiting",
	}, []string{parent1.ID, parent2.ID})
	require.NoError(t, err)
	require.NotNil(t, child)

	var count int
	err = testDB.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM task_dependencies WHERE child_id = $1
	`, child.ID).Scan(&count)
	require.NoError(t, err)
	require.Equal(t, 2, count)
}

func TestTasksRepo_InsertTask_WithMissingDependency_ReturnsNotFound(t *testing.T) {
	testUtils.ResetDB(t)

	repo := newRepo()
	ctx := context.Background()

	_, err := repo.InsertTask(ctx, &models.Tasks{
		TargetURL: "https://child.com",
		Payload:   json.RawMessage(`{}`),
		RunAt:     time.Now(),
		Priority:  1,
		Status:    "waiting",
	}, []string{"00000000-0000-0000-0000-000000000000"})

	require.ErrorIs(t, err, ErrNotFound)
}

func TestTasksRepo_FetchDueTasks(t *testing.T) {
	testUtils.ResetDB(t)

	ctx := context.Background()

	_, err := testDB.ExecContext(ctx, `
		INSERT INTO tasks (id, target_url, payload, run_at, priority, status)
		VALUES
			(gen_random_uuid(), 'https://a.com', '{}', NOW() - INTERVAL '10 minutes', 1, 'pending'),
			(gen_random_uuid(), 'https://b.com', '{}', NOW() - INTERVAL '5 minutes', 10, 'pending'),
			(gen_random_uuid(), 'https://c.com', '{}', NOW() + INTERVAL '30 minutes', 100, 'pending')
	`)
	require.NoError(t, err)

	repo := newRepo()
	tasks, err := repo.FetchDueTasks(ctx, 2)
	require.NoError(t, err)
	require.Len(t, tasks, 2)

	require.Len(t, tasks, 2)

	urls := []string{tasks[0].TargetURL, tasks[1].TargetURL}
	require.ElementsMatch(t, []string{"https://a.com", "https://b.com"}, urls)

	var runningCount int
	err = testDB.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM tasks WHERE status = 'running'
	`).Scan(&runningCount)
	require.NoError(t, err)
	require.Equal(t, 2, runningCount)
}

func TestTasksRepo_UpdateTaskStatus(t *testing.T) {
	testUtils.ResetDB(t)

	repo := newRepo()
	ctx := context.Background()

	task, err := repo.InsertTask(ctx, &models.Tasks{
		TargetURL: "https://example.com",
		Payload:   json.RawMessage(`{}`),
		RunAt:     time.Now(),
		Priority:  1,
		Status:    "pending",
	}, nil)
	require.NoError(t, err)

	err = repo.UpdateTaskStatus(ctx, "running", task.ID)
	require.NoError(t, err)

	var status string
	err = testDB.QueryRowContext(ctx, `SELECT status FROM tasks WHERE id = $1`, task.ID).Scan(&status)
	require.NoError(t, err)
	require.Equal(t, "running", status)
}

func TestTasksRepo_UpdateTaskStatus_NotFound(t *testing.T) {
	testUtils.ResetDB(t)

	repo := newRepo()
	ctx := context.Background()

	err := repo.UpdateTaskStatus(ctx, "running", "00000000-0000-0000-0000-000000000000")
	require.ErrorIs(t, err, ErrNotFound)
}

func TestTasksRepo_MarkTaskSucceed_UnblocksChildren(t *testing.T) {
	testUtils.ResetDB(t)

	repo := newRepo()
	ctx := context.Background()

	parent1, err := repo.InsertTask(ctx, &models.Tasks{
		TargetURL: "https://p1.com",
		Payload:   json.RawMessage(`{}`),
		RunAt:     time.Now(),
		Priority:  1,
		Status:    "running",
	}, nil)
	require.NoError(t, err)

	parent2, err := repo.InsertTask(ctx, &models.Tasks{
		TargetURL: "https://p2.com",
		Payload:   json.RawMessage(`{}`),
		RunAt:     time.Now(),
		Priority:  1,
		Status:    "succeed",
	}, nil)
	require.NoError(t, err)

	child, err := repo.InsertTask(ctx, &models.Tasks{
		TargetURL: "https://child.com",
		Payload:   json.RawMessage(`{}`),
		RunAt:     time.Now(),
		Priority:  1,
		Status:    "waiting",
	}, []string{parent1.ID, parent2.ID})
	require.NoError(t, err)

	err = repo.MarkTaskSucceed(ctx, parent1.ID)
	require.NoError(t, err)

	var parentStatus string
	err = testDB.QueryRowContext(ctx, `SELECT status FROM tasks WHERE id = $1`, parent1.ID).Scan(&parentStatus)
	require.NoError(t, err)
	require.Equal(t, "succeed", parentStatus)

	var childStatus string
	err = testDB.QueryRowContext(ctx, `SELECT status FROM tasks WHERE id = $1`, child.ID).Scan(&childStatus)
	require.NoError(t, err)
	require.Equal(t, "pending", childStatus)
}

func TestTasksRepo_MarkTaskStatusFailed(t *testing.T) {
	testUtils.ResetDB(t)

	repo := newRepo()
	ctx := context.Background()

	task, err := repo.InsertTask(ctx, &models.Tasks{
		TargetURL: "https://example.com",
		Payload:   json.RawMessage(`{"x":1}`),
		RunAt:     time.Now().UTC(),
		Priority:  1,
		Status:    "running",
	}, nil)
	require.NoError(t, err)

	task.Status = "pending"
	task.RunAt = time.Now().UTC().Add(10 * time.Minute)
	task.RetryCount = 2

	err = repo.MarkTaskStatusFailed(ctx, task)
	require.NoError(t, err)

	var status string
	var retryCount int
	var runAt time.Time

	err = testDB.QueryRowContext(ctx,
		`SELECT status, retry_count, run_at FROM tasks WHERE id = $1`,
		task.ID,
	).Scan(&status, &retryCount, &runAt)
	require.NoError(t, err)

	require.Equal(t, "pending", status)
	require.Equal(t, 2, retryCount)
	require.WithinDuration(t, task.RunAt, runAt, time.Second)
}
