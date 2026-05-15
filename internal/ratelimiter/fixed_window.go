package ratelimiter

import (
	"sync"
	"time"
)

type FixedWindowLimiter struct {
	sync.RWMutex
	Limit   int
	Window  time.Duration
	clients map[string]int
}

func NewFixedWindowLimiter(limit int, window time.Duration) *FixedWindowLimiter {
	return &FixedWindowLimiter{
		Limit:   limit,
		Window:  window,
		clients: make(map[string]int),
	}
}

func (f *FixedWindowLimiter) Allow(ip string) (bool, time.Duration) {
	f.RLock()
	count, exists := f.clients[ip]
	f.RUnlock()

	if !exists || count < f.Limit {
		f.Lock()
		if !exists {
			go f.resetCount(ip)
		}
		f.clients[ip]++
		f.Unlock()
		return true, 0
	}
	return false, f.Window
}

func (f *FixedWindowLimiter) resetCount(ip string) {
	time.Sleep(f.Window)
	f.Lock()
	delete(f.clients, ip)
	f.Unlock()
}
