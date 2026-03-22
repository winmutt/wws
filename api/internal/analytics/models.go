package analytics

import "time"

// WorkspaceUsage represents usage metrics for a workspace
type WorkspaceUsage struct {
	ID            int       `db:"id" json:"id"`
	WorkspaceID   int       `db:"workspace_id" json:"workspace_id"`
	CPUUsage      float64   `db:"cpu_usage" json:"cpu_usage"`       // Percentage
	MemoryUsage   float64   `db:"memory_usage" json:"memory_usage"` // Percentage
	StorageUsedGB float64   `db:"storage_used_gb" json:"storage_used_gb"`
	NetworkInMB   float64   `db:"network_in_mb" json:"network_in_mb"`
	NetworkOutMB  float64   `db:"network_out_mb" json:"network_out_mb"`
	UptimeSeconds int64     `db:"uptime_seconds" json:"uptime_seconds"`
	Timestamp     time.Time `db:"timestamp" json:"timestamp"`
}

// OrganizationUsage represents aggregated usage for an organization
type OrganizationUsage struct {
	OrganizationID   int       `db:"organization_id" json:"organization_id"`
	TotalWorkspaces  int       `db:"total_workspaces" json:"total_workspaces"`
	ActiveWorkspaces int       `db:"active_workspaces" json:"active_workspaces"`
	TotalCPUUsage    float64   `db:"total_cpu_usage" json:"total_cpu_usage"`
	TotalMemoryUsage float64   `db:"total_memory_usage" json:"total_memory_usage"`
	TotalStorageGB   float64   `db:"total_storage_gb" json:"total_storage_gb"`
	TotalNetworkMB   float64   `db:"total_network_mb" json:"total_network_mb"`
	AverageUptime    float64   `db:"average_uptime" json:"average_uptime"`
	LastUpdated      time.Time `db:"last_updated" json:"last_updated"`
}

// UserWorkspaceActivity represents workspace activity for a user
type UserWorkspaceActivity struct {
	UserID           int       `db:"user_id" json:"user_id"`
	WorkspaceID      int       `db:"workspace_id" json:"workspace_id"`
	LoginCount       int       `db:"login_count" json:"login_count"`
	TotalSessionTime int64     `db:"total_session_time" json:"total_session_time"` // seconds
	LastLogin        time.Time `db:"last_login" json:"last_login"`
	FirstLogin       time.Time `db:"first_login" json:"first_login"`
}

// UsageTimeSeries represents time-series data for usage trends
type UsageTimeSeries struct {
	WorkspaceID int       `db:"workspace_id" json:"workspace_id"`
	Timestamp   time.Time `db:"timestamp" json:"timestamp"`
	MetricName  string    `db:"metric_name" json:"metric_name"` // "cpu", "memory", "storage", "network"
	MetricValue float64   `db:"metric_value" json:"metric_value"`
}

// WorkspaceStats represents detailed workspace statistics
type WorkspaceStats struct {
	WorkspaceID      int       `json:"workspace_id"`
	WorkspaceTag     string    `json:"workspace_tag"`
	WorkspaceName    string    `json:"workspace_name"`
	TotalUptimeHours float64   `json:"total_uptime_hours"`
	UptimeSeconds    int64     `json:"-"`
	AverageCPU       float64   `json:"average_cpu"`
	AverageMemory    float64   `json:"average_memory"`
	AverageStorage   float64   `json:"average_storage_gb"`
	TotalNetworkMB   float64   `json:"total_network_mb"`
	LastActiveAt     time.Time `json:"last_active_at"`
	CreatedAt        time.Time `json:"created_at"`
}

// AnalyticsSummary represents a summary of all analytics
type AnalyticsSummary struct {
	TotalWorkspaces      int                 `json:"total_workspaces"`
	ActiveWorkspaces     int                 `json:"active_workspaces"`
	TotalUsers           int                 `json:"total_users"`
	AverageWorkspaceAge  float64             `json:"average_workspace_age_days"`
	OrganizationUsage    []OrganizationUsage `json:"organization_usage"`
	TopWorkspacesByUsage []WorkspaceStats    `json:"top_workspaces_by_usage"`
	TrendData            []TimeSeriesTrend   `json:"trend_data"`
}

// TimeSeriesTrend represents trend data over time
type TimeSeriesTrend struct {
	MetricName string            `json:"metric_name"`
	TimeRange  string            `json:"time_range"` // "daily", "weekly", "monthly"
	DataPoints []TimeSeriesPoint `json:"data_points"`
}

// TimeSeriesPoint represents a single data point in a time series
type TimeSeriesPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
}

// UsageAlert represents a usage alert
type UsageAlert struct {
	ID             int        `db:"id" json:"id"`
	OrganizationID int        `db:"organization_id" json:"organization_id"`
	WorkspaceID    *int       `db:"workspace_id" json:"workspace_id,omitempty"`
	AlertType      string     `db:"alert_type" json:"alert_type"` // "cpu_high", "memory_high", "storage_full", "quota_exceeded"
	Severity       string     `db:"severity" json:"severity"`     // "low", "medium", "high", "critical"
	Message        string     `db:"message" json:"message"`
	Value          float64    `db:"value" json:"value"`
	Threshold      float64    `db:"threshold" json:"threshold"`
	Resolved       bool       `db:"resolved" json:"resolved"`
	CreatedAt      time.Time  `db:"created_at" json:"created_at"`
	ResolvedAt     *time.Time `db:"resolved_at" json:"resolved_at,omitempty"`
}
