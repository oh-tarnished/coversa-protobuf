// Copyright 2026 The Protobuf Project authors.
// SPDX-License-Identifier: Apache-2.0

package vss_test

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/the-protobuf-project/vdm/codec/vss"
)

// load reads the committed manifest, which is what a third party would.
func load(t *testing.T) *vss.Manifest {
	t.Helper()
	body, err := os.ReadFile("../manifest.json")
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	var m vss.Manifest
	if err := json.Unmarshal(body, &m); err != nil {
		t.Fatalf("decode manifest: %v", err)
	}
	return &m
}

const seat = "protobuf.covesa.vss.interior.seat.v1.Seat"

// TestToVSSUsesFullyQualifiedNames checks the conversion produces the names a
// VSS-native system addresses signals by, not the protobuf field names.
func TestToVSSUsesFullyQualifiedNames(t *testing.T) {
	got, err := load(t).ToVSS(seat, []byte(`{"height":420,"isBelted":true}`))
	if err != nil {
		t.Fatalf("ToVSS: %v", err)
	}
	if got["Vehicle.Cabin.Seat.Height"] != float64(420) {
		t.Errorf("Height = %v, want 420", got["Vehicle.Cabin.Seat.Height"])
	}
	if got["Vehicle.Cabin.Seat.IsBelted"] != true {
		t.Errorf("IsBelted = %v, want true", got["Vehicle.Cabin.Seat.IsBelted"])
	}
}

// TestAcceptsBothJSONSpellings checks that protobuf JSON's lowerCamelCase and
// the declared snake_case both resolve, since both reach a decoder in
// practice.
func TestAcceptsBothJSONSpellings(t *testing.T) {
	m := load(t)
	for _, body := range []string{`{"isBelted":true}`, `{"is_belted":true}`} {
		got, err := m.ToVSS(seat, []byte(body))
		if err != nil {
			t.Fatalf("ToVSS(%s): %v", body, err)
		}
		if got["Vehicle.Cabin.Seat.IsBelted"] != true {
			t.Errorf("ToVSS(%s) did not resolve the field", body)
		}
	}
}

// TestRoundTrip is what the manifest exists to make possible: out to the VSS
// shape and back, unchanged.
func TestRoundTrip(t *testing.T) {
	m := load(t)
	const original = `{"height":420,"isBelted":true,"heatingCooling":-40}`

	tree, err := m.ToVSS(seat, []byte(original))
	if err != nil {
		t.Fatalf("ToVSS: %v", err)
	}
	back, err := m.FromVSS(seat, tree)
	if err != nil {
		t.Fatalf("FromVSS: %v", err)
	}

	var want, got map[string]any
	if err := json.Unmarshal([]byte(original), &want); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(back, &got); err != nil {
		t.Fatal(err)
	}
	// The round trip returns declared names; compare on those.
	for _, k := range []string{"height", "is_belted", "heating_cooling"} {
		if _, ok := got[k]; !ok {
			t.Errorf("round trip lost %q: got %v", k, got)
		}
	}
	if got["height"] != want["height"] {
		t.Errorf("height = %v, want %v", got["height"], want["height"])
	}
}

// TestEnumValuesUseSourceSpelling is the half of the mapping a consumer
// cannot derive: VSS writes "ON" where the schema emits STATE_ON_VALUE, and a
// peer expecting the former cannot read the latter.
//
// The suffix is not decoration. Cap'n Proto strips an enum's name prefix off
// its values, which leaves `on` -- one of its keywords -- so the constant is
// spelled to survive that and the manifest is the only record of what it was.
func TestEnumValuesUseSourceSpelling(t *testing.T) {
	const massage = "protobuf.covesa.vss.interior.seat.v1.Massage"

	got, err := load(t).ToVSS(massage, []byte(`{"activation":"STATE_ON_VALUE"}`))
	if err != nil {
		t.Fatalf("ToVSS: %v", err)
	}
	if got["Vehicle.Cabin.Seat.Massage.Status"] != "ON" {
		t.Errorf("Status = %v, want ON", got["Vehicle.Cabin.Seat.Massage.Status"])
	}
}

