package handlers

import "net/http"

type Handler func(w http.ResponseWriter, r *http.Request) error

func WriteJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
