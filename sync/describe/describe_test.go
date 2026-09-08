// Copyright 2026 The Protobuf Project authors.
// SPDX-License-Identifier: Apache-2.0

package describe

import (
	"strings"
	"testing"

	"github.com/the-protobuf-project/vdm/sync/plan"
	"github.com/the-protobuf-project/vdm/sync/sdl"
)

// TestEveryFieldIsDescribed is the rule this package exists to enforce:
// nothing is ever left blank, whichever format is being written.
//
// The source model documents most signals and none of the instance-tag
// machinery, and api-linter requires a comment on every field regardless.
func TestEveryFieldIsDescribed(t *testing.T) {
	tests := []struct{ name, doc, want string }{
		{"speed", "Vehicle speed.", "Vehicle speed."},
		{"instance_tag", "", "Which instance of this branch the values belong to."},
		{"dimension1", "", "Instance axis 1."},
		{"is_belted", "", "Is belted."},
	}
	for _, tt := range tests {
		got := Summary(plan.Field{Name: tt.name, Doc: tt.doc})
		if got == "" {
			t.Errorf("Summary(%q) is empty", tt.name)
		}
		if !strings.HasPrefix(got, tt.want) {
			t.Errorf("Summary(%q) = %q, want it to start %q", tt.name, got, tt.want)
		}
	}
}

// TestFieldCarriesUnitAndProvenance checks the facts protobuf cannot state on
// its own reach the comment, since an option is invisible in generated docs.
func TestFieldCarriesUnitAndProvenance(t *testing.T) {
	got := Field(plan.Field{
		Name: "speed", Doc: "Vehicle speed.",
		Unit: "UNIT_KILOMETER_PER_HOUR", Element: "ELEMENT_SENSOR",
		FQN: "Vehicle.Speed",
	})
	for _, want := range []string{
		"Vehicle speed.",
		"Unit: km/h.",
		"VSS: Vehicle.Speed (sensor).",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("Field() is missing %q:\n%s", want, got)
		}
	}
}

// TestDeprecatedComesFirst checks AIP-192, which requires the reason to be
// the first line rather than a note at the end.
func TestDeprecatedComesFirst(t *testing.T) {
	got := Field(plan.Field{Name: "pan", Doc: "Mirror pan.", Deprecated: "use Yaw"})
	if !strings.HasPrefix(got, "Deprecated: use Yaw") {
		t.Errorf("Field() does not open with the deprecation reason:\n%s", got)
	}
}

// TestSummaryOpensWithProse guards the defect that left `vdm_uid` with two
// blank comment lines and an empty table cell: text appended to an empty
// description must become the description, not follow a gap.
func TestSummaryOpensWithProse(t *testing.T) {
	got := Summary(plan.Field{Name: "vdm_uid", Doc: "\n\nThe identifier."})
	if strings.HasPrefix(got, "\n") {
		t.Errorf("Summary opens with a blank line: %q", got)
	}
}

// TestEveryEnumValueIsDescribed covers the 396 values the source model leaves
// undocumented.
func TestEveryEnumValueIsDescribed(t *testing.T) {
	tests := []struct {
		name, doc string
		index     int
		want      string
	}{
		{"FORWARD_WHEEL_DRIVE", "", 1, "Forward wheel drive."},
		{"UNDEFINED", "", 0, "Not specified."},
		{"LOCK", "Locked.", 1, "Locked."},
	}
	for _, tt := range tests {
		got := EnumValue(sdl.EnumValue{Name: tt.name, Doc: tt.doc}, tt.index)
		if got != tt.want {
			t.Errorf("EnumValue(%q) = %q, want %q", tt.name, got, tt.want)
		}
	}
}

// TestEnumValueRecordsSourceSpelling checks the note a codec needs: the
// sanitised name is what the schema uses, the source spelling is what a
// VSS-native peer sends.
func TestEnumValueRecordsSourceSpelling(t *testing.T) {
	v := sdl.EnumValue{
		Name: "ROW1",
		Directives: []sdl.Directive{{
			Name: "vspec",
			Args: []sdl.Arg{{Name: "originalName", Raw: "Row1", Str: true}},
		}},
	}
	if got := EnumValue(v, 0); !strings.Contains(got, `spells this "Row1"`) {
		t.Errorf("EnumValue() does not record the source spelling:\n%s", got)
	}
}

// TestUnitSymbol checks the table both the enum comments and the field
// comments read, so the two cannot disagree about what a unit is called.
func TestUnitSymbol(t *testing.T) {
	tests := []struct{ in, want string }{
		{"KILOMETER_PER_HOUR", "km/h"},
		{"DEGREE_CELSIUS", "degC"},
		{"NOT_A_REAL_UNIT", "not a real unit"}, // readable fallback
	}
	for _, tt := range tests {
		if got := UnitSymbol(tt.in); got != tt.want {
			t.Errorf("UnitSymbol(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
