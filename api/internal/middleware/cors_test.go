package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCORSMiddleware(t *testing.T) {
	handler := CORSMiddleware([]string{"http://localhost:3000", "https://example.com"})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))

	tests := []struct {
		name           string
		origin         string
		expectedHeader string
		expectCORS     bool
	}{
		{
			name:           "Allowed origin",
			origin:         "http://localhost:3000",
			expectedHeader: "http://localhost:3000",
			expectCORS:     true,
		},
		{
			name:           "Another allowed origin",
			origin:         "https://example.com",
			expectedHeader: "https://example.com",
			expectCORS:     true,
		},
		{
			name:       "Not allowed origin",
			origin:     "https://malicious.com",
			expectCORS: false,
		},
		{
			name:       "No origin",
			origin:     "",
			expectCORS: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			if tt.expectCORS {
				if rr.Header().Get("Access-Control-Allow-Origin") != tt.expectedHeader {
					t.Errorf("Expected Access-Control-Allow-Origin: %s, got: %s", tt.expectedHeader, rr.Header().Get("Access-Control-Allow-Origin"))
				}
				if rr.Header().Get("Access-Control-Allow-Credentials") != "true" {
					t.Errorf("Expected Access-Control-Allow-Credentials: true, got: %s", rr.Header().Get("Access-Control-Allow-Credentials"))
				}
			} else {
				if rr.Header().Get("Access-Control-Allow-Origin") != "" {
					t.Errorf("Expected no CORS header, got: %s", rr.Header().Get("Access-Control-Allow-Origin"))
				}
			}
		})
	}
}

func TestCORSOptions(t *testing.T) {
	handler := CORSMiddleware([]string{"http://localhost:3000"})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodOptions, "/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got: %d", rr.Code)
	}

	if rr.Header().Get("Access-Control-Allow-Methods") == "" {
		t.Error("Expected Access-Control-Allow-Methods header")
	}

	if rr.Header().Get("Access-Control-Allow-Headers") == "" {
		t.Error("Expected Access-Control-Allow-Headers header")
	}
}

func TestCORSMiddlewareWildcard(t *testing.T) {
	handler := CORSMiddleware([]string{"*"})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "https://any-origin.com")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Header().Get("Access-Control-Allow-Origin") != "https://any-origin.com" {
		t.Errorf("Expected Access-Control-Allow-Origin: https://any-origin.com, got: %s", rr.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestContentTypeMiddleware(t *testing.T) {
	handler := ContentTypeMiddleware("application/json")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	tests := []struct {
		name         string
		method       string
		contentType  string
		expectStatus int
	}{
		{
			name:         "Correct content type",
			method:       http.MethodPost,
			contentType:  "application/json",
			expectStatus: http.StatusOK,
		},
		{
			name:         "Wrong content type",
			method:       http.MethodPost,
			contentType:  "text/plain",
			expectStatus: http.StatusUnsupportedMediaType,
		},
		{
			name:         "Missing content type on GET",
			method:       http.MethodGet,
			contentType:  "",
			expectStatus: http.StatusOK,
		},
		{
			name:         "Missing content type on POST",
			method:       http.MethodPost,
			contentType:  "",
			expectStatus: http.StatusUnsupportedMediaType,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/test", nil)
			if tt.contentType != "" {
				req.Header.Set("Content-Type", tt.contentType)
			}

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			if rr.Code != tt.expectStatus {
				t.Errorf("Expected status %d, got: %d", tt.expectStatus, rr.Code)
			}
		})
	}
}

func TestLoggingMiddleware(t *testing.T) {
	handler := Logging(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got: %d", rr.Code)
	}
}

func TestRecoveryMiddleware(t *testing.T) {
	panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})

	handler := Recovery(panicHandler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got: %d", rr.Code)
	}

	if rr.Body.String() != "Internal Server Error\n" {
		t.Errorf("Expected 'Internal Server Error', got: %s", rr.Body.String())
	}
}
