package handlers

import "net/http"

func HealthHandler(w http.ResponseWriter, r *http.Request) error {
	WriteJSON(w, http.StatusOK, map[string]string{"status": "healthy"})
	return nil
}
