// Copyright 2026 The Protobuf Project authors.
// SPDX-License-Identifier: Apache-2.0

// Command docs renders the generated schema as Markdown reference
// documentation, one README per package plus an index.
//
// It reads the same specification and builds the same model the sync command
// does, rather than parsing the emitted .proto files back in: several of the
// facts the documentation states -- which methods a resource's shape gets,
// what unit a signal is in -- are decisions the generator made, and
// recovering them from the output would mean deriving them a second time.
//
// Usage:
//
//	docs                 write READMEs under protobuf/covesa
//	docs -pin PATH       read a different revision pin
//	docs -out DIR        write somewhere else
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/the-protobuf-project/vdm/sync/internal/docs"
	"github.com/the-protobuf-project/vdm/sync/internal/model"
	"github.com/the-protobuf-project/vdm/sync/internal/spec"
)

func main() {
	pin := flag.String("pin", "sync/spec.yaml", "the specification revision pin")
	out := flag.String("out", "protobuf/covesa", "directory to write READMEs into")
	flag.Parse()

	if err := run(*pin, *out); err != nil {
		fmt.Fprintln(os.Stderr, "docs:", err)
		os.Exit(1)
	}
}

// run loads the pinned specification and renders its documentation.
func run(pin, out string) error {
	s, err := spec.Load(pin)
	if err != nil {
		return err
	}
	defs, err := loadTree(s.Path)
	if err != nil {
		return err
	}

	m, err := model.Build(defs)
	if err != nil {
		return err
	}
	m.Spec = s

	written, err := docs.New(m).Generate(out)
	if err != nil {
		return err
	}
	fmt.Printf("docs: spec %s (%s) -> %d files\n", s.Version, s.Short(), written)
	return nil
}
