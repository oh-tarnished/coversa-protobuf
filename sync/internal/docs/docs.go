// Copyright 2026 Srikanth Kandarpa.
// SPDX-License-Identifier: Apache-2.0

// Package docs renders the schema as Markdown reference documentation.
//
// # Why it reads the model rather than the .proto files
//
// The obvious implementation parses the emitted protobuf back in. This one
// does not: the model the emitters render from already holds every fact the
// documentation needs -- the resource pattern, which methods a shape gets,
// what a signal's unit is -- and several of those are decisions the generator
// made rather than text it wrote. Recovering them from the output would mean
// re-deriving them, and a second derivation is a second thing that can drift.
//
// One README per package, beside the .proto files it describes, so a reader
// browsing the tree finds the prose where the schema is.
package docs

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/oh-tarnished/coversa-protobuf/sync/model"
	"github.com/oh-tarnished/coversa-protobuf/sync/plan"
)

// Generator renders documentation for one model.
type Generator struct {
	M       *model.Model
	Planner *plan.Planner
}

// New returns a Generator over m.
func New(m *model.Model) *Generator {
	return &Generator{M: m, Planner: plan.New(m)}
}

// Generate writes a README beside every package, plus an index at the root.
func (g *Generator) Generate(out string) (int, error) {
	written := 0
	for _, pkg := range g.M.Packages {
		dir := filepath.Join(append([]string{out}, pkg.Segments()...)...)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return 0, err
		}
		if err := write(dir, g.packageDoc(pkg)); err != nil {
			return 0, err
		}
		written++
	}
	if err := write(out, g.index()); err != nil {
		return 0, err
	}
	return written + 1, nil
}

// write puts a README.md in dir.
func write(dir, body string) error {
	path := filepath.Join(dir, "README.md")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

// firstParagraph returns the leading paragraph of a doc comment, which is the
// summary sentence; the rest is detail the .proto already carries.
func firstParagraph(doc string) string {
	if doc == "" {
		return ""
	}
	if i := strings.Index(doc, "\n\n"); i >= 0 {
		doc = doc[:i]
	}
	return strings.Join(strings.Fields(doc), " ")
}
