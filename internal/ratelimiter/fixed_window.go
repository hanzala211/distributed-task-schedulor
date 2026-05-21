package ratelimiter

import (
	"sync"
	"time"
)

type clientWindow struct {
	count   int
	resetAt time.Time
}

type FixedWindowLimiter struct {
	mu      sync.Mutex
	Limit   int
	Window  time.Duration
	clients map[string]*clientWindow
}

func NewFixedWindowLimiter(limit int, window time.Duration) *FixedWindowLimiter {
	limiter := &FixedWindowLimiter{
		Limit:   limit,
		Window:  window,
		clients: make(map[string]*clientWindow),
	}
	go limiter.cleanup()
	return limiter
}

func (f *FixedWindowLimiter) Allow(ip string) (bool, time.Duration) {
	f.mu.Lock()
	defer f.mu.Unlock()

	now := time.Now()
	cw, exists := f.clients[ip]

	if !exists || now.After(cw.resetAt) {
		f.clients[ip] = &clientWindow{
			count:   1,
			resetAt: now.Add(f.Window),
		}
		return true, 0
	}

	if cw.count < f.Limit {
		cw.count++
		return true, 0
	}

	return false, time.Until(cw.resetAt)
}

func (f *FixedWindowLimiter) cleanup() {
	ticker := time.NewTicker(f.Window)
	defer ticker.Stop()

	for range ticker.C {
		f.mu.Lock()
		now := time.Now()
		for ip, cw := range f.clients {
			if now.After(cw.resetAt) {
				delete(f.clients, ip)
			}
		}
		f.mu.Unlock()
	}
}
