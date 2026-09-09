// Copyright 2026 Srikanth Kandarpa.
// SPDX-License-Identifier: Apache-2.0

package emit

// emit_test.go checks the facts the emitted protobuf carries that protobuf
// itself cannot state: what a number means, whether it can be written, and
// which specification revision it came from.

import (
	"strings"
	"testing"
)

// TestSignalAnnotation checks the facts protobuf cannot state on its own: a
// double is kilometres per hour, and it is a sensor rather than an actuator.
func TestSignalAnnotation(t *testing.T) {
	got := read(t, build(t), "vss/vehicle/v1/vehicle.proto")
	for _, want := range []string{
		"double speed =",
		"(google.api.field_behavior) = OUTPUT_ONLY", // a sensor is not writable
		"element: ELEMENT_SENSOR",
		`fqn: "Vehicle.Speed"`,
		"unit: UNIT_KM_PER_H",
		"quantity_kind: QUANTITY_KIND_VELOCITY",
		"Unit: km/h. VSS: Vehicle.Speed (sensor).", // restated for a human
	} {
		if !strings.Contains(got, want) {
			t.Errorf("vehicle.proto is missing %q", want)
		}
	}
}

// TestCommentReachesTheComment is what reading .vspec directly bought: the
// GraphQL translation dropped every one of these.
func TestCommentReachesTheComment(t *testing.T) {
	got := read(t, build(t), "vss/vehicle/v1/vehicle.proto")
	if !strings.Contains(got, "Reported by the vehicle; no API call sets it.") {
		t.Error("the specification's comment did not reach the generated file")
	}
}

// TestBoundsAreIntersected checks a width bound and a declared range become
// one rule. Protobuf accepts a single rule per field, and emitting two is a
// compile error.
func TestBoundsAreIntersected(t *testing.T) {
	got := read(t, build(t), "vss/vehicle/v1/vehicle.proto")
	if !strings.Contains(got, "lte: 10") {
		t.Error("the declared upper bound did not win over the width bound")
	}
	if strings.Contains(got, "lte: 255") {
		t.Error("both bounds were emitted; they must be intersected")
	}
}

// TestReservedRange checks the held field numbers are reserved rather than
// merely skipped. A serialization target numbers slots contiguously, so an
// unreserved hole silently shifts every field after it.
func TestReservedRange(t *testing.T) {
	if got := read(t, build(t), "vss/vehicle/v1/vehicle.proto"); !strings.Contains(got, "reserved 8 to 15;") {
		t.Error("vehicle.proto does not reserve the held field numbers")
	}
}

// TestAllowedBecomesEnum checks an allowed-value set becomes a protobuf enum
// with the zero value AIP-126 requires.
func TestAllowedBecomesEnum(t *testing.T) {
	got := read(t, build(t), "vss/interior/seat/v1/seat.proto")
	for _, want := range []string{
		"enum OccupancyState{", "enum OccupancyState {",
	} {
		if strings.Contains(got, want) {
			goto found
		}
	}
	t.Errorf("no OccupancyState enum:\n%s", got[:min(600, len(got))])
	return
found:
	for _, want := range []string{"OCCUPANCY_STATE_UNSPECIFIED = 0;", "OCCUPANCY_STATE_OCCUPIED"} {
		if !strings.Contains(got, want) {
			t.Errorf("enum is missing %q", want)
		}
	}
}

// TestInstancedBranchIsItsOwnResource checks a branch VSS gives instances to
// becomes separately addressable — the whole point of promoting them.
func TestInstancedBranchIsItsOwnResource(t *testing.T) {
	got := read(t, build(t), "vss/interior/seat/v1/seat.proto")
	if !strings.Contains(got, `pattern: "vehicles/{vehicle}/seats/{seat}"`) {
		t.Error("Seat is not addressable on its own")
	}
}

// TestBannerNamesBothRevisions checks every file says where it came from,
// which is what makes that answerable from one file.
func TestBannerNamesBothRevisions(t *testing.T) {
	got := read(t, build(t), "vss/vehicle/v1/vehicle.proto")
	for _, want := range []string{"revision 2026-09-02 (cd4bc50)", "revision 2026-07-24 (36bc939)"} {
		if !strings.Contains(got, want) {
			t.Errorf("banner is missing %q", want)
		}
	}
}
