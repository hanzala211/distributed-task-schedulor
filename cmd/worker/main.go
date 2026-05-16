package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/hanzala211/go-backend-template/internal/db"
	"github.com/hanzala211/go-backend-template/internal/env"
	"github.com/hanzala211/go-backend-template/internal/models"
	"github.com/hanzala211/go-backend-template/internal/store"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
)

var httpClient = &http.Client{
	Timeout: 10 * time.Second,
	Transport: &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 100,
		IdleConnTimeout:     90 * time.Second,
	},
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Printf("Failed to load .env: %v\n", err)
	}
	loggerProd, _ := zap.NewDevelopment()
	defer loggerProd.Sync()
	logger := loggerProd.Sugar()
	db := db.New(fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		env.GetEnv("DB_USER", "postgres"),
		env.GetEnv("DB_PASSWORD", "postgres"),
		env.GetEnv("DB_HOST", "localhost"),
		env.GetEnv("DB_PORT", "5432"),
		env.GetEnv("DB_NAME", "postgres"),
	))
	taskStore := store.NewTaskRepo(db)
	store := store.NewStorage(taskStore)
	ticker := time.NewTicker(1 * time.Second)
	for range ticker.C {
		ctx := context.Background()
		tasks, err := store.Tasks.FetchDueTasks(ctx)
		if err != nil {
			logger.Error("Failed to fetch tasks", err)
		}

		for _, task := range tasks {
			logger.Infow("Executing task", "task_id", task.ID, "target", task.TargetURL)
			go callTask(ctx, task, logger, store)
		}
	}
}

func callTask(ctx context.Context, task *models.Tasks, logger *zap.SugaredLogger, store *store.Storage) {
	httpReq, err := http.NewRequest("POST", task.TargetURL, bytes.NewBuffer(task.Payload))
	if err != nil {
		store.Tasks.ChangeTaskStatus(ctx, task.ID, "failed")
		logger.Errorw("Failed to create the HTTP Req", "error", err, "taskId", task.ID)
		return
	}
	httpReq.Header.Add("Content-Type", "application/json")
	resp, err := httpClient.Do(httpReq)
	if err != nil {
		store.Tasks.ChangeTaskStatus(ctx, task.ID, "failed")
		logger.Errorw("Webhook delivery failed", "error", err, "taskId", task.ID)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 200 && resp.StatusCode <= 299 {
		logger.Infow("Webhook delivered successfully", "status", resp.StatusCode, "task_id", task.ID)
		store.Tasks.ChangeTaskStatus(ctx, task.ID, "succeed")
	} else {
		logger.Errorw("Webhook rejected by target", "status", resp.StatusCode, "task_id", task.ID)
		store.Tasks.ChangeTaskStatus(ctx, task.ID, "failed")
	}
}
