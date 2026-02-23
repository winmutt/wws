package handlers

import "net/http"

func HealthHandler(w http.ResponseWriter, r *http.Request) error {
	return WriteJSON(w, http.StatusOK, map[string]string{"status": "healthy"})
}
