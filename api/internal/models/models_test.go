// Package models_test verifies the Workspace struct includes a Region field.
package models

import (
	"reflect"
	"testing"
)

func TestWorkspaceHasRegionField(t *testing.T) {
	ws := Workspace{}
	typ := reflect.TypeOf(ws)
	if _, ok := typ.FieldByName("Region"); !ok {
		t.Fatalf("Workspace struct missing Region field")
	}
}
