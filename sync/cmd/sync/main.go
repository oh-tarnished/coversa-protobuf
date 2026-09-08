// Copyright 2026 The Protobuf Project authors.
// SPDX-License-Identifier: Apache-2.0

// Command sync generates this repository's protobuf from the COVESA Vehicle
// Data Model.
//
// Everything under protobuf/ is written by this program and carries a
// DO NOT EDIT banner naming the specification revision it came from. A fix
// belongs here -- usually in one of the named tables in internal/catalog or
// internal/model -- never in an emitted file, which the next run reverts.
//
// Usage:
//
//	sync                 regenerate protobuf/covesa from the pinned revision
//	sync -survey         report the parsed model and its AIP naming traps
//	sync -pin PATH       read a different revision pin
//	sync -out DIR        write somewhere other than protobuf/covesa
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/the-protobuf-project/vdm/sync/internal/emit"
	"github.com/the-protobuf-project/vdm/sync/internal/model"
	"github.com/the-protobuf-project/vdm/sync/internal/spec"
)

func main() {
	pin := flag.String("pin", "sync/spec.yaml", "the specification revision pin")
	out := flag.String("out", "protobuf/covesa", "directory to write generated packages into")
	survey := flag.Bool("survey", false, "report the parsed model instead of emitting")
	flag.Parse()

	if err := run(*pin, *out, *survey); err != nil {
		fmt.Fprintln(os.Stderr, "sync:", err)
		os.Exit(1)
	}
}

// run loads the pinned specification and either surveys or emits it.
func run(pin, out string, survey bool) error {
	s, err := spec.Load(pin)
	if err != nil {
		return err
	}
	defs, err := loadTree(s.Path)
	if err != nil {
		return err
	}

	if survey {
		fmt.Printf("spec revision %s (%s), from %s\n\n", s.Version, s.Short(), s.Path)
		return surveyModel(defs)
	}

	m, err := model.Build(defs)
	if err != nil {
		return err
	}
	m.Spec = s

	total, err := emit.New(m).Generate(defs, out)
	if err != nil {
		return err
	}
	fmt.Printf("sync: spec %s (%s) -> %d packages, %d files\n",
		s.Version, s.Short(), len(m.Packages), total)
	return nil
}
