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

// Global audit log handler instance (initialized in main.go)
var AuditLogHandlerInstance *AuditLogHandler

// GetAuditLogsHandler wrapper for audit log handler
func GetAuditLogsHandler(w http.ResponseWriter, r *http.Request) {
	if AuditLogHandlerInstance != nil {
		AuditLogHandlerInstance.GetAuditLogs(w, r)
	} else {
		http.Error(w, "Audit logging not configured", http.StatusServiceUnavailable)
	}
}

// GetAuditLogByIDHandler wrapper for audit log handler
func GetAuditLogByIDHandler(w http.ResponseWriter, r *http.Request) {
	if AuditLogHandlerInstance != nil {
		AuditLogHandlerInstance.GetAuditLogByID(w, r)
	} else {
		http.Error(w, "Audit logging not configured", http.StatusServiceUnavailable)
	}
}

// Global quota handler instance (initialized in main.go)
var QuotaHandlerInstance *QuotaHandler

// QuotaGetHandler wrapper for quota get handler
func QuotaGetHandler(w http.ResponseWriter, r *http.Request) {
	if QuotaHandlerInstance != nil {
		QuotaHandlerInstance.GetQuota(w, r)
	} else {
		http.Error(w, "Quota management not configured", http.StatusServiceUnavailable)
	}
}

// QuotaUpdateHandler wrapper for quota update handler
func QuotaUpdateHandler(w http.ResponseWriter, r *http.Request) {
	if QuotaHandlerInstance != nil {
		QuotaHandlerInstance.UpdateQuota(w, r)
	} else {
		http.Error(w, "Quota management not configured", http.StatusServiceUnavailable)
	}
}

// QuotaUsageHandler wrapper for quota usage handler
func QuotaUsageHandler(w http.ResponseWriter, r *http.Request) {
	if QuotaHandlerInstance != nil {
		QuotaHandlerInstance.GetUsage(w, r)
	} else {
		http.Error(w, "Quota management not configured", http.StatusServiceUnavailable)
	}
}

// QuotaUpdateUsageHandler wrapper for quota update usage handler
func QuotaUpdateUsageHandler(w http.ResponseWriter, r *http.Request) {
	if QuotaHandlerInstance != nil {
		QuotaHandlerInstance.UpdateUsage(w, r)
	} else {
		http.Error(w, "Quota management not configured", http.StatusServiceUnavailable)
	}
}

// QuotaCheckHandler wrapper for quota check handler
func QuotaCheckHandler(w http.ResponseWriter, r *http.Request) {
	if QuotaHandlerInstance != nil {
		QuotaHandlerInstance.CheckQuota(w, r)
	} else {
		http.Error(w, "Quota management not configured", http.StatusServiceUnavailable)
	}
}

// GetAuditLogSummaryHandler wrapper for audit log handler
func GetAuditLogSummaryHandler(w http.ResponseWriter, r *http.Request) {
	if AuditLogHandlerInstance != nil {
		AuditLogHandlerInstance.GetAuditLogSummary(w, r)
	} else {
		http.Error(w, "Audit logging not configured", http.StatusServiceUnavailable)
	}
}
