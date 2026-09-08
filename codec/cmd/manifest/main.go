// Copyright 2026 The Protobuf Project authors.
// SPDX-License-Identifier: Apache-2.0

// Command manifest writes codec/manifest.json: the language-neutral mapping
// between this schema and the COVESA Vehicle Data Model's own shape.
//
// A consumer in any language reads that file and can convert a message to the
// tree a VSS-native system expects -- without a protobuf toolchain, and
// without reimplementing this repository's naming rules. The Go package
// beside it is the reference implementation of the same conversion.
//
// Usage:
//
//	manifest              write codec/manifest.json
//	manifest -check       fail if it is stale, changing nothing
//	manifest -pin PATH    read a different revision pin
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/the-protobuf-project/vdm/codec/vss"
	"github.com/the-protobuf-project/vdm/sync/model"
	"github.com/the-protobuf-project/vdm/sync/spec"
)

func main() {
	pin := flag.String("pin", "sync/spec.yaml", "the specification revision pin")
	out := flag.String("out", "codec/manifest.json", "where to write the manifest")
	check := flag.Bool("check", false, "fail if the manifest is stale, changing nothing")
	flag.Parse()

	if err := run(*pin, *out, *check); err != nil {
		fmt.Fprintln(os.Stderr, "manifest:", err)
		os.Exit(1)
	}
}

// run builds the manifest and writes or verifies it.
func run(pin, out string, check bool) error {
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

	// Indented and newline-terminated: this file is committed and reviewed,
	// and a one-line JSON blob makes every change look like a rewrite.
	body, err := json.MarshalIndent(vss.Build(m), "", "  ")
	if err != nil {
		return err
	}
	body = append(body, '\n')

	if check {
		have, err := os.ReadFile(out)
		if err != nil || !bytes.Equal(have, body) {
			return fmt.Errorf("%s is stale; run 'just manifest'", out)
		}
		fmt.Printf("manifest: %s is current\n", out)
		return nil
	}

	if err := os.WriteFile(out, body, 0o644); err != nil {
		return err
	}
	fmt.Printf("manifest: spec %s (%s) -> %s\n", s.Version, s.Short(), out)
	return nil
}
