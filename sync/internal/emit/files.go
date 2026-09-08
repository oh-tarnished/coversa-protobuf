// Copyright 2026 The Protobuf Project authors.
// SPDX-License-Identifier: Apache-2.0

package emit

// files.go decides which type goes in which file, and where an enum lands.

import (
	"sort"
	"strings"

	"github.com/the-protobuf-project/vdm/sync/model"
	"github.com/the-protobuf-project/vdm/sync/sdl"
)

// lineCap is the per-file line limit. A file over it is split, never
// compressed.
const lineCap = 250

// File is one planned .proto file.
type File struct {
	Base  string     // file name, e.g. "cabin.proto"
	Type  *sdl.Def   // the object type it holds
	Enums []*sdl.Def // enums that belong with it
	Root  bool       // it holds the package's resource

	// TypesOnly marks a file holding only the enums split out of another.
	TypesOnly bool

	// EnumFile names the sibling this file's enums moved to, so the message
	// can import them.
	EnumFile string

	Imports map[string]bool
	Body    string
}

// planFiles assigns types to files, renders each and returns them.
func (e *Emitter) planFiles(pkg *model.Package) ([]*File, error) {
	fileOf := map[string]string{}
	for _, t := range pkg.Types {
		fileOf[t.Name] = model.TypeFile(t.Name)
	}
	byFile := e.placeEnums(pkg, fileOf)

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
		body, err := e.renderFile(pkg, f, fileOf)
		if err != nil {
			return nil, err
		}
		if split, err := e.splitEnums(pkg, f, fileOf, body); err != nil {
			return nil, err
		} else if split != nil {
			files = append(files, split)
			if body, err = e.renderFile(pkg, f, fileOf); err != nil {
				return nil, err
			}
		}
		f.Body = body
		files = append(files, f)
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Base < files[j].Base })
	return files, nil
}

// placeEnums decides which file each enum is written into.
//
// An enum belongs with the first message that names it, which for VSS is
// always the only message that names it: an allowed-value enum is generated
// per signal. An enum nothing names -- one reached only through the package
// root -- goes with the root.
func (e *Emitter) placeEnums(pkg *model.Package, fileOf map[string]string) map[string][]*sdl.Def {
	placed := map[string]bool{}
	byFile := map[string][]*sdl.Def{}

	for _, t := range pkg.Types {
		for _, f := range t.Fields {
			enum, ok := e.M.Enums[f.Type.Name]
			if !ok || !e.emits(enum.Name) || placed[enum.Name] {
				continue
			}
			placed[enum.Name] = true
			base := fileOf[t.Name]
			byFile[base] = append(byFile[base], enum)
		}
	}
	for _, enum := range pkg.Enums {
		if placed[enum.Name] {
			continue
		}
		placed[enum.Name] = true
		base := model.TypeFile(pkg.Root.Name)
		byFile[base] = append(byFile[base], enum)
	}
	return byFile
}

// splitEnums moves a file's enums into a sibling when the two together would
// exceed the line cap, returning the new file or nil.
//
// The cap wants a file decomposed, not compressed. A message cannot be
// continued across files, so the only split available is to move its enums --
// which is a real decomposition rather than a trick: the allowed-value sets
// are separate declarations that happen to be declared alongside the message
// naming them.
func (e *Emitter) splitEnums(pkg *model.Package, f *File, fileOf map[string]string, body string) (*File, error) {
	if len(f.Enums) == 0 || countLines(body) <= lineCap {
		return nil, nil
	}

	types := &File{
		Base:      strings.TrimSuffix(f.Base, ".proto") + "_types.proto",
		Type:      f.Type,
		Enums:     f.Enums,
		Imports:   map[string]bool{},
		TypesOnly: true,
	}
	rendered, err := e.renderFile(pkg, types, fileOf)
	if err != nil {
		return nil, err
	}
	types.Body = rendered

	// Re-render the message without them, importing the sibling instead.
	f.Enums = nil
	f.Imports = map[string]bool{}
	f.EnumFile = types.Base
	return types, nil
}

// emits reports whether an enum becomes a protobuf enum here, rather than an
// annotation or a documented scalar.
func (e *Emitter) emits(name string) bool {
	return !model.IsUnitEnum(name) && !isScalarEnum(name)
}

// countLines counts the lines in a rendered file.
func countLines(s string) int { return strings.Count(s, "\n") + 1 }
