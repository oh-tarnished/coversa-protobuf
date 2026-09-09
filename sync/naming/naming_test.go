// Copyright 2026 Srikanth Kandarpa.
// SPDX-License-Identifier: Apache-2.0

package naming

import (
	"slices"
	"testing"
)

func TestSplitWords(t *testing.T) {
	tests := []struct {
		in   string
		want []string
	}{
		{"driverPosition", []string{"driver", "position"}},
		{"Chassis_Axle", []string{"chassis", "axle"}},
		{"emissionsCo2", []string{"emissions", "co2"}},
		{"ADASConfig", []string{"adas", "config"}},
		{"Row1", []string{"row1"}},
		// AIP-140's abbreviation table applies to every identifier, so it is
		// enforced here rather than in each caller.
		{"configuration", []string{"config"}},
		{"VehicleConfiguration", []string{"vehicle", "config"}},
	}
	for _, tt := range tests {
		if got := SplitWords(tt.in); !slices.Equal(got, tt.want) {
			t.Errorf("SplitWords(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func TestCases(t *testing.T) {
	tests := []struct{ in, snake, pascal, lowerCamel string }{
		{"driverPosition", "driver_position", "DriverPosition", "driverPosition"},
		{"Chassis_Axle", "chassis_axle", "ChassisAxle", "chassisAxle"},
		{"ControlUnit", "control_unit", "ControlUnit", "controlUnit"},
	}
	for _, tt := range tests {
		if got := Snake(tt.in); got != tt.snake {
			t.Errorf("Snake(%q) = %q, want %q", tt.in, got, tt.snake)
		}
		if got := Pascal(tt.in); got != tt.pascal {
			t.Errorf("Pascal(%q) = %q, want %q", tt.in, got, tt.pascal)
		}
		if got := LowerCamel(tt.in); got != tt.lowerCamel {
			t.Errorf("LowerCamel(%q) = %q, want %q", tt.in, got, tt.lowerCamel)
		}
	}
}

// TestPluralise covers the rule-driven cases and the irregulars.
//
// A plural is load-bearing: AIP-123 ties it to the collection segment of
// every resource name and to the service name, so "Chassiss" would appear in
// every URL and every generated client.
func TestPluralise(t *testing.T) {
	tests := []struct{ in, want string }{
		{"seat", "seats"},
		{"door", "doors"},
		{"velocity", "velocities"}, // consonant + y
		{"angularVelocity", "angularVelocities"},
		{"person", "people"},         // irregular
		{"chassis", "chassisSet"},    // sibilant, and "chassises" is not a word
		{"adas", "adas"},             // already plural
		{"powertrain", "powertrain"}, // mass noun
		{"mirrors", "mirrors"},       // already plural in the source model
	}
	for _, tt := range tests {
		if got := Pluralise(tt.in); got != tt.want {
			t.Errorf("Pluralise(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
