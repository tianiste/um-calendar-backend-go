package middleware

import (
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type ipRateLimiter struct {
	mu       sync.Mutex
	visitors map[string]*visitor
	limit    rate.Limit
	burst    int
}

func NewIPRateLimiter(limit rate.Limit, burst int, cleanupInterval time.Duration, staleAfter time.Duration) gin.HandlerFunc {
	rl := &ipRateLimiter{
		visitors: make(map[string]*visitor),
		limit:    limit,
		burst:    burst,
	}

	go rl.cleanupVisitors(cleanupInterval, staleAfter)

	return func(ctx *gin.Context) {
		ip := ctx.ClientIP()
		limiter := rl.getVisitorLimiter(ip)
		if !limiter.Allow() {
			ctx.JSON(429, gin.H{
				"error": "too many requests",
			})
			ctx.Abort()
			return
		}

		ctx.Next()
	}
}

func (rl *ipRateLimiter) getVisitorLimiter(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	if value, exists := rl.visitors[ip]; exists {
		value.lastSeen = time.Now().UTC()
		return value.limiter
	}

	limiter := rate.NewLimiter(rl.limit, rl.burst)
	rl.visitors[ip] = &visitor{
		limiter:  limiter,
		lastSeen: time.Now().UTC(),
	}

	return limiter
}

func (rl *ipRateLimiter) cleanupVisitors(cleanupInterval time.Duration, staleAfter time.Duration) {
	ticker := time.NewTicker(cleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		cutoff := time.Now().UTC().Add(-staleAfter)

		rl.mu.Lock()
		for ip, value := range rl.visitors {
			if value.lastSeen.Before(cutoff) {
				delete(rl.visitors, ip)
			}
		}
		rl.mu.Unlock()
	}
}
