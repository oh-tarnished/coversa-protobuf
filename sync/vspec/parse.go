// Copyright 2026 The Protobuf Project authors.
// SPDX-License-Identifier: Apache-2.0

package vspec

// parse.go loads one .vspec file and resolves the includes it names.

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// entry is one node as it appears in the YAML, before the tree is assembled.
//
// Every field VSS defines is named here: a key the specification uses and
// this struct does not would be dropped silently, so the decoder is told to
// reject unknown keys and the list is kept complete.
type entry struct {
	Type        string     `yaml:"type"`
	Description string     `yaml:"description"`
	Comment     string     `yaml:"comment"`
	Datatype    string     `yaml:"datatype"`
	Unit        string     `yaml:"unit"`
	Min         any        `yaml:"min"`
	Max         any        `yaml:"max"`
	Allowed     []string   `yaml:"allowed"`
	Enum        *yaml.Node `yaml:"enum"`
	Default     any        `yaml:"default"`
	Pattern     string     `yaml:"pattern"`
	Deprecation string     `yaml:"deprecation"`
	Instances   any        `yaml:"instances"`

	// Children are every other key: a nested node. yaml.v3 has no "rest"
	// tag, so the raw node is kept and walked in tree.go.
	rest *yaml.Node
}

// Parse reads the specification rooted at path.
func Parse(path string) (*Node, error) {
	nodes, err := parseFile(path, "", filepath.Dir(path))
	if err != nil {
		return nil, err
	}
	if len(nodes) != 1 {
		return nil, fmt.Errorf("%s: expected one root node, found %d", path, len(nodes))
	}
	return nodes[0], nil
}

// parseFile reads one file and returns its top-level nodes, with includes
// already grafted in.
func parseFile(path, prefix, specRoot string) ([]*Node, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	text := string(raw)

	var doc yaml.Node
	if err := yaml.Unmarshal([]byte(text), &doc); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}

	var nodes []*Node
	if len(doc.Content) > 0 {
		nodes, err = mapping(doc.Content[0], prefix, path)
		if err != nil {
			return nil, err
		}
	}
	return graft(nodes, text, path, prefix, specRoot)
}

// graft resolves every `#include` in the file and attaches the result.
//
// `#include` is not YAML -- a YAML parser sees a comment -- so it is read
// from the source text after the document is decoded. The form is
//
//	#include <path> [attachment point]
//
// where the attachment point is a dotted path relative to this file's own
// nodes. Without one, the included nodes join at the top level.
func graft(nodes []*Node, text, path, prefix, specRoot string) ([]*Node, error) {

	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "#include") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			return nil, fmt.Errorf("%s: malformed include: %q", path, line)
		}

		target := ""
		if len(fields) > 2 {
			target = fields[2]
		}
		attachPrefix := prefix
		if target != "" {
			attachPrefix = join(prefix, target)
		}

		resolved, err := resolve(fields[1], filepath.Dir(path), specRoot)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		included, err := parseFile(resolved, attachPrefix, specRoot)
		if err != nil {
			return nil, err
		}
		if nodes, err = attach(nodes, target, included, path); err != nil {
			return nil, err
		}
	}
	return nodes, nil
}

// attach places the included nodes at target, a dotted path into nodes.
//
// An empty target means the top level. A target naming a node that does not
// exist is an error rather than a silently created branch: the specification
// always declares the branch before including into it, and a typo that
// invented one would put signals somewhere no consumer expects.
func attach(nodes []*Node, target string, included []*Node, path string) ([]*Node, error) {
	if target == "" {
		return append(nodes, included...), nil
	}

	parent, err := find(nodes, strings.Split(target, "."))
	if err != nil {
		return nil, fmt.Errorf("%s: include into %q: %w", path, target, err)
	}
	parent.Children = append(parent.Children, included...)
	return nodes, nil
}

// find walks a dotted path through the tree.
func find(nodes []*Node, segments []string) (*Node, error) {
	var found *Node
	for _, n := range nodes {
		if n.Name == segments[0] {
			found = n
			break
		}
	}
	if found == nil {
		return nil, fmt.Errorf("no node named %q", segments[0])
	}
	if len(segments) == 1 {
		return found, nil
	}
	return find(found.Children, segments[1:])
}

// resolve finds an included file.
//
// VSS uses two conventions and both appear in the catalogue: a path relative
// to the including file (`../include/AirComposition.vspec`, from
// Vehicle/Exterior.vspec) and one relative to the specification root
// (`include/PowerOptimize.vspec`, from Vehicle/Vehicle.vspec). The including
// file wins where both would resolve, which is what vss-tools does.
func resolve(name, dir, specRoot string) (string, error) {
	for _, base := range []string{dir, specRoot} {
		candidate := filepath.Join(base, name)
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("include %q not found in %s or %s", name, dir, specRoot)
}

// join appends a dotted segment to a prefix, tolerating either being empty.
//
// Both are: a node at the root has no prefix, and a node whose key is a bare
// name contributes no intermediate path. Concatenating regardless produced
// `Vehicle.Cabin.Seat..OccupancyStatus`, an fqn that matches nothing.
func join(prefix, name string) string {
	switch {
	case prefix == "":
		return name
	case name == "":
		return prefix
	default:
		return prefix + "." + name
	}
}
