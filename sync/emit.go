// Copyright 2026 The Protobuf Project authors.
// SPDX-License-Identifier: Apache-2.0

package main

// emit.go plans the files of each package and writes them.
//
// One message per file. A protobuf message cannot be continued across files,
// so the only decomposition available is one type at a time -- which also
// keeps each file as small as the source model allows, and makes the file a
// stable address for a type rather than an accident of grouping.

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Root_ is the import prefix for this package: family, then domain when the
// package has one. Vehicle itself sits directly under the family, because it
// is the root every domain hangs from rather than a member of one.
func (p *Package) Root_() string {
	if p.Domain == "" {
		return covesaRoot + "/" + string(p.Family)
	}
	return covesaRoot + "/" + string(p.Family) + "/" + p.Domain
}

// segments is the path from the family root down to this package.
func (p *Package) segments() []string {
	if p.Domain == "" {
		return []string{string(p.Family), p.Dir}
	}
	return []string{string(p.Family), p.Domain, p.Dir}
}

// JavaPackage is the AIP-191 java_package for this package.
func (p *Package) JavaPackage() string {
	return javaBase + "." + strings.Join(p.segments(), ".") + ".v1"
}

// ImportPath is the import path of one file in this package.
func (p *Package) ImportPath(base string) string {
	return p.Root_() + "/" + p.Dir + "/v1/" + base
}

// File is one planned .proto file.
type File struct {
	Base  string // file name without directory, e.g. "cabin.proto"
	Type  *Def   // the object type it holds
	Enums []*Def // enums that belong with it
	Root  bool   // it holds the package's resource
	// TypesOnly marks a file that holds only the enums split out of another,
	// written when the two together would exceed the line cap.
	TypesOnly bool
	// EnumFile names the sibling this file's enums were moved to, so the
	// message can import them.
	EnumFile string
	Imports  map[string]bool
	Body     string
}

// lineCap is the per-file line limit; a file over it is split, never
// compressed.
const lineCap = 250

// countLines counts the lines in a rendered file.
func countLines(s string) int { return strings.Count(s, "\n") + 1 }

// generate renders every VSS package into out.
func generate(spec Spec, defs []Def, out string) error {
	m, err := BuildModel(defs)
	if err != nil {
		return err
	}
	m.Spec = spec
	// The output tree is rewritten, not merged into. A type that stops being
	// generated -- a value object promoted to its own package, an enum
	// replaced by a scalar -- would otherwise leave its file behind, and a
	// stale .proto still compiles and still lints, so nothing would report it.
	if err := os.RemoveAll(out); err != nil {
		return err
	}

	total, err := generateVocab(m.Spec, defs, out)
	if err != nil {
		return err
	}
	for _, pkg := range m.Packages {
		files, err := m.planPackage(pkg)
		if err != nil {
			return err
		}
		dir := filepath.Join(append([]string{out}, append(pkg.segments(), "v1")...)...)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
		for _, f := range files {
			path := filepath.Join(dir, f.Base)
			if err := os.WriteFile(path, []byte(f.Body), 0o644); err != nil {
				return err
			}
			total++
		}
		// Rule 8: every resource gets its full standard method set. The shape
		// depends on whether a vehicle holds one of this branch or many --
		// see rpc.go.
		for base, body := range map[string]string{
			"messages.proto": m.renderMessages(pkg),
			"service.proto":  m.renderService(pkg),
		} {
			if err := os.WriteFile(filepath.Join(dir, base), []byte(body), 0o644); err != nil {
				return err
			}
			total++
		}
	}
	fmt.Printf("sync: spec %s (%s) -> %d packages, %d files\n", spec.Version, spec.Short(), len(m.Packages), total)
	return nil
}

// planPackage assigns types to files, renders each and returns them.
func (m *Model) planPackage(pkg *Package) ([]*File, error) {
	// Which file each type and enum lands in, so cross-references can be
	// turned into imports.
	fileOf := map[string]string{}
	for _, t := range pkg.Types {
		fileOf[t.Name] = typeFile(t.Name)
	}

	// An enum belongs with the first message that names it, which for VSS is
	// always the only message that names it: an allowed-value enum is
	// generated per signal.
	enumFile := map[string]string{}
	byFile := map[string][]*Def{}
	for _, t := range pkg.Types {
		for _, f := range t.Fields {
			if e, ok := m.Enums[f.Type.Name]; ok && !isUnitEnum(e.Name) && !isScalarEnum(e.Name) {
				if _, done := enumFile[e.Name]; !done {
					enumFile[e.Name] = fileOf[t.Name]
					byFile[fileOf[t.Name]] = append(byFile[fileOf[t.Name]], e)
				}
			}
		}
	}
	for _, e := range pkg.Enums {
		if _, done := enumFile[e.Name]; !done {
			base := typeFile(pkg.Root.Name)
			enumFile[e.Name] = base
			byFile[base] = append(byFile[base], e)
		}
	}
	for _, e := range enumFile {
		_ = e
	}

	var files []*File
	for _, t := range pkg.Types {
		base := fileOf[t.Name]
		f := &File{
			Base:    base,
			Type:    t,
			Enums:   byFile[base],
			Root:    t.Name == pkg.Root.Name,
			Imports: map[string]bool{},
		}
		body, err := m.renderFile(pkg, f, fileOf)
		if err != nil {
			return nil, err
		}
		// The 250-line cap wants a file decomposed, not compressed. A message
		// cannot be continued across files, so the only split available is to
		// move its enums into a sibling -- which is a real decomposition, not
		// a trick: the allowed-value sets are separate declarations that
		// happen to be declared alongside the message that names them.
		//
		// Done only when the combined file is over the cap, so a small branch
		// keeps its enums beside the message they belong to.
		if len(f.Enums) > 0 && countLines(body) > lineCap {
			tf := &File{
				Base:      strings.TrimSuffix(base, ".proto") + "_types.proto",
				Type:      t,
				Enums:     f.Enums,
				Imports:   map[string]bool{},
				TypesOnly: true,
			}
			tb, err := m.renderFile(pkg, tf, fileOf)
			if err != nil {
				return nil, err
			}
			tf.Body = tb
			files = append(files, tf)

			f.Enums = nil
			f.Imports = map[string]bool{}
			f.EnumFile = tf.Base
			body, err = m.renderFile(pkg, f, fileOf)
			if err != nil {
				return nil, err
			}
		}
		f.Body = body
		files = append(files, f)
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Base < files[j].Base })
	return files, nil
}
