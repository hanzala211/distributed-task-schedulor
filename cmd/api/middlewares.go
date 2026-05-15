package main

import (
	"context"
	"fmt"
	"net/http"
)

func (app *application) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		if token == "" {
			app.writeJSONError(w, http.StatusUnauthorized, "Unauthorized")
			return
		}
		userId, err := app.jwtAuthenticator.ValidateToken(token)
		if err != nil {
			app.writeJSONError(w, http.StatusUnauthorized, "Invalid token")
			return
		}
		ctx := context.WithValue(r.Context(), "user_id", userId)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (app *application) RateLimiterMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if app.config.rateLimiterConfig.Enabled {
			ip := r.RemoteAddr
			allowed, wait := app.rateLimiter.Allow(ip)
			if !allowed {
				app.writeJSONError(w, http.StatusTooManyRequests, fmt.Sprintf("Too many requests retry after %v seconds", int(wait.Seconds())))
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}
