package middleware

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// RedisRateLimiter implements a sliding window rate limiter backed by Redis,
// with automatic fallback to in-memory limiting when Redis is unavailable.
type RedisRateLimiter struct {
	client   *redis.Client
	limit    int
	window   time.Duration
	fallback *rateLimiter
}

// NewRedisRateLimiter creates a Redis-backed rate limiter.
// Pass a nil client to use in-memory limiting only.
func NewRedisRateLimiter(client *redis.Client, limit int, window time.Duration) *RedisRateLimiter {
	return &RedisRateLimiter{
		client:   client,
		limit:    limit,
		window:   window,
		fallback: newRateLimiter(limit, window),
	}
}

// allow checks whether the given IP is within the rate limit using a Redis
// sorted-set sliding window. Falls back to in-memory on any Redis error.
func (r *RedisRateLimiter) allow(ctx context.Context, ip string) bool {
	if r.client == nil {
		return r.fallback.allow(ip)
	}

	now := time.Now()
	windowStart := now.Add(-r.window)
	key := fmt.Sprintf("rate_limit:%s", ip)

	pipe := r.client.Pipeline()
	pipe.ZRemRangeByScore(ctx, key, "0", fmt.Sprintf("%d", windowStart.UnixNano()))
	countCmd := pipe.ZCard(ctx, key)
	pipe.ZAdd(ctx, key, redis.Z{Score: float64(now.UnixNano()), Member: now.UnixNano()})
	pipe.Expire(ctx, key, r.window)

	if _, err := pipe.Exec(ctx); err != nil {
		return r.fallback.allow(ip)
	}

	return countCmd.Val() < int64(r.limit)
}

// rateLimiter is the in-memory sliding window fallback.
type rateLimiter struct {
	requests map[string][]time.Time
	mu       sync.Mutex
	limit    int
	window   time.Duration
}

func newRateLimiter(limit int, window time.Duration) *rateLimiter {
	rl := &rateLimiter{
		requests: make(map[string][]time.Time),
		limit:    limit,
		window:   window,
	}
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			rl.cleanup()
		}
	}()
	return rl
}

func (rl *rateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	now := time.Now()
	for ip, timestamps := range rl.requests {
		valid := []time.Time{}
		for _, ts := range timestamps {
			if now.Sub(ts) < rl.window {
				valid = append(valid, ts)
			}
		}
		if len(valid) == 0 {
			delete(rl.requests, ip)
		} else {
			rl.requests[ip] = valid
		}
	}
}

func (rl *rateLimiter) allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	now := time.Now()
	timestamps := rl.requests[ip]
	valid := []time.Time{}
	for _, ts := range timestamps {
		if now.Sub(ts) < rl.window {
			valid = append(valid, ts)
		}
	}
	if len(valid) >= rl.limit {
		return false
	}
	rl.requests[ip] = append(valid, now)
	return true
}

// RateLimitMiddleware limits requests per IP using in-memory sliding window.
func RateLimitMiddleware(limit int, window time.Duration) gin.HandlerFunc {
	limiter := newRateLimiter(limit, window)
	return func(c *gin.Context) {
		if !limiter.allow(c.ClientIP()) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "Rate limit exceeded",
				"code":  "RATE_LIMIT_EXCEEDED",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

// RateLimitMiddlewareWithRedis limits requests per IP using a Redis sliding window.
// Automatically falls back to in-memory limiting when Redis is unavailable.
func RateLimitMiddlewareWithRedis(client *redis.Client, limit int, window time.Duration) gin.HandlerFunc {
	limiter := NewRedisRateLimiter(client, limit, window)
	return func(c *gin.Context) {
		if !limiter.allow(c.Request.Context(), c.ClientIP()) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "Rate limit exceeded",
				"code":  "RATE_LIMIT_EXCEEDED",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}
