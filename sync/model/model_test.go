// Copyright 2026 The Protobuf Project authors.
// SPDX-License-Identifier: Apache-2.0

package model

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/oh-tarnished/coversa-protobuf/sync/vspec"
)

// fixture is a vehicle small enough to reason about, exercising the shapes
// the partitioner decides between: a singleton branch, an instanced branch, a
// grouping node with one occurrence, and a branch nested two deep.
const fixture = `
Vehicle:
  type: branch
  description: High-level vehicle data.
  Speed:
    datatype: float
    type: sensor
    unit: km/h
    description: Vehicle speed.
  Cabin:
    type: branch
    description: The cabin.
    Seat:
      type: branch
      description: All seats.
      instances:
        - Row[1,2]
        - ["DriverSide", "PassengerSide"]
      Height:
        datatype: uint16
        type: actuator
        unit: mm
        description: Seat height.
    Light:
      type: branch
      description: Cabin lighting.
      AmbientLight:
        type: branch
        description: Ambient lights.
        instances: ["Front", "Rear"]
        IsOn:
          datatype: boolean
          type: actuator
          description: Is it lit.
  ControlUnit:
    type: branch
    description: Control units.
    instances: ["Central", "FrontLeft"]
    Serial:
      datatype: string
      type: attribute
      description: Hardware serial.
`

// catalogue is the smallest unit and quantity pair the fixture needs. Build
// resolves every unit a signal names, so a signal carrying one the catalogue
// does not define is an error rather than an unannotated field.
const catalogue = `
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

func build(t *testing.T) *Model {
	t.Helper()
	dir := t.TempDir()
	for name, body := range map[string]string{
		"root.vspec": fixture, "units.yaml": catalogue, "quantities.yaml": quantities,
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	root, err := vspec.Parse(filepath.Join(dir, "root.vspec"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	units, err := vspec.LoadCatalogue(dir)
	if err != nil {
		t.Fatalf("LoadCatalogue: %v", err)
	}
	m, err := Build(root, nil, units)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	return m
}

// TestShapes checks the two resource shapes and the names that follow.
func TestShapes(t *testing.T) {
	m := build(t)
	tests := []struct {
		fqn       string
		pattern   string
		singleton bool
	}{
		{"Vehicle", "vehicles/{vehicle}", false},
		{"Vehicle.Cabin", "vehicles/{vehicle}/cabin", true},
		// AIP-123 forbids two literals in a row, so a seat hangs off the
		// vehicle rather than beneath the cabin singleton.
		{"Vehicle.Cabin.Seat", "vehicles/{vehicle}/seats/{seat}", false},
		{"Vehicle.ControlUnit", "vehicles/{vehicle}/controlUnits/{control_unit}", false},
	}
	for _, tt := range tests {
		pkg := m.PackageOf(tt.fqn)
		if pkg == nil {
			t.Errorf("%s is not a resource", tt.fqn)
			continue
		}
		if pkg.Pattern != tt.pattern {
			t.Errorf("%s pattern = %q, want %q", tt.fqn, pkg.Pattern, tt.pattern)
		}
		if pkg.Singleton() != tt.singleton {
			t.Errorf("%s singleton = %v, want %v", tt.fqn, pkg.Singleton(), tt.singleton)
		}
	}
}

// TestGroupingNodeStaysEmbedded checks a singular branch below the top level
// is not promoted: it has one occurrence, so a segment naming it would
// identify nothing.
func TestGroupingNodeStaysEmbedded(t *testing.T) {
	m := build(t)
	if pkg := m.PackageOf("Vehicle.Cabin.Light"); pkg != nil {
		t.Errorf("Light became a package (%s); it should stay inside Cabin", pkg.Pattern)
	}
	if cabin := m.PackageOf("Vehicle.Cabin"); cabin == nil || !cabin.Holds("Vehicle.Cabin.Light") {
		t.Error("Cabin does not hold Light")
	}
}

// TestWalksThroughGroupingNodes checks an instanced branch beneath a grouping
// node is still promoted, and hangs off the nearest identifiable ancestor.
func TestWalksThroughGroupingNodes(t *testing.T) {
	m := build(t)
	pkg := m.PackageOf("Vehicle.Cabin.Light.AmbientLight")
	if pkg == nil {
		t.Fatal("AmbientLight was not promoted")
	}
	if want := "vehicles/{vehicle}/ambientLights/{ambient_light}"; pkg.Pattern != want {
		t.Errorf("pattern = %q, want %q", pkg.Pattern, want)
	}
}

// TestDomains checks a branch inherits its ancestor's functional area, and
// that an unlisted one lands in platform rather than being dropped.
func TestDomains(t *testing.T) {
	m := build(t)
	for fqn, want := range map[string]string{
		"Vehicle.Cabin":       "interior",
		"Vehicle.Cabin.Seat":  "interior", // inherited
		"Vehicle.ControlUnit": "platform",
	} {
		pkg := m.PackageOf(fqn)
		if pkg == nil {
			t.Errorf("%s is not a resource", fqn)
			continue
		}
		if pkg.Domain != want {
			t.Errorf("%s domain = %q, want %q", fqn, pkg.Domain, want)
		}
	}
}

// TestPackagePaths checks path, proto package and import path agree, so buf's
// PACKAGE_DIRECTORY_MATCH holds without a lookup table.
func TestPackagePaths(t *testing.T) {
	seat := build(t).PackageOf("Vehicle.Cabin.Seat")
	if got, want := seat.ProtoPackage(), "protobuf.covesa.vss.interior.seat.v1"; got != want {
		t.Errorf("ProtoPackage = %q, want %q", got, want)
	}
	if got, want := seat.ImportPath("seat.proto"),
		"protobuf/covesa/vss/interior/seat/v1/seat.proto"; got != want {
		t.Errorf("ImportPath = %q, want %q", got, want)
	}
}
