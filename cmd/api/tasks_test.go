package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/hanzala211/go-backend-template/internal/models"
	"github.com/hanzala211/go-backend-template/internal/service"
	"github.com/hanzala211/go-backend-template/internal/store"
	"github.com/hanzala211/go-backend-template/internal/testutils/dbtest"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
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
func newApp() *application {
	taskRepo := store.NewTaskRepo(testDB)
	store := store.NewStorage(taskRepo)
	taskService := service.NewTaskService(store)
	svc := service.NewService(taskService)
	return &application{
		service: svc,
		logger:  zap.NewNop().Sugar(),
		db:      testDB,
		store:   store,
		config:  config{},
	}
}

func TestTasksHandler_AddTask_Success(t *testing.T) {
	testUtils.ResetDB(t)
	app := newApp()
	req := models.AddTaskAPIDTO{
		TargetURL: "https://example.com",
		Payload:   []byte(`{"a":1}`),
		RunAt:     time.Now().UTC().Truncate(time.Microsecond),
		Priority:  1,
	}
	jsonByte, err := json.Marshal(req)
	if err != nil {
		panic(err)
	}
	httpReq := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", bytes.NewReader(jsonByte))
	httpReq.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()
	app.addTask(w, httpReq)
	require.Equal(t, http.StatusCreated, w.Code)
	type response struct {
		Data models.Tasks `json:"data"`
	}
	var resp response
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	if err != nil {
		panic(err)
	}
	require.Equal(t, resp.Data.TargetURL, req.TargetURL)
	require.Equal(t, resp.Data.Payload, req.Payload)
	require.Equal(t, resp.Data.RunAt, req.RunAt)
	require.Equal(t, resp.Data.Status, "pending")
	require.Equal(t, resp.Data.Priority, req.Priority)
}

func TestTaskHandler_AddTask_WithDependencies(t *testing.T) {
	testUtils.ResetDB(t)
	app := newApp()
	req := models.AddTaskAPIDTO{
		TargetURL: "https://example.com",
		Payload:   []byte(`{"a":1}`),
		RunAt:     time.Now().UTC().Truncate(time.Microsecond),
		Priority:  1,
	}
	jsonByte, err := json.Marshal(req)
	if err != nil {
		panic(err)
	}
	httpReq := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", bytes.NewReader(jsonByte))
	httpReq.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()
	app.addTask(w, httpReq)
	require.Equal(t, http.StatusCreated, w.Code)
	type response struct {
		Data models.Tasks `json:"data"`
	}
	var resp response
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	if err != nil {
		panic(err)
	}
	reqWithDeps := models.AddTaskAPIDTO{
		TargetURL: "https://example.com",
		Payload:   []byte(`{"a":1}`),
		RunAt:     time.Now().UTC().Truncate(time.Microsecond),
		Priority:  1,
		Dependencies: []string{
			resp.Data.ID,
		},
	}
	jsonByte, err = json.Marshal(reqWithDeps)
	if err != nil {
		panic(err)
	}
	httpReq = httptest.NewRequest(http.MethodPost, "/api/v1/tasks", bytes.NewReader(jsonByte))
	httpReq.Header.Add("Content-Type", "application/json")
	w = httptest.NewRecorder()
	app.addTask(w, httpReq)
	require.Equal(t, http.StatusCreated, w.Code)
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	if err != nil {
		panic(err)
	}
	require.Equal(t, resp.Data.TargetURL, reqWithDeps.TargetURL)
	require.Equal(t, resp.Data.Payload, reqWithDeps.Payload)
	require.Equal(t, resp.Data.RunAt, reqWithDeps.RunAt)
	require.Equal(t, resp.Data.Status, "waiting")
	require.Equal(t, resp.Data.Priority, reqWithDeps.Priority)
}

func TestTaskHandler_AddTask_WithDependencies_NotFound(t *testing.T) {
	testUtils.ResetDB(t)
	app := newApp()
	req := models.AddTaskAPIDTO{
		TargetURL: "https://example.com",
		Payload:   []byte(`{"a":1}`),
		RunAt:     time.Now().UTC().Truncate(time.Microsecond),
		Priority:  1,
		Dependencies: []string{
			"00000000-0000-0000-0000-000000000000",
		},
	}
	jsonByte, err := json.Marshal(req)
	if err != nil {
		panic(err)
	}
	httpReq := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", bytes.NewReader(jsonByte))
	httpReq.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()
	app.addTask(w, httpReq)
	type response struct {
		Status  string `json:"status"`
		Message string `json:"message"`
	}
	var resp response
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	if err != nil {
		panic(err)
	}
	require.Equal(t, http.StatusBadRequest, w.Code)
	require.Equal(t, "One or more dependent task do not exist", resp.Message)
}

func TestTaskHandler_AddTask_MissingRequiredFields(t *testing.T) {
	testUtils.ResetDB(t)
	app := newApp()
	req := models.AddTaskAPIDTO{
		TargetURL: "",
		Payload:   []byte(`{"a":1}`),
		RunAt:     time.Now().UTC().Truncate(time.Microsecond),
		Priority:  1,
	}
	jsonByte, err := json.Marshal(req)
	if err != nil {
		panic(err)
	}
	httpReq := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", bytes.NewReader(jsonByte))
	httpReq.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()
	app.addTask(w, httpReq)
	type response struct {
		Status  string `json:"status"`
		Message string `json:"message"`
	}
	var resp response
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	if err != nil {
		panic(err)
	}
	require.Equal(t, http.StatusBadRequest, w.Code)
}
