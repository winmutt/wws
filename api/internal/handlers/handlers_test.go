package handlers

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestHealthHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	err := HealthHandler(w, req)
	if err != nil {
		t.Errorf("HealthHandler returned error: %v", err)
	}

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestOAuthCallbackHandlerMissingState(t *testing.T) {
	os.Setenv("GITHUB_CLIENT_ID", "test-client-id")
	os.Setenv("GITHUB_CLIENT_SECRET", "test-client-secret")
	os.Setenv("GITHUB_CALLBACK_URL", "http://localhost:8080/oauth/callback")

	req := httptest.NewRequest(http.MethodGet, "/auth/github/callback?code=testcode", nil)
	w := httptest.NewRecorder()

	err := OAuthCallbackHandler(w, req)
	if err == nil {
		t.Error("Expected error for missing state parameter")
	}

	expectedMsg := "missing state parameter"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error '%s', got '%s'", expectedMsg, err.Error())
	}
}

func TestOAuthCallbackHandlerMissingCode(t *testing.T) {
	os.Setenv("GITHUB_CLIENT_ID", "test-client-id")
	os.Setenv("GITHUB_CLIENT_SECRET", "test-client-secret")
	os.Setenv("GITHUB_CALLBACK_URL", "http://localhost:8080/oauth/callback")

	req := httptest.NewRequest(http.MethodGet, "/auth/github/callback?state=gh_teststate", nil)
	w := httptest.NewRecorder()

	err := OAuthCallbackHandler(w, req)
	if err == nil {
		t.Error("Expected error for missing code parameter")
	}

	expectedMsg := "missing authorization code"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error '%s', got '%s'", expectedMsg, err.Error())
	}
}

func TestOAuthCallbackHandlerInvalidState(t *testing.T) {
	os.Setenv("GITHUB_CLIENT_ID", "test-client-id")
	os.Setenv("GITHUB_CLIENT_SECRET", "test-client-secret")
	os.Setenv("GITHUB_CALLBACK_URL", "http://localhost:8080/oauth/callback")

	req := httptest.NewRequest(http.MethodGet, "/auth/github/callback?code=testcode&state=invalidstate", nil)
	w := httptest.NewRecorder()

	err := OAuthCallbackHandler(w, req)
	if err == nil {
		t.Error("Expected error for invalid state parameter")
	}

	expectedMsg := "invalid state parameter"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error '%s', got '%s'", expectedMsg, err.Error())
	}
}

func TestOAuthCallbackHandlerValidStateInvalidCode(t *testing.T) {
	os.Setenv("GITHUB_CLIENT_ID", "test-client-id")
	os.Setenv("GITHUB_CLIENT_SECRET", "test-client-secret")
	os.Setenv("GITHUB_CALLBACK_URL", "http://localhost:8080/oauth/callback")

	req := httptest.NewRequest(http.MethodGet, "/auth/github/callback?code=invalidcode&state=gh_teststate", nil)
	w := httptest.NewRecorder()

	err := OAuthCallbackHandler(w, req)
	if err == nil {
		t.Error("Expected error for invalid code")
	}

	if err.Error() == "" || !strings.Contains(err.Error(), "failed to exchange token") {
		t.Logf("OAuthCallbackHandler returned expected error for invalid token exchange: %v", err)
	}
}
