// Copyright 2026 The Protobuf Project authors.
// SPDX-License-Identifier: Apache-2.0

package spec

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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
	s, err := Load(write(t, `# a comment
version: 2026-07-24
commit: 36bc93936eab097647b52eea377a3374bcf926f5
source: https://github.com/COVESA/vdm
path: vdm/spec
`))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if s.Version != "2026-07-24" {
		t.Errorf("Version = %q", s.Version)
	}
	if s.Short() != "36bc939" {
		t.Errorf("Short() = %q, want 36bc939", s.Short())
	}
	if s.Path != "vdm/spec" {
		t.Errorf("Path = %q", s.Path)
	}
}

// TestLoadRejects is the rule the package exists to enforce: a malformed pin
// is an error, never an empty field silently stamped into every banner.
func TestLoadRejects(t *testing.T) {
	tests := []struct{ name, body, want string }{
		{
			"unknown key",
			"version: 2026-07-24\nrevision: abc\n",
			"unknown key",
		},
		{
			"missing commit",
			"version: 2026-07-24\nsource: x\npath: y\n",
			"commit is empty",
		},
		{
			"version is not a date",
			"version: v1\ncommit: abc\nsource: x\npath: y\n",
			"not a YYYY-MM-DD date",
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

// TestBanner checks the revision reaches the text stamped into every file.
func TestBanner(t *testing.T) {
	s := Spec{Version: "2026-07-24", Commit: "36bc93936eab097647b52eea"}
	banner := s.Banner()
	for _, want := range []string{
		"SPDX-License-Identifier: Apache-2.0",
		"spec revision 2026-07-24 (36bc939). DO NOT EDIT.",
		"Regenerate with: just sync",
	} {
		if !strings.Contains(banner, want) {
			t.Errorf("banner is missing %q:\n%s", want, banner)
		}
	}
}
