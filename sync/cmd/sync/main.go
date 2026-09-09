// Copyright 2026 The Protobuf Project authors.
// SPDX-License-Identifier: Apache-2.0

// Command sync generates this repository's protobuf from the COVESA Vehicle
// Signal Specification and Vehicle Data Model.
//
// Everything under protobuf/ is written by this program and carries a
// DO NOT EDIT banner naming the revisions it came from. A fix belongs here --
// usually in one of the named tables in sync/catalog or sync/model -- never
// in an emitted file, which the next run reverts.
//
// Usage:
//
//	sync                 regenerate protobuf/covesa from the pinned revisions
//	sync -survey         report the parsed model and its AIP naming traps
//	sync -pin PATH       read a different revision pin
//	sync -out DIR        write somewhere other than protobuf/covesa
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/the-protobuf-project/vdm/sync/internal/emit"
	"github.com/the-protobuf-project/vdm/sync/load"
)

func main() {
	pin := flag.String("pin", "sync/spec.yaml", "the specification revision pins")
	out := flag.String("out", "protobuf/covesa", "directory to write generated packages into")
	survey := flag.Bool("survey", false, "report the parsed model instead of emitting")
	flag.Parse()

	if err := run(*pin, *out, *survey); err != nil {
		fmt.Fprintln(os.Stderr, "sync:", err)
		os.Exit(1)
	}
}

// run loads the pinned specifications and either surveys or emits them.
func run(pin, out string, survey bool) error {
	m, err := load.Model(pin)
	if err != nil {
		return err
	}
	if survey {
		return surveyModel(m)
	}

	total, err := emit.New(m).Generate(out)
	if err != nil {
		return err
	}
	fmt.Printf("sync: vss %s (%s) + vdm %s (%s) -> %d packages, %d files\n",
		m.Spec.VSS.Version, m.Spec.VSS.Short(),
		m.Spec.VDM.Version, m.Spec.VDM.Short(),
		len(m.Packages), total)
	return nil
}
