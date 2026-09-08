// Copyright 2026 The Protobuf Project authors.
// SPDX-License-Identifier: Apache-2.0

package model

import (
	"testing"

	"github.com/the-protobuf-project/vdm/sync/internal/sdl"
)

// parse is the fixture helper: a small specification exercising the shapes
// the partitioner decides between.
func parse(t *testing.T, src string) *Model {
	t.Helper()
	defs, err := sdl.Parse("test.graphql", src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	m, err := Build(defs)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	return m
}

const fixture = `
type Vehicle @vspec(element: BRANCH, fqn: "Vehicle") {
  speed: Float @vspec(element: SENSOR, fqn: "Vehicle.Speed")
  cabin: Cabin @vspec(element: BRANCH, fqn: "Vehicle.Cabin")
  controlUnits: [ControlUnit] @vspec(element: BRANCH, fqn: "Vehicle.ControlUnit")
}

type Cabin @vspec(element: BRANCH, fqn: "Vehicle.Cabin") {
  doorCount: UInt8 @vspec(element: ATTRIBUTE, fqn: "Vehicle.Cabin.DoorCount")
  seats: [Seat] @vspec(element: BRANCH, fqn: "Vehicle.Cabin.Seat")
  light: Light @vspec(element: BRANCH, fqn: "Vehicle.Cabin.Light")
}

type Seat @vspec(element: BRANCH, fqn: "Vehicle.Cabin.Seat") {
  height: UInt16 @vspec(element: ACTUATOR, fqn: "Vehicle.Cabin.Seat.Height")
}

type Light @vspec(element: BRANCH, fqn: "Vehicle.Cabin.Light") {
  isOn: Boolean @vspec(element: ACTUATOR, fqn: "Vehicle.Cabin.Light.IsOn")
}

type ControlUnit @vspec(element: BRANCH, fqn: "Vehicle.ControlUnit") {
  serial: String @vspec(element: ATTRIBUTE, fqn: "Vehicle.ControlUnit.Serial")
}

type Person { id: ID! }
type ChargingStation { location: String }
type ChargingSession { vehicle: Vehicle! }
`

// TestShapes checks the two resource shapes and the names that follow.
//
// A branch a vehicle holds once is a singleton; one it holds many of is a
// collection whose members are individually addressable.
func TestShapes(t *testing.T) {
	m := parse(t, fixture)

	tests := []struct {
		resource  string
		pattern   string
		singleton bool
	}{
		{"Vehicle", "vehicles/{vehicle}", false},
		{"Cabin", "vehicles/{vehicle}/cabin", true},
		// AIP-123 forbids two literals in a row, so a seat hangs off the
		// vehicle rather than beneath the cabin singleton.
		{"Seat", "vehicles/{vehicle}/seats/{seat}", false},
		{"ControlUnit", "vehicles/{vehicle}/controlUnits/{control_unit}", false},
	}
	for _, tt := range tests {
		pkg := m.PackageOf(tt.resource)
		if pkg == nil {
			t.Errorf("%s is not a resource", tt.resource)
			continue
		}
		if pkg.Pattern != tt.pattern {
			t.Errorf("%s pattern = %q, want %q", tt.resource, pkg.Pattern, tt.pattern)
		}
		if pkg.Singleton() != tt.singleton {
			t.Errorf("%s singleton = %v, want %v", tt.resource, pkg.Singleton(), tt.singleton)
		}
	}
}

// TestGroupingNodeStaysEmbedded checks that a singular branch below the top
// level is *not* promoted: it has one occurrence, so a segment naming it
// would identify nothing.
func TestGroupingNodeStaysEmbedded(t *testing.T) {
	m := parse(t, fixture)
	if pkg := m.PackageOf("Light"); pkg != nil {
		t.Errorf("Light became a package (%s); it should stay inside Cabin", pkg.Pattern)
	}

	cabin := m.PackageOf("Cabin")
	if cabin == nil {
		t.Fatal("Cabin is not a resource")
	}
	if !cabin.Holds("Light") {
		t.Error("Cabin does not hold Light")
	}
}

// TestDomains checks that a branch inherits its ancestor's functional area,
// and that an unlisted one lands in platform rather than being dropped.
func TestDomains(t *testing.T) {
	m := parse(t, fixture)
	tests := []struct{ resource, domain string }{
		{"Cabin", "interior"},
		{"Seat", "interior"}, // inherited from Cabin
		{"ControlUnit", "platform"},
	}
	for _, tt := range tests {
		pkg := m.PackageOf(tt.resource)
		if pkg == nil {
			t.Errorf("%s is not a resource", tt.resource)
			continue
		}
		if pkg.Domain != tt.domain {
			t.Errorf("%s domain = %q, want %q", tt.resource, pkg.Domain, tt.domain)
		}
	}
}

// TestPackagePaths checks the path, proto package and Java package agree, so
// buf's PACKAGE_DIRECTORY_MATCH holds without a lookup table.
func TestPackagePaths(t *testing.T) {
	m := parse(t, fixture)
	seat := m.PackageOf("Seat")
	if seat == nil {
		t.Fatal("Seat is not a resource")
	}
	if got, want := seat.ProtoPackage(), "protobuf.covesa.vss.interior.seat.v1"; got != want {
		t.Errorf("ProtoPackage = %q, want %q", got, want)
	}
	if got, want := seat.ImportPath("seat.proto"),
		"protobuf/covesa/vss/interior/seat/v1/seat.proto"; got != want {
		t.Errorf("ImportPath = %q, want %q", got, want)
	}
}

// TestServiceNameAvoidsCollision covers the mass-noun case: a plural equal to
// the message name would collide inside the package, which protobuf rejects.
func TestServiceNameAvoidsCollision(t *testing.T) {
	m := parse(t, fixture)
	seat := m.PackageOf("Seat")
	if got := seat.ServiceName(); got != "Seats" {
		t.Errorf("Seat service = %q, want Seats", got)
	}
}
