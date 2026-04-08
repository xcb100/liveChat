package middleware

import (
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

type rateLimiterEntry struct {
	limiter    *rate.Limiter
	lastAccess time.Time
}

type RateLimiter struct {
	burst     int
	limit     rate.Limit
	entries   map[string]*rateLimiterEntry
	entryLock sync.Mutex
}

// NewRateLimiter 输入每秒令牌数和突发容量，输出为限流器实例，目的在于为请求级限流提供可复用对象。
func NewRateLimiter(requestsPerSecond float64, burst int) *RateLimiter {
	return &RateLimiter{
		burst:   burst,
		limit:   rate.Limit(requestsPerSecond),
		entries: make(map[string]*rateLimiterEntry),
	}
}

// Middleware 输入为空，输出为 Gin 中间件，目的在于基于客户端地址和路由进行滑动令牌桶限流。
func (rateLimiter *RateLimiter) Middleware() gin.HandlerFunc {
	go rateLimiter.cleanupExpiredEntries()
	return func(c *gin.Context) {
		entryKey := c.ClientIP() + ":" + c.Request.Method + ":" + c.FullPath()
		limiter := rateLimiter.getLimiter(entryKey)
		if limiter.Allow() {
			c.Next()
			return
		}
		c.AbortWithStatusJSON(429, gin.H{
			"code": 429,
			"msg":  "请求过于频繁，请稍后重试",
		})
	}
}

// getLimiter 输入限流键，输出为令牌桶实例，目的在于按需创建并复用请求限流器。
func (rateLimiter *RateLimiter) getLimiter(key string) *rate.Limiter {
	rateLimiter.entryLock.Lock()
	defer rateLimiter.entryLock.Unlock()

	entry, exists := rateLimiter.entries[key]
	if !exists {
		entry = &rateLimiterEntry{
			limiter:    rate.NewLimiter(rateLimiter.limit, rateLimiter.burst),
			lastAccess: time.Now(),
		}
		rateLimiter.entries[key] = entry
		return entry.limiter
	}

	entry.lastAccess = time.Now()
	return entry.limiter
}

// cleanupExpiredEntries 输入为空，输出为过期限流器清理结果，目的在于避免长时间运行后的内存堆积。
func (rateLimiter *RateLimiter) cleanupExpiredEntries() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		expireBefore := time.Now().Add(-10 * time.Minute)
		rateLimiter.entryLock.Lock()
		for key, entry := range rateLimiter.entries {
			if entry.lastAccess.Before(expireBefore) {
				delete(rateLimiter.entries, key)
			}
		}
		rateLimiter.entryLock.Unlock()
	}
}
