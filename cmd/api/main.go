package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hanzala211/go-backend-template/internal/auth"
	"github.com/hanzala211/go-backend-template/internal/db"
	"github.com/hanzala211/go-backend-template/internal/env"
	"github.com/hanzala211/go-backend-template/internal/ratelimiter"
	"github.com/hanzala211/go-backend-template/internal/service"
	"github.com/hanzala211/go-backend-template/internal/store"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Error loading .env file", err)
	}
	cfg := config{
		addr: ":" + env.GetEnv("ADDR", "4000"),
		dbConfig: dbConfig{
			host:     env.GetEnv("DB_HOST", "localhost"),
			port:     env.GetEnv("DB_PORT", "5432"),
			user:     env.GetEnv("DB_USER", "postgres"),
			password: env.GetEnv("DB_PASSWORD", "postgres"),
			dbname:   env.GetEnv("DB_NAME", "postgres"),
		},
		jwtConfig: jwtConfig{
			secret:     env.GetEnv("JWT_SECRET", "secret"),
			expiryTime: time.Now().Add(24 * time.Hour),
		},
		rateLimiterConfig: ratelimiter.RateLimiterConfig{
			MaxRequests: env.GetEnvInt("RATE_LIMITER_MAX_REQUESTS", 2),
			Duration:    time.Duration(env.GetEnvInt("RATE_LIMITER_DURATION", 5)) * time.Second,
			Enabled:     env.GetEnvBool("RATE_LIMITER_ENABLED", true),
		},
	}
	fmt.Println(cfg.rateLimiterConfig)

	loggerProd, _ := zap.NewDevelopment()
	defer loggerProd.Sync()
	db := db.New(fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		cfg.dbConfig.user,
		cfg.dbConfig.password,
		cfg.dbConfig.host,
		cfg.dbConfig.port,
		cfg.dbConfig.dbname,
	))
	logger := loggerProd.Sugar()
	jwtAuthenticator := auth.NewJWTAuthenticator(cfg.jwtConfig.secret, cfg.jwtConfig.expiryTime)
	taskRepo := store.NewTaskRepo(db)
	storage := store.NewStorage(taskRepo)
	taskService := service.NewTaskService(storage)
	service := service.NewService(taskService)
	app := application{
		service:          service,
		config:           cfg,
		logger:           logger,
		db:               db,
		store:            storage,
		jwtAuthenticator: jwtAuthenticator,
		rateLimiter:      ratelimiter.NewFixedWindowLimiter(cfg.rateLimiterConfig.MaxRequests, cfg.rateLimiterConfig.Duration),
	}
	srv := app.serve()
	serverErr := make(chan error, 1) // one buffer because multiple errors can cause the channel to miss the real err
	go func() {
		logger.Infow("server started", zap.String("addr", srv.Addr))
		if err := srv.ListenAndServe(); err != nil {
			serverErr <- err
		}
		close(serverErr)
	}()
	stopSign := make(chan os.Signal, 1)
	signal.Notify(stopSign, os.Interrupt, syscall.SIGTERM)
	select {
	case err := <-serverErr:
		logger.Fatalw("server error", zap.Error(err))
	case <-stopSign:
		logger.Info("shutdown signal received, gracefully shutting down server")
	}
	shutDownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutDownCtx); err != nil {
		logger.Fatalw("server shutdown error", zap.Error(err))
	}
	logger.Info("server stopped gracefully")
}
