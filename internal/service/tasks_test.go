package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/hanzala211/go-backend-template/internal/models"
	"github.com/hanzala211/go-backend-template/internal/store"
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
	if err := testUtils.RunMigrations(); err != nil {
		panic(err)
	}
	testDB = testUtils.DB
	code := m.Run()
	if err := testUtils.Close(); err != nil {
		panic(err)
	}
	os.Exit(code)
}

func newService() *TaskService {
	taskSvc := store.NewTaskRepo(testDB)
	storage := store.NewStorage(taskSvc)

	return NewTaskService(storage)
}

func TestTaskService_InsertTask_Integration(t *testing.T) {
	testUtils.ResetDB(t)
	svc := newService()
	req := &models.AddTaskAPIDTO{
		TargetURL: "https://example.com",
		Payload:   json.RawMessage(`{"a":1}`),
		RunAt:     time.Now().UTC(),
		Priority:  1,
	}
	ctx := context.Background()
	got, err := svc.InsertTask(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, got)
	require.NotEmpty(t, got.ID)

	require.Equal(t, req.TargetURL, got.TargetURL)
	require.JSONEq(t, string(req.Payload), string(got.Payload))
	require.Equal(t, req.Priority, got.Priority)
	require.WithinDuration(t, req.RunAt, got.RunAt, time.Second)
	require.Equal(t, "pending", got.Status)
}

func TestTaskService_InsertTask_Integration_WithDependencies(t *testing.T) {
	testUtils.ResetDB(t)
	svc := newService()
	req := &models.AddTaskAPIDTO{
		TargetURL: "https://example.com",
		Payload:   json.RawMessage(`{"a":1}`),
		RunAt:     time.Now().UTC(),
		Priority:  1,
	}
	ctx := context.Background()
	got, err := svc.InsertTask(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, got)
	require.NotEmpty(t, got.ID)

	require.Equal(t, req.TargetURL, got.TargetURL)
	require.JSONEq(t, string(req.Payload), string(got.Payload))
	require.Equal(t, req.Priority, got.Priority)
	require.WithinDuration(t, req.RunAt, got.RunAt, time.Second)
	require.Equal(t, "pending", got.Status)
	req2 := &models.AddTaskAPIDTO{
		TargetURL: "https://example.com",
		Payload:   json.RawMessage(`{"a":1}`),
		RunAt:     time.Now().UTC(),
		Priority:  1,
		Dependencies: []string{
			got.ID,
		},
	}
	_, err = svc.InsertTask(ctx, req2)
	require.NoError(t, err)
	require.NotNil(t, got)
	require.NotEmpty(t, got.ID)
	require.Equal(t, req2.TargetURL, got.TargetURL)
}

func TestTaskService_HandleTaskFailure_Integration(t *testing.T) {
	testUtils.ResetDB(t)

	svc := newService()
	req := &models.AddTaskAPIDTO{
		TargetURL: "https://example.com",
		Payload:   json.RawMessage(`{"a":1}`),
		RunAt:     time.Now().UTC(),
		Priority:  1,
	}

	ctx := context.Background()
	got, err := svc.InsertTask(ctx, req)
	require.NoError(t, err)

	before := time.Now().UTC()

	err = svc.HandleTaskFailure(ctx, got, 2)
	require.NoError(t, err)

	after := time.Now().UTC()

	var status string
	var retryCount int
	var runAt time.Time

	err = testDB.QueryRowContext(ctx,
		`SELECT status, retry_count, run_at FROM tasks WHERE id = $1`,
		got.ID,
	).Scan(&status, &retryCount, &runAt)
	require.NoError(t, err)

	require.Equal(t, "pending", status)
	require.Equal(t, 1, retryCount)

	expectedMin := before.Add(30 * time.Second)
	expectedMax := after.Add(30 * time.Second)

	runAtUTC := runAt.UTC()
	require.False(t, runAtUTC.Before(expectedMin), "run_at is earlier than expected retry window")
	require.False(t, runAtUTC.After(expectedMax), "run_at is later than expected retry window")
}

func TestTaskService_ChangeStatus_Integration(t *testing.T) {
	testUtils.ResetDB(t)

	svc := newService()
	req := &models.AddTaskAPIDTO{
		TargetURL: "https://example.com",
		Payload:   json.RawMessage(`{"a":1}`),
		RunAt:     time.Now().UTC(),
		Priority:  1,
	}

	ctx := context.Background()
	got, err := svc.InsertTask(ctx, req)
	require.NoError(t, err)

	child := &models.AddTaskAPIDTO{
		TargetURL:    "https://example.com",
		Payload:      json.RawMessage(`{"a":1}`),
		RunAt:        time.Now().UTC().Add(1 * time.Hour),
		Priority:     1,
		Dependencies: []string{got.ID},
	}

	childTask, err := svc.InsertTask(ctx, child)
	require.NoError(t, err)
	err = svc.ChangeStatus(ctx, got, "succeed")
	require.NoError(t, err)

	var status string
	err = testDB.QueryRowContext(ctx, `SELECT status FROM tasks WHERE id = $1`, got.ID).Scan(&status)
	require.NoError(t, err)
	require.Equal(t, "succeed", status)
	var childStatus string
	err = testDB.QueryRowContext(ctx, `SELECT status FROM tasks WHERE id = $1`, childTask.ID).Scan(&childStatus)
	require.NoError(t, err)
	require.Equal(t, "pending", childStatus)
	req2 := &models.AddTaskAPIDTO{
		TargetURL: "https://example.com",
		Payload:   json.RawMessage(`{"a":1}`),
		RunAt:     time.Now().UTC(),
		Priority:  1,
	}

	_, err = svc.InsertTask(ctx, req2)
	require.NoError(t, err)

	err = svc.ChangeStatus(ctx, got, "pending")
	require.NoError(t, err)

	var status2 string
	err = testDB.QueryRowContext(ctx, `SELECT status FROM tasks WHERE id = $1`, got.ID).Scan(&status2)
	require.NoError(t, err)
	require.Equal(t, "pending", status2)
}

func TestTaskService_FetchDueTasks_Integration(t *testing.T) {
	testUtils.ResetDB(t)

	svc := newService()
	req := &models.AddTaskAPIDTO{
		TargetURL: "https://example.com",
		Payload:   json.RawMessage(`{"a":1}`),
		RunAt:     time.Now().UTC(),
		Priority:  1,
	}
	ctx := context.Background()
	got, err := svc.InsertTask(ctx, req)
	require.NoError(t, err)

	req2 := &models.AddTaskAPIDTO{
		TargetURL: "https://example.com",
		Payload:   json.RawMessage(`{"a":1}`),
		RunAt:     time.Now().UTC().Add(1 * time.Hour),
		Priority:  1,
		Dependencies: []string{
			got.ID,
		},
	}
	_, err = svc.InsertTask(ctx, req2)
	require.NoError(t, err)

	tasks, err := svc.FetchDueTasks(ctx, 2)
	require.NoError(t, err)
	require.Len(t, tasks, 1) // only one task is due for current time
}
