package billing

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"wws/api/internal/db"
)

// BillingType represents the type of billing entry
type BillingType string

const (
	BillingTypeCompute     BillingType = "compute"
	BillingTypeStorage     BillingType = "storage"
	BillingTypeNetwork     BillingType = "network"
	BillingTypeSnapshot    BillingType = "snapshot"
	BillingTypeBackup      BillingType = "backup"
	BillingTypeProvision   BillingType = "provisioning"
	BillingTypeTermination BillingType = "termination"
)

// BillingRecord represents a billing record
type BillingRecord struct {
	ID             int            `db:"id" json:"id"`
	OrganizationID int            `db:"organization_id" json:"organization_id"`
	WorkspaceID    sql.NullInt64  `db:"workspace_id" json:"workspace_id"`
	BillingType    BillingType    `db:"billing_type" json:"billing_type"`
	Amount         float64        `db:"amount" json:"amount"`
	Currency       string         `db:"currency" json:"currency"`
	Description    sql.NullString `db:"description" json:"description"`
	PeriodStart    sql.NullTime   `db:"period_start" json:"period_start"`
	PeriodEnd      sql.NullTime   `db:"period_end" json:"period_end"`
	CreatedAt      time.Time      `db:"created_at" json:"created_at"`
}

// CreateBillingRecord creates a new billing record
func CreateBillingRecord(ctx context.Context, orgID int, workspaceID *int, billingType BillingType, amount float64, currency, description string, periodStart, periodEnd *time.Time) (*BillingRecord, error) {
	record := &BillingRecord{
		OrganizationID: orgID,
		BillingType:    billingType,
		Amount:         amount,
		Currency:       currency,
		CreatedAt:      time.Now(),
	}

	if workspaceID != nil {
		record.WorkspaceID = sql.NullInt64{Int64: int64(*workspaceID), Valid: true}
	}

	if description != "" {
		record.Description = sql.NullString{String: description, Valid: true}
	}

	if periodStart != nil {
		record.PeriodStart = sql.NullTime{Time: *periodStart, Valid: true}
	}

	if periodEnd != nil {
		record.PeriodEnd = sql.NullTime{Time: *periodEnd, Valid: true}
	}

	query := `
		INSERT INTO billing_records (organization_id, workspace_id, billing_type, amount, currency, description, period_start, period_end, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	result, err := db.DB.ExecContext(ctx, query,
		record.OrganizationID,
		record.WorkspaceID.Int64,
		record.BillingType,
		record.Amount,
		record.Currency,
		record.Description.String,
		record.PeriodStart.Time,
		record.PeriodEnd.Time,
		record.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create billing record: %w", err)
	}

	recordID, _ := result.LastInsertId()
	record.ID = int(recordID)

	return record, nil
}

// GetBillingRecords retrieves billing records for an organization
func GetBillingRecords(ctx context.Context, orgID int, workspaceID *int, billingType *BillingType, startDate, endDate *time.Time, limit, offset int) ([]BillingRecord, error) {
	if limit <= 0 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	query := `
		SELECT id, organization_id, workspace_id, billing_type, amount, currency, description, period_start, period_end, created_at
		FROM billing_records
		WHERE organization_id = ?
	`
	args := []interface{}{orgID}

	if workspaceID != nil {
		query += " AND workspace_id = ?"
		args = append(args, *workspaceID)
	}

	if billingType != nil {
		query += " AND billing_type = ?"
		args = append(args, *billingType)
	}

	if startDate != nil {
		query += " AND created_at >= ?"
		args = append(args, *startDate)
	}

	if endDate != nil {
		query += " AND created_at <= ?"
		args = append(args, *endDate)
	}

	query += " ORDER BY created_at DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := db.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query billing records: %w", err)
	}
	defer rows.Close()

	var records []BillingRecord
	for rows.Next() {
		var record BillingRecord
		if err := rows.Scan(&record.ID, &record.OrganizationID, &record.WorkspaceID,
			&record.BillingType, &record.Amount, &record.Currency,
			&record.Description, &record.PeriodStart, &record.PeriodEnd, &record.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan billing record: %w", err)
		}
		records = append(records, record)
	}

	return records, nil
}

// GetBillingSummary retrieves a summary of billing for an organization
func GetBillingSummary(ctx context.Context, orgID int, startDate, endDate *time.Time) (map[string]interface{}, error) {
	query := `
		SELECT 
			SUM(amount) as total_amount,
			COUNT(*) as total_records,
			billing_type,
			SUM(CASE WHEN billing_type = ? THEN amount ELSE 0 END) as compute_amount,
			SUM(CASE WHEN billing_type = ? THEN amount ELSE 0 END) as storage_amount,
			SUM(CASE WHEN billing_type = ? THEN amount ELSE 0 END) as network_amount
		FROM billing_records
		WHERE organization_id = ?
	`
	args := []interface{}{orgID}

	if startDate != nil {
		query += " AND created_at >= ?"
		args = append(args, *startDate)
	}

	if endDate != nil {
		query += " AND created_at <= ?"
		args = append(args, *endDate)
	}

	var totalAmount sql.NullFloat64
	var totalRecords sql.NullInt64
	var computeAmount, storageAmount, networkAmount sql.NullFloat64

	allArgs := []interface{}{BillingTypeCompute, BillingTypeStorage, BillingTypeNetwork}
	allArgs = append(allArgs, args...)

	err := db.DB.QueryRowContext(ctx, query, allArgs...).Scan(
		&totalAmount, &totalRecords, &totalAmount, &computeAmount, &storageAmount, &networkAmount)
	if err != nil {
		return nil, fmt.Errorf("failed to query billing summary: %w", err)
	}

	summary := map[string]interface{}{
		"total_amount":   totalAmount.Float64,
		"total_records":  totalRecords.Int64,
		"compute_amount": computeAmount.Float64,
		"storage_amount": storageAmount.Float64,
		"network_amount": networkAmount.Float64,
	}

	return summary, nil
}

// GetWorkspaceBilling retrieves billing records for a specific workspace
func GetWorkspaceBilling(ctx context.Context, workspaceID int, startDate, endDate *time.Time) ([]BillingRecord, error) {
	return GetBillingRecords(ctx, 0, &workspaceID, nil, startDate, endDate, 100, 0)
}

// CalculateComputeCost calculates compute costs based on usage
func CalculateComputeCost(hours float64, hourlyRate float64) float64 {
	return hours * hourlyRate
}

// CalculateStorageCost calculates storage costs based on size
func CalculateStorageCost(GB float64, monthlyRate float64, days int) float64 {
	return GB * monthlyRate * (float64(days) / 30.0)
}

// CalculateNetworkCost calculates network costs based on bandwidth
func CalculateNetworkCost(MB float64, perMBRate float64) float64 {
	return MB * perMBRate
}

// RecordWorkspaceUsage records usage and creates corresponding billing records
func RecordWorkspaceUsage(ctx context.Context, workspaceID int, orgID int, hours float64, storageGB float64, networkMB float64) error {
	// Default rates (can be configured)
	hourlyComputeRate := 0.05  // $0.05 per hour
	monthlyStorageRate := 0.10 // $0.10 per GB per month
	networkRate := 0.001       // $0.001 per MB

	// Calculate costs
	computeCost := CalculateComputeCost(hours, hourlyComputeRate)
	storageCost := CalculateStorageCost(storageGB, monthlyStorageRate, 1)
	networkCost := CalculateNetworkCost(networkMB, networkRate)

	now := time.Now()

	// Create billing records
	if computeCost > 0 {
		_, err := CreateBillingRecord(ctx, orgID, &workspaceID, BillingTypeCompute, computeCost, "USD",
			fmt.Sprintf("Compute cost for %.2f hours", hours), &now, nil)
		if err != nil {
			return fmt.Errorf("failed to record compute cost: %w", err)
		}
	}

	if storageCost > 0 {
		_, err := CreateBillingRecord(ctx, orgID, &workspaceID, BillingTypeStorage, storageCost, "USD",
			fmt.Sprintf("Storage cost for %.2f GB", storageGB), &now, nil)
		if err != nil {
			return fmt.Errorf("failed to record storage cost: %w", err)
		}
	}

	if networkCost > 0 {
		_, err := CreateBillingRecord(ctx, orgID, &workspaceID, BillingTypeNetwork, networkCost, "USD",
			fmt.Sprintf("Network cost for %.2f MB", networkMB), &now, nil)
		if err != nil {
			return fmt.Errorf("failed to record network cost: %w", err)
		}
	}

	return nil
}

// GetMonthlyBilling retrieves monthly billing summary
func GetMonthlyBilling(ctx context.Context, orgID int, year, month int) (map[string]interface{}, error) {
	startDate := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	endDate := startDate.AddDate(0, 1, 0)

	summary, err := GetBillingSummary(ctx, orgID, &startDate, &endDate)
	if err != nil {
		return nil, err
	}

	summary["period"] = fmt.Sprintf("%d-%02d", year, month)
	summary["start_date"] = startDate
	summary["end_date"] = endDate

	return summary, nil
}
