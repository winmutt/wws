// Package db_test verifies the database schema includes the region column.
package db

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	"testing"
)

func TestWorkspaceTableHasRegionColumn(t *testing.T) {
	// Use in‑memory SQLite DB for testing.
	Init(":memory:")
	defer Close()

	rows, err := DB.Query("PRAGMA table_info(workspaces)")
	if err != nil {
		t.Fatalf("failed to query table info: %v", err)
	}
	defer rows.Close()

	found := false
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dfltValue sql.NullString
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dfltValue, &pk); err != nil {
			t.Fatalf("scan error: %v", err)
		}
		if name == "region" {
			found = true
			break
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("row iteration error: %v", err)
	}
	if !found {
		t.Fatalf("region column not found in workspaces table")
	}
}
