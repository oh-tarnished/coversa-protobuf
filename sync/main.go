// Copyright 2026 The Protobuf Project authors.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func main() {
	pin := flag.String("pin", "sync/spec.yaml", "the specification revision pin")
	out := flag.String("out", "protobuf/covesa", "directory to write generated packages into")
	survey := flag.Bool("survey", false, "report the parsed model instead of emitting")
	flag.Parse()

	spec, err := LoadSpec(*pin)
	if err != nil {
		fmt.Fprintln(os.Stderr, "sync:", err)
		os.Exit(1)
	}

	defs, err := loadSpecTree(spec.Path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "sync:", err)
		os.Exit(1)
	}
	if *survey {
		fmt.Printf("spec revision %s (%s), from %s\n\n", spec.Version, spec.Short(), spec.Path)
		surveyModel(defs)
		return
	}
	if err := generate(spec, defs, *out); err != nil {
		fmt.Fprintln(os.Stderr, "sync:", err)
		os.Exit(1)
	}
}

// loadSpecTree parses every .graphql file under root.
func loadSpecTree(root string) ([]Def, error) {
	var files []string
	err := filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(p, ".graphql") {
			files = append(files, p)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	var defs []Def
	for _, f := range files {
		src, err := os.ReadFile(f)
		if err != nil {
			return nil, err
		}
		d, err := ParseSDL(f, string(src))
		if err != nil {
			return nil, err
		}
		defs = append(defs, d...)
	}
	return defs, nil
}

// surveyModel prints what the parser found, and every field name that trips a
// known AIP naming rule. It is how the rename catalogue in docs/conventions.md
// was built, and it is kept so the catalogue can be rechecked after a spec
// bump rather than trusted.
func surveyModel(defs []Def) {
	kinds := map[string]int{}
	for _, d := range defs {
		kinds[d.Kind]++
	}
	var ks []string
	for k := range kinds {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	for _, k := range ks {
		fmt.Printf("%-14s %d\n", k, kinds[k])
	}

	type hit struct{ owner, field, rule string }
	var hits []hit
	seen := map[string]bool{}
	for _, d := range defs {
		for _, f := range d.Fields {
			n := snake(f.Name)
			var rule string
			switch {
			case n == "name":
				rule = "AIP-122 bare `name` marks the message a resource"
			case strings.HasSuffix(n, "_name") && n != "display_name" && n != "given_name" && n != "family_name":
				rule = "AIP-122 `_name` suffix"
			case n == "state" || n == "status":
				rule = "AIP-216 reserved"
			case hasPreposition(n):
				rule = "AIP-140 preposition"
			case bareTimeUnit(n):
				rule = "AIP-142 bare time unit reads as a Timestamp"
			case strings.HasSuffix(n, "_time") || strings.HasSuffix(n, "_date"):
				rule = "AIP-142 `_time`/`_date` suffix implies Timestamp"
			case strings.HasSuffix(n, "_id"):
				rule = "AIP-122 `_id` suffix"
			}
			if rule != "" {
				key := d.Name + "." + n + rule
				if !seen[key] {
					seen[key] = true
					hits = append(hits, hit{d.Name, f.Name + " -> " + n, rule})
				}
			}
		}
	}
	sort.Slice(hits, func(i, j int) bool {
		if hits[i].rule != hits[j].rule {
			return hits[i].rule < hits[j].rule
		}
		return hits[i].owner < hits[j].owner
	})
	fmt.Printf("\nAIP naming traps: %d\n", len(hits))
	for _, h := range hits {
		fmt.Printf("  %-40s %-46s %s\n", h.owner, h.field, h.rule)
	}
}
