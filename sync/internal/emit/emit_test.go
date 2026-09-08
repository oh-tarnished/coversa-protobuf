// Copyright 2026 The Protobuf Project authors.
// SPDX-License-Identifier: Apache-2.0

package emit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/the-protobuf-project/vdm/sync/internal/model"
	"github.com/the-protobuf-project/vdm/sync/internal/sdl"
	"github.com/the-protobuf-project/vdm/sync/internal/spec"
)

const fixture = `
type Vehicle @vspec(element: BRANCH, fqn: "Vehicle") {
  """Vehicle speed."""
  speed(unit: VelocityUnitEnum = KILOMETER_PER_HOUR): Float @vspec(element: SENSOR, fqn: "Vehicle.Speed")

  """Power optimization level."""
  powerOptimizeLevel: UInt8 @vspec(element: ACTUATOR, fqn: "Vehicle.PowerOptimizeLevel")
    @range(min: 0, max: 10)

  seats: [Seat] @vspec(element: BRANCH, fqn: "Vehicle.Cabin.Seat")
}

type Seat @vspec(element: BRANCH, fqn: "Vehicle.Cabin.Seat") {
  """Seat height."""
  height: UInt16 @vspec(element: ACTUATOR, fqn: "Vehicle.Cabin.Seat.Height")
}

enum VelocityUnitEnum @vspec(element: QUANTITY_KIND) {
  KILOMETER_PER_HOUR @vspec(element: UNIT)
}

type Person { id: ID! }
type ChargingStation { location: String }
type ChargingSession { vehicle: Vehicle! }
`

// build renders the fixture into a temporary directory and returns it.
func build(t *testing.T) string {
	t.Helper()

	defs, err := sdl.Parse("test.graphql", fixture)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	m, err := model.Build(defs)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	m.Spec = spec.Spec{Version: "2026-07-24", Commit: "36bc93936eab", Path: "vdm/spec"}

	dir := t.TempDir()
	if _, err := New(m).Generate(defs, dir); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	return dir
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

// TestSignalAnnotation checks the fact protobuf cannot state on its own: a
// double is kilometres per hour, and it is a sensor rather than an actuator.
func TestSignalAnnotation(t *testing.T) {
	got := read(t, build(t), "vss/vehicle/v1/vehicle.proto")

	for _, want := range []string{
		"double speed =",
		"(google.api.field_behavior) = OUTPUT_ONLY", // a sensor is not writable
		"element: ELEMENT_SENSOR",
		`fqn: "Vehicle.Speed"`,
		"unit: UNIT_KILOMETER_PER_HOUR",
		"quantity_kind: QUANTITY_KIND_VELOCITY",
		"Unit: km/h. VSS: Vehicle.Speed (sensor).", // restated for a human
	} {
		if !strings.Contains(got, want) {
			t.Errorf("vehicle.proto is missing %q", want)
		}
	}
}

// TestBoundsAreIntersected checks that a width bound and an explicit @range
// become one rule. Protobuf accepts a single rule per field, and emitting two
// is a compile error.
func TestBoundsAreIntersected(t *testing.T) {
	got := read(t, build(t), "vss/vehicle/v1/vehicle.proto")

	// UInt8 gives 0..255; @range(min: 0, max: 10) is tighter and wins.
	if !strings.Contains(got, "lte: 10") {
		t.Error("the @range upper bound did not win over the width bound")
	}
	if strings.Contains(got, "lte: 255") {
		t.Error("both bounds were emitted; they must be intersected")
	}
}

// TestReservedRange checks the held field numbers are reserved rather than
// merely skipped. A serialization target numbers slots contiguously, so an
// unreserved hole silently shifts every field after it.
func TestReservedRange(t *testing.T) {
	got := read(t, build(t), "vss/vehicle/v1/vehicle.proto")
	if !strings.Contains(got, "reserved 8 to 15;") {
		t.Error("vehicle.proto does not reserve the held field numbers")
	}
}

// TestSingletonHasNoCreate checks AIP-156: a resource that exists because its
// parent does cannot be created or deleted independently.
func TestSingletonHasNoCreate(t *testing.T) {
	dir := build(t)
	// Seat is a collection, so it has the full set. It lands in `platform`
	// here because this fixture hangs it directly off Vehicle and the domain
	// table does not name it -- which is the documented default, and is what
	// makes a new branch visible rather than silently misfiled.
	seat := read(t, dir, "vss/platform/seat/v1/service.proto")
	for _, want := range []string{"rpc GetSeat", "rpc ListSeats", "rpc CreateSeat",
		"rpc UpdateSeat", "rpc DeleteSeat", "rpc UndeleteSeat"} {
		if !strings.Contains(seat, want) {
			t.Errorf("Seats service is missing %q", want)
		}
	}

	if strings.Count(seat, "rpc ") != 6 {
		t.Errorf("Seats has %d RPCs, want 6", strings.Count(seat, "rpc "))
	}
}

// TestBannerNamesTheRevision checks every file carries the pin, which is what
// makes "which specification is this from?" answerable from one file.
func TestBannerNamesTheRevision(t *testing.T) {
	got := read(t, build(t), "vss/vehicle/v1/vehicle.proto")
	if !strings.Contains(got, "spec revision 2026-07-24 (36bc939). DO NOT EDIT.") {
		t.Error("the generated file does not name its specification revision")
	}
}

// TestCrossPackageReference checks AIP-215: a field naming a resource in
// another package becomes the resource name, not the message.
func TestCrossPackageReference(t *testing.T) {
	got := read(t, build(t), "vdm/charging_session/v1/charging_session.proto")

	for _, want := range []string{
		"string vehicle =",
		`(google.api.resource_reference) = {type: "vdm.covesa.org/Vehicle"}`,
		"(google.api.field_behavior) = IMMUTABLE",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("charging_session.proto is missing %q", want)
		}
	}
}
