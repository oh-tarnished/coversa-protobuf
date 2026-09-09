// Copyright 2026 The Protobuf Project authors.
// SPDX-License-Identifier: Apache-2.0

package spec

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// pins is a well-formed two-source pin, the shape sync/spec.yaml carries.
const pins = `# a comment
vss:
  version: 2026-09-02
  commit: cd4bc50ba4aae74fad2172ee34f71acbeab79837
  source: https://github.com/COVESA/vehicle_signal_specification
  path: vss/spec/VehicleSignalSpecification.vspec
vdm:
  version: 2026-07-24
  commit: 36bc93936eab097647b52eea377a3374bcf926f5
  source: https://github.com/COVESA/vdm
  path: vdm/spec
`

// write puts a pin in a temporary directory and returns its path.
func write(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "spec.yaml")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoad(t *testing.T) {
	s, err := Load(write(t, pins))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	for _, tt := range []struct{ name, got, want string }{
		{"vss.version", s.VSS.Version, "2026-09-02"},
		{"vss.short", s.VSS.Short(), "cd4bc50"},
		{"vss.path", s.VSS.Path, "vss/spec/VehicleSignalSpecification.vspec"},
		{"vdm.version", s.VDM.Version, "2026-07-24"},
		{"vdm.short", s.VDM.Short(), "36bc939"},
		{"vdm.path", s.VDM.Path, "vdm/spec"},
	} {
		if tt.got != tt.want {
			t.Errorf("%s = %q, want %q", tt.name, tt.got, tt.want)
		}
	}
}

// TestLoadRejects is the rule the package exists to enforce: a malformed pin
// is an error, never an empty field silently stamped into every banner.
//
// Each case damages one source and leaves the other well-formed, so what is
// under test is that the bad one is caught rather than averaged away.
func TestLoadRejects(t *testing.T) {
	tests := []struct{ name, body, want string }{
		{
			"unknown key",
			"vss:\n  revision: abc\n",
			"not found",
		},
		{
			"unknown source",
			pins + "cvis:\n  version: 2026-08-21\n",
			"not found",
		},
		{
			"missing commit",
			strings.Replace(pins, "  commit: cd4bc50ba4aae74fad2172ee34f71acbeab79837\n", "", 1),
			"vss.commit is empty",
		},
		{
			"version is not a date",
			strings.Replace(pins, "version: 2026-07-24", "version: v1", 1),
			"vdm.version \"v1\" is not a YYYY-MM-DD date",
		},
		{
			"one source missing entirely",
			"vss:\n  version: 2026-09-02\n  commit: cd4bc50\n  source: x\n  path: y\n",
			"vdm.version is empty",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Load(write(t, tt.body))
			if err == nil {
				t.Fatal("Load accepted a malformed pin; want an error")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("error = %v, want it to mention %q", err, tt.want)
			}
		})
	}
}

// TestBanner checks both revisions reach the text stamped into every file.
// A banner naming one source would leave the other unanswerable from a file a
// consumer is holding, which is the whole point of stamping it.
func TestBanner(t *testing.T) {
	s, err := Load(write(t, pins))
	if err != nil {
		t.Fatal(err)
	}
	banner := s.Banner()
	for _, want := range []string{
		"SPDX-License-Identifier: Apache-2.0",
		"revision 2026-09-02 (cd4bc50)",
		"revision 2026-07-24 (36bc939)",
		"DO NOT EDIT.",
		"Regenerate with: just sync",
	} {
		if !strings.Contains(banner, want) {
			t.Errorf("banner is missing %q:\n%s", want, banner)
		}
	}
}
