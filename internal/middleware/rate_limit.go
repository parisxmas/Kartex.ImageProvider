package middleware

import (
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type RateLimiter struct {
	requests map[string][]time.Time
	mu       sync.Mutex
	limit    int
	window   time.Duration
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		requests: make(map[string][]time.Time),
		limit:    limit,
		window:   window,
	}
}

func (rl *RateLimiter) Allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-rl.window)

	// Clean up old requests
	requests := rl.requests[ip]
	var validRequests []time.Time
	for _, t := range requests {
		if t.After(windowStart) {
			validRequests = append(validRequests, t)
		}
	}
	rl.requests[ip] = validRequests

	// Check if we're over the limit
	if len(validRequests) >= rl.limit {
		return false
	}

	// Add new request
	rl.requests[ip] = append(validRequests, now)
	return true
}

func RateLimitMiddleware() gin.HandlerFunc {
	// Get rate limit from environment or use default
	limitStr := os.Getenv("RATE_LIMIT")
	limit := 100 // default value
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}

	// Get window from environment or use default
	windowStr := os.Getenv("RATE_LIMIT_WINDOW")
	window := time.Minute // default value
	if windowStr != "" {
		if w, err := strconv.Atoi(windowStr); err == nil {
			window = time.Duration(w) * time.Second
		}
	}

	limiter := NewRateLimiter(limit, window)

	return func(c *gin.Context) {
		ip := c.ClientIP()
		if !limiter.Allow(ip) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "Too many requests. Please try again later.",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}