// TestNestedMessagesAreRecursed checks the translation reaches past the first
// boundary. A converter that stops at an embedded message hands back the
// protobuf spelling of everything inside it, which is not the VSS shape.
func TestNestedMessagesAreRecursed(t *testing.T) {
	got, err := load(t).ToVSS(seat, []byte(`{"massage":{"activation":"STATE_OFF","level":50}}`))
	if err != nil {
		t.Fatalf("ToVSS: %v", err)
	}
	nested, ok := got["Vehicle.Cabin.Seat.Massage"].(map[string]any)
	if !ok {
		t.Fatalf("Massage = %#v, want a nested map", got["Vehicle.Cabin.Seat.Massage"])
	}
	if nested["activation"] != "OFF" {
		t.Errorf("nested activation = %v, want OFF", nested["activation"])
	}
	if nested["level"] != float64(50) {
		t.Errorf("nested level = %v, want 50", nested["level"])
	}
}

// TestInstanceAxesAreRecorded is what replaced the instance-tag message: a
// seat is addressed as `vehicles/{v}/seats/row1DriverSide`, and nothing in
// the protobuf says that id is two VSS path segments joined.
func TestInstanceAxesAreRecorded(t *testing.T) {
	r, ok := load(t).Resources[seat]
	if !ok {
		t.Fatal("no mapping for the seat")
	}
	if len(r.Instances) != 2 {
		t.Fatalf("Instances = %v, want two axes", r.Instances)
	}
	if r.Instances[0][0] != "Row1" || r.Instances[1][0] != "DriverSide" {
		t.Errorf("Instances = %v, want rows then sides", r.Instances)
	}
	if r.Pattern != "vehicles/{vehicle}/seats/{seat}" {
		t.Errorf("Pattern = %q", r.Pattern)
	}
}

// TestUnknownMessageIsAnError checks the failure a revision mismatch produces
// is reported rather than silently returning nothing.
func TestUnknownMessageIsAnError(t *testing.T) {
	if _, err := load(t).ToVSS("does.not.Exist", []byte(`{}`)); err == nil {
		t.Fatal("ToVSS accepted an unknown message; want an error")
	}
}

// TestManifestRecordsRenames checks the renames are present, since a consumer
// mapping back to VSS vocabulary has no other way to learn them.
//
// The source spelling is the VSS path segment, PascalCase: the specification
// writes `Vehicle.VehicleIdentification.ProductionDate`, and a peer joining
// on that name needs the segment as it appears in the path.
func TestManifestRecordsRenames(t *testing.T) {
	m := load(t)
	const vid = "protobuf.covesa.vss.platform.vehicle_identification.v1.VehicleIdentification"

	f, ok := m.Lookup(vid, "production")
	if !ok {
		t.Fatal("no mapping for production")
	}
	if f.Source != "ProductionDate" {
		t.Errorf("production source = %q, want ProductionDate", f.Source)
	}
	if f.FQN != "Vehicle.VehicleIdentification.ProductionDate" {
		t.Errorf("production fqn = %q", f.FQN)
	}
}

// TestManifestNamesBothRevisions checks a consumer can tell which
// specifications the mapping describes. A manifest and a message from
// different revisions disagree silently otherwise: a signal added upstream is
// simply absent here, and the conversion drops it without a word.
func TestManifestNamesBothRevisions(t *testing.T) {
	s := load(t).Spec
	if s.VSS.Version == "" || s.VSS.Commit == "" {
		t.Errorf("vss revision = %+v", s.VSS)
	}
	if s.VDM.Version == "" || s.VDM.Commit == "" {
		t.Errorf("vdm revision = %+v", s.VDM)
	}
}
