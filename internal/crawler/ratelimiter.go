package crawler

import "time"

// RateLimiter throttles operations to at most one per interval.
type RateLimiter struct {
	ticker *time.Ticker
}

// NewRateLimiter creates a limiter that ticks once per interval.
func NewRateLimiter(interval time.Duration) *RateLimiter {
	return &RateLimiter{
		ticker: time.NewTicker(interval),
	}
}

// Wait blocks until the next tick.
func (r *RateLimiter) Wait() {
	<-r.ticker.C
}

// Stop releases the underlying ticker.
func (r *RateLimiter) Stop() {
	r.ticker.Stop()
}
