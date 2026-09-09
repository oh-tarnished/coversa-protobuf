// Copyright 2026 Srikanth Kandarpa.
// SPDX-License-Identifier: Apache-2.0

package vspec

// tree.go turns a decoded YAML mapping into nodes.
//
// Every key of a mapping is either one of VSS's own attributes -- `type`,
// `description`, `datatype` and the rest -- or the name of a child node.
// yaml.v3 offers no "everything else" tag, so the mapping is walked by hand
// and each key sorted into one or the other.

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// attributes are the keys VSS defines on a node. Anything else is a child.
var attributes = map[string]bool{
	"type": true, "description": true, "comment": true, "datatype": true,
	"unit": true, "min": true, "max": true, "allowed": true,
	"default": true, "pattern": true, "deprecation": true, "instances": true,
	"enum": true,
	// Present in the specification and not modelled here: `aggregate` marks a
	// struct whose members travel together, which this schema expresses by
	// the message itself, and `expand` suppresses instance expansion.
	"aggregate": true, "expand": true,
}

// mapping converts a YAML mapping node into the nodes it declares.
//
// A key may be a dotted path rather than a bare name: the root file writes
//
//	Vehicle.Powertrain:
//	  type: branch
//
// which declares Powertrain *beneath* Vehicle, not a sibling of it called
// "Vehicle.Powertrain". Such a node is spliced into the tree at its path, and
// the ancestors must already have been declared -- which the specification
// guarantees by writing them first.
func mapping(m *yaml.Node, prefix, file string) ([]*Node, error) {
	if m.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("%s: expected a mapping at line %d", file, m.Line)
	}

	var out []*Node
	for i := 0; i+1 < len(m.Content); i += 2 {
		key, body := m.Content[i].Value, m.Content[i+1]
		if attributes[key] {
			continue
		}

		path := strings.Split(key, ".")
		name := path[len(path)-1]

		node, err := decode(name, body, join(prefix, dotted(path[:len(path)-1])), file)
		if err != nil {
			return nil, err
		}
		if len(path) == 1 {
			out = append(out, node)
			continue
		}

		parent, err := descend(out, path[:len(path)-1])
		if err != nil {
			return nil, fmt.Errorf("%s: %s at line %d: %w", file, key, body.Line, err)
		}
		parent.Children = append(parent.Children, node)
	}
	return out, nil
}

// descend walks a dotted path through nodes already declared in this file.
func descend(nodes []*Node, path []string) (*Node, error) {
	var found *Node
	for _, n := range nodes {
		if n.Name == path[0] {
			found = n
			break
		}
	}
	if found == nil {
		return nil, fmt.Errorf("no node named %q declared before it", path[0])
	}
	if len(path) == 1 {
		return found, nil
	}
	return descend(found.Children, path[1:])
}

// decode builds one node from its YAML body.
func decode(name string, body *yaml.Node, prefix, file string) (*Node, error) {
	var e entry
	if err := body.Decode(&e); err != nil {
		return nil, fmt.Errorf("%s: %s at line %d: %w", file, name, body.Line, err)
	}

	node := &Node{
		Name:        name,
		FQN:         join(prefix, name),
		Kind:        Kind(strings.TrimSpace(e.Type)),
		Description: clean(e.Description),
		Comment:     clean(e.Comment),
		Datatype:    strings.TrimSpace(e.Datatype),
		Unit:        strings.TrimSpace(e.Unit),
		Min:         scalar(e.Min),
		Max:         scalar(e.Max),
		Allowed:     e.Allowed,
		Enum:        enumEntries(valueFor(body, "enum")),
		Default:     scalar(e.Default),
		Pattern:     e.Pattern,
		Deprecation: clean(e.Deprecation),
		File:        file,
	}
	if node.Kind == "" {
		return nil, fmt.Errorf("%s: %s at line %d has no type", file, node.FQN, body.Line)
	}

	instances, err := expandInstances(e.Instances)
	if err != nil {
		return nil, fmt.Errorf("%s: %s: %w", file, node.FQN, err)
	}
	node.Instances = instances

	children, err := mapping(body, node.FQN, file)
	if err != nil {
		return nil, err
	}
	node.Children = children
	return node, nil
}

// scalar renders a YAML scalar as the text VSS wrote.
//
// A bound may decode as an int or a float depending on how it was written,
// and the datatype -- not the literal -- decides how it should be rendered
// downstream. Keeping the original text defers that decision.
func scalar(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case bool:
		if t {
			return "true"
		}
		return "false"
	case int:
		return fmt.Sprintf("%d", t)
	case float64:
		// Render a whole number without a trailing ".0", so a bound written
		// `100` does not become `100.0` and disagree with an integer field.
		if t == float64(int64(t)) {
			return fmt.Sprintf("%d", int64(t))
		}
		return fmt.Sprintf("%g", t)
	default:
		return fmt.Sprintf("%v", t)
	}
}

// clean normalises a description or comment into one paragraph.
//
// VSS wraps prose across lines with leading indentation, which YAML folds
// into a single string carrying the newlines. Those newlines are layout, not
// structure, so they collapse to spaces.
func clean(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// dotted joins path segments, returning "" for an empty path so join does not
// append a bare separator.
func dotted(segments []string) string {
	if len(segments) == 0 {
		return ""
	}
	return strings.Join(segments, ".")
}
