// Copyright 2026 The Protobuf Project authors.
// SPDX-License-Identifier: Apache-2.0

package emit

// fixture_test.go holds the specification the emitter tests render, and the
// two helpers that turn it into files on disk. Kept apart from the assertions
// so a test reads as what it checks rather than as how the input was built.

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/the-protobuf-project/vdm/sync/model"
	"github.com/the-protobuf-project/vdm/sync/spec"
	"github.com/the-protobuf-project/vdm/sync/vspec"
)

const fixture = `
Vehicle:
  type: branch
  description: High-level vehicle data.
  Speed:
    datatype: float
    type: sensor
    unit: km/h
    description: Vehicle speed.
    comment: Reported by the vehicle; no API call sets it.
  PowerOptimizeLevel:
    datatype: uint8
    type: actuator
    min: 0
    max: 10
    description: Power optimization level.
  Cabin:
    type: branch
    description: The cabin.
    Seat:
      type: branch
      description: All seats.
      instances: ["Row1", "Row2"]
      Height:
        datatype: uint16
        type: actuator
        unit: mm
        description: Seat height.
      OccupancyStatus:
        datatype: string
        type: sensor
        allowed: ['UNKNOWN', 'OCCUPIED', 'EMPTY']
        description: Occupancy of the seat.
`

// units is the smallest catalogue the fixture needs.
const units = `
km/h:
  definition: Velocity measured in kilometers per hour
  unit: kilometer per hour
  quantity: velocity
  allowed-datatypes: ['numeric']
mm:
  definition: Length measured in millimeters
  unit: millimeter
  quantity: length
  allowed-datatypes: ['numeric']
`

const quantities = `
velocity:
  definition: Rate of change of a position vector
length:
  definition: Linear extent in space between any two points
`

// build renders the fixture into a temporary directory and returns it.
func build(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for name, body := range map[string]string{
		"root.vspec": fixture, "units.yaml": units, "quantities.yaml": quantities,
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	root, err := vspec.Parse(filepath.Join(dir, "root.vspec"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	catalogue, err := vspec.LoadCatalogue(dir)
	if err != nil {
		t.Fatalf("LoadCatalogue: %v", err)
	}
	m, err := model.Build(root, nil, catalogue)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	m.Spec = spec.Spec{
		VSS: spec.Source{Version: "2026-09-02", Commit: "cd4bc50ba4aa"},
		VDM: spec.Source{Version: "2026-07-24", Commit: "36bc93936eab"},
	}

	out := filepath.Join(dir, "out")
	if _, err := New(m).Generate(out); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	return out
}

// read returns one generated file.
func read(t *testing.T, dir, path string) string {
	t.Helper()
	body, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(path)))
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(body)
}
