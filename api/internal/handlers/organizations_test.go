package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateOrganizationHandler(t *testing.T) {
	reqBody := map[string]string{"name": "Test Org"}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/organizations?owner_id=1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	err := CreateOrganizationHandler(w, req)
	if err != nil {
		t.Fatalf("CreateOrganizationHandler returned error: %v", err)
	}

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status %d, got %d", http.StatusCreated, w.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["name"] != "Test Org" {
		t.Errorf("Expected name 'Test Org', got '%v'", response["name"])
	}

	if response["owner_id"] == nil {
		t.Error("Expected owner_id to be set")
	}
}

func TestCreateOrganizationHandlerMissingName(t *testing.T) {
	reqBody := map[string]string{}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/organizations?owner_id=1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	err := CreateOrganizationHandler(w, req)
	if err == nil {
		t.Error("Expected error for missing name")
	}
}

func TestCreateOrganizationHandlerInvalidOwnerID(t *testing.T) {
	reqBody := map[string]string{"name": "Test Org"}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/organizations?owner_id=invalid", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	err := CreateOrganizationHandler(w, req)
	if err == nil {
		t.Error("Expected error for invalid owner_id")
	}
}
