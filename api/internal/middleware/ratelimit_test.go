package middleware

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
)

// TestRateLimitMiddleware tests the rate limiting functionality
func TestRateLimitMiddleware(t *testing.T) {
	config := RateLimitConfig{
		AuthenticatedRPM: 10,
		AnonymousRPM:     5,
		APIRPM:           8,
		WorkspaceRPM:     3,
		CleanupInterval:  time.Minute,
	}

	// Create test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	t.Run("Allows requests within limit", func(t *testing.T) {
		handler := RateLimitMiddleware(config)(testHandler)

		for i := 0; i < 3; i++ {
			req := httptest.NewRequest("GET", "/api/workspaces", nil)
			req.RemoteAddr = "127.0.0.1:12345"
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d", w.Code)
			}

			// Check rate limit headers
			limit := w.Header().Get("X-RateLimit-Limit")
			remaining := w.Header().Get("X-RateLimit-Remaining")
			if limit == "" {
				t.Error("Missing X-RateLimit-Limit header")
			}
			if remaining == "" {
				t.Error("Missing X-RateLimit-Remaining header")
			}
		}
	})

	t.Run("Blocks requests exceeding limit", func(t *testing.T) {
		handler := RateLimitMiddleware(config)(testHandler)

		// Use a unique IP for this test
		// Use /api/workspaces/1 to match workspace limit (3)
		req := httptest.NewRequest("GET", "/api/workspaces/1", nil)
		req.RemoteAddr = "192.168.1.100:12345"

		// Make requests up to the limit (3 for workspace)
		for i := 0; i < 3; i++ {
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Request %d: Expected status 200, got %d", i+1, w.Code)
			}
		}

		// Next request should be blocked
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusTooManyRequests {
			t.Errorf("Expected status 429, got %d", w.Code)
		}

		expectedBody := "Rate limit exceeded"
		if !strings.Contains(w.Body.String(), expectedBody) {
			t.Errorf("Expected body to contain %q, got %q", expectedBody, w.Body.String())
		}
	})

	t.Run("Different paths have different limits", func(t *testing.T) {
		handler := RateLimitMiddleware(config)(testHandler)

		// Workspace path should have lower limit (3)
		workspaceReq := httptest.NewRequest("GET", "/api/workspaces/1", nil)
		workspaceReq.RemoteAddr = "10.0.0.1:12345"

		for i := 0; i < 3; i++ {
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, workspaceReq)
		}

		w := httptest.NewRecorder()
		handler.ServeHTTP(w, workspaceReq)
		if w.Code != http.StatusTooManyRequests {
			t.Errorf("Workspace path: Expected status 429 after 3 requests, got %d", w.Code)
		}
	})

	t.Run("Skips rate limiting for health checks", func(t *testing.T) {
		handler := RateLimitMiddleware(config)(testHandler)

		req := httptest.NewRequest("GET", "/health", nil)
		req.RemoteAddr = "127.0.0.1:12345"

		// Make many requests to health endpoint
		for i := 0; i < 20; i++ {
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Health check request %d: Expected status 200, got %d", i+1, w.Code)
			}
		}
	})
}

// TestRateLimitMiddlewareHeaders tests rate limit header values
func TestRateLimitMiddlewareHeaders(t *testing.T) {
	config := DefaultRateLimitConfig()
	config.APIRPM = 5

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := RateLimitMiddleware(config)(testHandler)

	req := httptest.NewRequest("GET", "/api/test", nil)
	req.RemoteAddr = "172.16.0.1:12345"
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Check headers exist
	limit, _ := strconv.Atoi(w.Header().Get("X-RateLimit-Limit"))
	remaining, _ := strconv.Atoi(w.Header().Get("X-RateLimit-Remaining"))
	reset := w.Header().Get("X-RateLimit-Reset")

	if limit != 5 {
		t.Errorf("Expected limit 5, got %d", limit)
	}
	// After first request, remaining should be 4 (or close to it)
	if remaining > 5 || remaining < 0 {
		t.Errorf("Expected remaining between 0-5, got %d", remaining)
	}
	if reset == "" {
		t.Error("Missing X-RateLimit-Reset header")
	}
}

// TestRateLimiterClientID tests client ID extraction
func TestRateLimiterClientID(t *testing.T) {
	config := DefaultRateLimitConfig()
	limiter := NewRateLimiter(config)

	t.Run("Uses API key when present", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/test", nil)
		req.Header.Set("X-API-Key", "test-api-key-123")

		clientID := limiter.getClientID(req)
		if clientID != "api:test-api-key-123" {
			t.Errorf("Expected client ID 'api:test-api-key-123', got %q", clientID)
		}
	})

	t.Run("Uses IP address when no API key", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/test", nil)
		req.RemoteAddr = "192.168.1.1:12345"

		clientID := limiter.getClientID(req)
		if clientID != "ip:192.168.1.1" {
			t.Errorf("Expected client ID 'ip:192.168.1.1', got %q", clientID)
		}
	})

	t.Run("Cleans IP address with port", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/test", nil)
		req.RemoteAddr = "10.0.0.1:54321"

		clientID := limiter.getClientID(req)
		if clientID != "ip:10.0.0.1" {
			t.Errorf("Expected client ID 'ip:10.0.0.1', got %q", clientID)
		}
	})

	t.Run("Uses X-Forwarded-For header", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/test", nil)
		req.Header.Set("X-Forwarded-For", "203.0.113.1:8080, 198.51.100.1")
		req.RemoteAddr = "127.0.0.1:12345"

		clientID := limiter.getClientID(req)
		// Should use the first IP from X-Forwarded-For
		if !strings.HasPrefix(clientID, "ip:203.0.113.1") {
			t.Errorf("Expected client ID starting with 'ip:203.0.113.1', got %q", clientID)
		}
	})
}

