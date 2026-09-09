// Copyright 2026 The Protobuf Project authors.
// SPDX-License-Identifier: Apache-2.0

package emit

// files.go decides which type goes in which file, and where an enum lands.

import (
	"sort"
	"strings"

	"github.com/oh-tarnished/coversa-protobuf/sync/model"
	"github.com/oh-tarnished/coversa-protobuf/sync/vspec"
)

// lineCap is the per-file line limit. A file over it is split, never
// compressed.
const lineCap = 250

// File is one planned .proto file.
type File struct {
	Base  string        // file name, e.g. "cabin.proto"
	Type  *vspec.Node   // the message it holds
	Enums []*vspec.Node // signals whose allowed values become enums
	Root  bool          // it holds the package's resource

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
	// Indexed twice, by fully qualified name and by emitted message name.
	// A field usually names the node it points at, but one promoted to
	// another type -- a date string becoming a Date -- names only the type,
	// and the import has to resolve either way.
	fileOf := map[string]string{}
	for _, t := range pkg.Types {
		base := model.TypeFile(t.Name)
		fileOf[t.FQN] = base
		fileOf[model.MessageName(t.Name)] = base
	}
	byFile := e.placeEnums(pkg, fileOf)

	var files []*File
	for _, t := range pkg.Types {
		base := fileOf[t.FQN]
		f := &File{
			Base:    base,
			Type:    t,
			Enums:   byFile[base],
			Root:    t.FQN == pkg.Root.FQN,
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
// An enum belongs with the message whose signal declares it, which for VSS is
// always exactly one message: an allowed-value set is written on the signal
// it constrains.
func (e *Emitter) placeEnums(pkg *model.Package, fileOf map[string]string) map[string][]*vspec.Node {
	byFile := map[string][]*vspec.Node{}
	for _, t := range pkg.Types {
		for _, c := range t.Children {
			if !model.HasEnum(c) {
				continue
			}
			base := fileOf[t.FQN]
			byFile[base] = append(byFile[base], c)
		}
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

// countLines counts the lines in a rendered file.
func countLines(s string) int { return strings.Count(s, "\n") + 1 }
