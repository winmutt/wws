package middleware

import (
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// RateLimiter implements a simple in-memory rate limiter using token bucket algorithm
type RateLimiter struct {
	mu              sync.RWMutex
	clients         map[string]*ClientRateLimiter
	cleanupInterval time.Duration
}

// ClientRateLimiter tracks rate limiting for a single client
type ClientRateLimiter struct {
	tokens     float64
	maxTokens  float64
	lastUpdate time.Time
}

// RateLimitConfig holds configuration for rate limiting
type RateLimitConfig struct {
	// Requests per minute for authenticated users
	AuthenticatedRPM int
	// Requests per minute for anonymous users
	AnonymousRPM int
	// Requests per minute for API endpoints (more restrictive)
	APIRPM int
	// Requests per minute for workspace operations
	WorkspaceRPM int
	// Cleanup interval for stale entries
	CleanupInterval time.Duration
}

// DefaultRateLimitConfig returns default rate limiting configuration
func DefaultRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		AuthenticatedRPM: 100,
		AnonymousRPM:     30,
		APIRPM:           60,
		WorkspaceRPM:     20,
		CleanupInterval:  time.Minute * 5,
	}
}

// NewRateLimiter creates a new rate limiter with the given configuration
func NewRateLimiter(config RateLimitConfig) *RateLimiter {
	// Ensure cleanup interval is reasonable
	cleanupInterval := config.CleanupInterval
	if cleanupInterval <= 0 {
		cleanupInterval = time.Minute * 5
	}

	rl := &RateLimiter{
		clients:         make(map[string]*ClientRateLimiter),
		cleanupInterval: cleanupInterval,
	}

	// Start cleanup goroutine
	go rl.cleanupStaleEntries()

	return rl
}

// cleanupStaleEntries removes rate limiter entries older than cleanup interval
func (rl *RateLimiter) cleanupStaleEntries() {
	ticker := time.NewTicker(rl.cleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for key, client := range rl.clients {
			if now.Sub(client.lastUpdate) > rl.cleanupInterval*2 {
				delete(rl.clients, key)
			}
		}
		rl.mu.Unlock()
	}
}

// getClientID extracts a unique identifier for the client
func (rl *RateLimiter) getClientID(r *http.Request) string {
	// Check for API key header first
	apiKey := r.Header.Get("X-API-Key")
	if apiKey != "" {
		return "api:" + apiKey
	}

	// Check for user ID in context (set by auth middleware)
	if userID, ok := r.Context().Value("user_id").(int); ok && userID > 0 {
		return "user:" + strconv.Itoa(userID)
	}

	// Fall back to IP address
	ipAddress := r.Header.Get("X-Forwarded-For")
	if ipAddress == "" {
		ipAddress = r.RemoteAddr
	}
	// Clean up IP address (remove port if present)
	if idx := strings.Index(ipAddress, ":"); idx > 0 {
		ipAddress = ipAddress[:idx]
	}
	return "ip:" + ipAddress
}

// getLimitForPath determines the rate limit based on the request path
func (rl *RateLimiter) getLimitForPath(r *http.Request, config RateLimitConfig) int {
	path := strings.ToLower(r.URL.Path)

	// More restrictive limits for workspace operations
	if strings.Contains(path, "/workspaces/") {
		return config.WorkspaceRPM
	}

	// API endpoints have moderate limits
	if strings.Contains(path, "/api/") {
		return config.APIRPM
	}

	// Check if user is authenticated
	if _, ok := r.Context().Value("user").(interface{ GetID() int }); ok {
		return config.AuthenticatedRPM
	}

	// Anonymous users get the lowest limit
	return config.AnonymousRPM
}

// Allow checks if a request should be allowed and returns rate limit headers
func (rl *RateLimiter) Allow(r *http.Request, config RateLimitConfig) (bool, map[string]string) {
	clientID := rl.getClientID(r)
	limit := rl.getLimitForPath(r, config)

	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	client, exists := rl.clients[clientID]

	if !exists {
		// New client, create entry with full tokens
		rl.clients[clientID] = &ClientRateLimiter{
			tokens:     float64(limit),
			maxTokens:  float64(limit),
			lastUpdate: now,
		}
		// Consume one token for this request
		rl.clients[clientID].tokens--
		remaining := int(rl.clients[clientID].tokens)
		if remaining < 0 {
			remaining = 0
		}
		return true, createRateLimitHeaders(limit, remaining)
	}

	// Calculate token refresh based on time elapsed
	timeElapsed := now.Sub(client.lastUpdate).Seconds()
	// Add tokens based on rate (limit tokens per 60 seconds)
	tokenIncrease := timeElapsed * (float64(limit) / 60.0)
	client.tokens = min(client.tokens+tokenIncrease, client.maxTokens)
	client.lastUpdate = now

	// Check if we have tokens available
	if client.tokens >= 1.0 {
		client.tokens--
		remaining := int(client.tokens)
		if remaining < 0 {
			remaining = 0
		}
		return true, createRateLimitHeaders(limit, remaining)
	}

	// Rate limit exceeded
	return false, createRateLimitHeaders(limit, 0)
}

// createRateLimitHeaders creates rate limit header values
func createRateLimitHeaders(limit, remaining int) map[string]string {
	return map[string]string{
		"X-RateLimit-Limit":     strconv.Itoa(limit),
		"X-RateLimit-Remaining": strconv.Itoa(remaining),
		"X-RateLimit-Reset":     strconv.Itoa(int(time.Now().Add(time.Minute).Unix())),
	}
}

// min returns the smaller of two float64 values
func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// RateLimitMiddleware returns a middleware that enforces rate limiting
func RateLimitMiddleware(config RateLimitConfig) func(http.Handler) http.Handler {
	limiter := NewRateLimiter(config)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip rate limiting for health checks and static assets
			if strings.HasPrefix(r.URL.Path, "/health") ||
				strings.HasPrefix(r.URL.Path, "/static") ||
				strings.HasPrefix(r.URL.Path, "/api/docs") {
				next.ServeHTTP(w, r)
				return
			}

			allowed, headers := limiter.Allow(r, config)

			// Add rate limit headers to response
			for key, value := range headers {
				w.Header().Set(key, value)
			}

			if !allowed {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				w.Write([]byte(`{"error": "Rate limit exceeded. Please try again later."}`))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RateLimitChecker provides a way to check rate limits manually
type RateLimitChecker struct {
	limiter *RateLimiter
	config  RateLimitConfig
}

// NewRateLimitChecker creates a new rate limit checker
func NewRateLimitChecker(config RateLimitConfig) *RateLimitChecker {
	return &RateLimitChecker{
		limiter: NewRateLimiter(config),
		config:  config,
	}
}

// Check checks if a request is allowed and returns remaining requests
func (rlc *RateLimitChecker) Check(r *http.Request) (allowed bool, remaining int) {
	allowed, headers := rlc.limiter.Allow(r, rlc.config)
	if remainingStr, ok := headers["X-RateLimit-Remaining"]; ok {
		remaining, _ = strconv.Atoi(remainingStr)
	}
	return allowed, remaining
}