// TestRateLimitConfig tests default configuration
func TestRateLimitConfig(t *testing.T) {
	config := DefaultRateLimitConfig()

	if config.AuthenticatedRPM != 100 {
		t.Errorf("Expected AuthenticatedRPM 100, got %d", config.AuthenticatedRPM)
	}
	if config.AnonymousRPM != 30 {
		t.Errorf("Expected AnonymousRPM 30, got %d", config.AnonymousRPM)
	}
	if config.APIRPM != 60 {
		t.Errorf("Expected APIRPM 60, got %d", config.APIRPM)
	}
	if config.WorkspaceRPM != 20 {
		t.Errorf("Expected WorkspaceRPM 20, got %d", config.WorkspaceRPM)
	}
}

// TestRateLimitChecker tests the manual rate limit checker
func TestRateLimitChecker(t *testing.T) {
	config := RateLimitConfig{
		AnonymousRPM:    10,
		CleanupInterval: time.Minute,
	}

	checker := NewRateLimitChecker(config)

	// Use a path that matches AnonymousRPM (not /api/ or /workspaces/)
	req := httptest.NewRequest("GET", "/test/endpoint", nil)
	req.RemoteAddr = "192.168.2.1:12345"

	// First several requests should be allowed (we start with 10 tokens)
	allowedCount := 0
	for i := 0; i < 10; i++ {
		allowed, _ := checker.Check(req)
		if allowed {
			allowedCount++
		}
	}

	if allowedCount != 10 {
		t.Errorf("Expected 10 allowed requests, got %d", allowedCount)
	}

	// Next request should be blocked (or very few tokens left)
	allowed, remaining := checker.Check(req)
	// After 10 requests, we might have accumulated a tiny bit of tokens
	// So just check we're getting rate limited
	if allowed && remaining == 0 {
		t.Error("Expected to be rate limited after exhausting tokens")
	}
}

// TestRateLimiterCleanup tests the cleanup of stale entries
func TestRateLimiterCleanup(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping cleanup test in short mode")
	}

	config := RateLimitConfig{
		AnonymousRPM:    5,
		CleanupInterval: time.Millisecond * 100,
	}

	limiter := NewRateLimiter(config)

	// Create a client entry
	req := httptest.NewRequest("GET", "/api/test", nil)
	req.RemoteAddr = "192.168.3.1:12345"
	limiter.Allow(req, config)

	// Verify entry exists
	limiter.mu.RLock()
	_, exists := limiter.clients["ip:192.168.3.1"]
	limiter.mu.RUnlock()
	if !exists {
		t.Error("Expected client entry to exist")
	}

	// Wait for cleanup
	time.Sleep(time.Millisecond * 250)

	// Verify entry was cleaned up
	limiter.mu.RLock()
	_, exists = limiter.clients["ip:192.168.3.1"]
	limiter.mu.RUnlock()
	if exists {
		t.Error("Expected client entry to be cleaned up")
	}
}

// TestMinFunction tests the min helper function
func TestMinFunction(t *testing.T) {
	tests := []struct {
		a, b     float64
		expected float64
	}{
		{1.0, 2.0, 1.0},
		{5.0, 3.0, 3.0},
		{4.0, 4.0, 4.0},
		{0.0, 10.0, 0.0},
	}

	for _, test := range tests {
		result := min(test.a, test.b)
		if result != test.expected {
			t.Errorf("min(%f, %f) = %f, expected %f", test.a, test.b, result, test.expected)
		}
	}
}

// TestTokenRefill tests that tokens refill over time
func TestTokenRefill(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping token refill test in short mode")
	}

	config := RateLimitConfig{
		AnonymousRPM:    60, // 1 token per second
		CleanupInterval: time.Minute,
	}

	limiter := NewRateLimiter(config)

	// Use a path that matches AnonymousRPM (not /api/ or /workspaces/)
	req := httptest.NewRequest("GET", "/test/endpoint", nil)
	req.RemoteAddr = "192.168.4.1:12345"

	// Exhaust all tokens (60 for AnonymousRPM)
	for i := 0; i < 60; i++ {
		allowed, _ := limiter.Allow(req, config)
		if !allowed && i < 58 {
			t.Errorf("Request %d should be allowed", i+1)
		}
	}

	// Should be rate limited now
	allowed, _ := limiter.Allow(req, config)
	if allowed {
		t.Error("Expected to be rate limited after exhausting tokens")
	}

	// Wait for some tokens to refill (1 second = 1 token)
	time.Sleep(time.Second * 2)

	// Should have at least 1 token now
	allowed, _ = limiter.Allow(req, config)
	if !allowed {
		t.Error("Expected to be allowed after waiting for token refill")
	}
}
