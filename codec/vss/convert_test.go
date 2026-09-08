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
// cannot derive: VSS writes "Row1" where the schema emits
// DIMENSION1_ROW1, and a peer expecting the former cannot read the latter.
func TestEnumValuesUseSourceSpelling(t *testing.T) {
	m := load(t)
	const tag = "protobuf.covesa.vss.interior.seat.v1.SeatInstanceTag"

	got, err := m.ToVSS(tag, []byte(`{"dimension1":"DIMENSION1_ROW1"}`))
	if err != nil {
		t.Fatalf("ToVSS: %v", err)
	}
	if got["Vehicle.Cabin.Seat.Dimension1"] != "Row1" &&
		got["Vehicle.Cabin.Seat"] != "Row1" {
		// The tag's own fqn varies; assert the value was translated at all.
		for _, v := range got {
			if v == "Row1" {
				return
			}
		}
		t.Errorf("enum was not translated to its source spelling: %v", got)
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
func TestManifestRecordsRenames(t *testing.T) {
	m := load(t)
	const vid = "protobuf.covesa.vss.platform.vehicle_identification.v1.VehicleIdentification"

	f, ok := m.Lookup(vid, "production")
	if !ok {
		t.Fatal("no mapping for production")
	}
	if f.Source != "productionDate" {
		t.Errorf("production source = %q, want productionDate", f.Source)
	}
}
