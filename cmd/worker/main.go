package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/http"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/hanzala211/go-backend-template/internal/db"
	"github.com/hanzala211/go-backend-template/internal/env"
	"github.com/hanzala211/go-backend-template/internal/models"
	"github.com/hanzala211/go-backend-template/internal/service"
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

const maxRetries = 3
const batchSize = 10

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Printf("Failed to load .env: %v\n", err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
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
	taskService := service.NewTaskService(store)
	service := service.NewService(taskService)
	ticker := time.NewTicker(1 * time.Second)
	var wg sync.WaitGroup

	tasksChan := make(chan *models.Tasks, 100)
	const workersNum = 3
	for w := 0; w < workersNum; w++ {
		wg.Add(1)
		go workerNode(tasksChan, logger, service, w, &wg)
	}
	keepRunning := true
	for keepRunning {
		select {
		case <-ctx.Done():
			logger.Info("Got Shutdown signal")
			ticker.Stop()
			keepRunning = false
		case <-ticker.C:
			tasks, err := service.Task.FetchDueTasks(ctx, batchSize)
			if err != nil {
				logger.Error("Failed to fetch tasks", err)
			}

			for _, task := range tasks {
				logger.Infow("Executing task", "task_id", task.ID, "target", task.TargetURL)
				tasksChan <- task
			}
		}
	}
	close(tasksChan)
	wg.Wait()
	logger.Info("All Channels are closed!")
}

func workerNode(tasksChan chan *models.Tasks, logger *zap.SugaredLogger, service *service.Service, workerID int, wg *sync.WaitGroup) {
	defer wg.Done()
	for task := range tasksChan {
		processTask(task, service, logger, maxRetries, workerID)
	}
} // test

func processTask(task *models.Tasks, service *service.Service, logger *zap.SugaredLogger, maxRetries int, workerID int) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	httpReq, err := http.NewRequest("POST", task.TargetURL, bytes.NewBuffer(task.Payload))
	if err != nil {
		service.Task.HandleTaskFailure(ctx, task, maxRetries)
		logger.Errorw("Failed to create the HTTP Req", "error", err, "taskId", task.ID, "Prority", task.Priority, "Worker", workerID)
		return
	}
	httpReq.Header.Add("Content-Type", "application/json")
	resp, err := httpClient.Do(httpReq)
	if err != nil {
		err := service.Task.HandleTaskFailure(ctx, task, maxRetries)
		if err != nil {
			logger.Errorw("CRITICAL: Failed to update database status", "error", err, "task_id", task.ID, "Priority", task.Priority, "Worker", workerID)
		}
		logger.Errorw("Webhook delivery failed", "error", err, "taskId", task.ID, "Priority", task.Priority, "Worker", workerID)
		return
	}
	if resp.StatusCode >= 200 && resp.StatusCode <= 299 {
		logger.Infow("Webhook delivered successfully", "status", resp.StatusCode, "task_id", task.ID, "Priority", task.Priority, "Worker", workerID)
		service.Task.ChangeStatus(ctx, task, "succeed")
	} else {
		logger.Errorw("Webhook rejected by target", "status", resp.StatusCode, "task_id", task.ID, "Priority", task.Priority, "Worker", workerID)
		service.Task.HandleTaskFailure(ctx, task, maxRetries)
	}
	resp.Body.Close()
}
