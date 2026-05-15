package ratelimiter

import "time"

type RateLimiterConfig struct {
	MaxRequests int
	Duration    time.Duration
	Enabled     bool
}
