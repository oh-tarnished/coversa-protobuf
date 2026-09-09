// Copyright 2026 The Protobuf Project authors.
// SPDX-License-Identifier: Apache-2.0

package vspec

import (
	"os"
	"path/filepath"
	"testing"
)

// write lays out a small specification in a temporary directory.
func write(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, body := range files {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// TestParseTree covers the shapes the catalogue is built from: a branch, the
// three signal kinds, units and bounds, and prose.
func TestParseTree(t *testing.T) {
	dir := write(t, map[string]string{"root.vspec": `
Vehicle:
  type: branch
  description: High-level vehicle data.
  Speed:
    datatype: float
    type: sensor
    unit: km/h
    description: Vehicle speed.
    comment: Reported by the vehicle, never set.
  Height:
    datatype: uint16
    type: attribute
    unit: mm
    min: 0
    max: 5000
    description: Overall height.
`})
	root, err := Parse(filepath.Join(dir, "root.vspec"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	if root.FQN != "Vehicle" || root.Kind != KindBranch {
		t.Fatalf("root = %s %s", root.FQN, root.Kind)
	}
	speed, ok := root.Child("Speed")
	if !ok {
		t.Fatal("no Speed")
	}
	if speed.FQN != "Vehicle.Speed" || speed.Kind != KindSensor {
		t.Errorf("speed = %s %s", speed.FQN, speed.Kind)
	}
	if speed.Unit != "km/h" || speed.Datatype != "float" {
		t.Errorf("speed unit/datatype = %q %q", speed.Unit, speed.Datatype)
	}
	// The comment is the whole reason for reading .vspec rather than the
	// GraphQL translation, which drops every one.
	if speed.Comment != "Reported by the vehicle, never set." {
		t.Errorf("comment = %q", speed.Comment)
	}
	if h, _ := root.Child("Height"); h.Min != "0" || h.Max != "5000" {
		t.Errorf("bounds = %q %q", h.Min, h.Max)
	}
}

// TestKindSignal checks which kinds carry a value, since that is what decides
// writability downstream.
func TestKindSignal(t *testing.T) {
	for kind, want := range map[Kind]bool{
		KindSensor: true, KindActuator: true, KindAttribute: true,
		KindBranch: false, KindStruct: false,
	} {
		if kind.Signal() != want {
			t.Errorf("%s.Signal() = %v, want %v", kind, kind.Signal(), want)
		}
	}
}
