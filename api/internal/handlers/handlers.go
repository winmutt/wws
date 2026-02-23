package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

type Handler func(w http.ResponseWriter, r *http.Request) error

func WriteJSON(w http.ResponseWriter, status int, data interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		return fmt.Errorf("failed to encode JSON: %w", err)
	}
	return nil
}

func WriteError(w http.ResponseWriter, status int, err error) {
	log.Printf("Error: %v", err)
	http.Error(w, err.Error(), status)
}

func Adapter(h Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := h(w, r); err != nil {
			WriteError(w, http.StatusInternalServerError, err)
		}
	}
}
