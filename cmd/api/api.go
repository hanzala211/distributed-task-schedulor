package main

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/hanzala211/go-backend-template/internal/auth"
	"github.com/hanzala211/go-backend-template/internal/ratelimiter"
	"github.com/hanzala211/go-backend-template/internal/service"
	"github.com/hanzala211/go-backend-template/internal/store"
	"go.uber.org/zap"
)

type application struct {
	config           config
	logger           *zap.SugaredLogger
	db               *sql.DB
	store            *store.Storage
	jwtAuthenticator *auth.JWTAuthenticator
	rateLimiter      *ratelimiter.FixedWindowLimiter
	service          *service.Service
}

type config struct {
	addr              string
	dbConfig          dbConfig
	jwtConfig         jwtConfig
	rateLimiterConfig ratelimiter.RateLimiterConfig
}

type jwtConfig struct {
	secret     string
	expiryTime time.Time
}

type dbConfig struct {
	host     string
	port     string
	user     string
	password string
	dbname   string
}

func (app *application) serve() *http.Server {
	r := chi.NewRouter()
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{"https://*", "http://*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
	}))
	if app.config.rateLimiterConfig.Enabled {
		r.Use(app.RateLimiterMiddleware)
	}
	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/tasks", app.addTask)
	})
	return &http.Server{
		Addr:    app.config.addr,
		Handler: r,
	}
}
