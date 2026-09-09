// Copyright 2026 The Protobuf Project authors.
// SPDX-License-Identifier: Apache-2.0

package vss

// convert.go is the reference conversion, both directions.
//
// It works on decoded JSON -- `map[string]any` -- rather than on generated Go
// types, deliberately. A converter tied to the generated structs would only
// serve Go, and would have to be regenerated with them; this one serves
// anything that can hand it protobuf JSON, and is the same algorithm a
// consumer in another language would write from the manifest.

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ToVSS converts a protobuf JSON message into the VSS shape.
//
// message is the message's full protobuf name, e.g.
// "protobuf.covesa.vss.interior.seat.v1.Seat". The result is keyed by fully
// qualified VSS name, which is how a VSS-native system addresses a signal:
//
//	{"Vehicle.Cabin.Seat.Height": 420}
//
// Flat rather than nested, because the fully qualified name *is* the path;
// nesting it would state the same thing twice and force the reader to walk a
// tree to find one signal.
//
// A field the manifest does not know is skipped rather than guessed at. That
// happens when the manifest and the message come from different specification
// revisions, which is why Manifest carries the revision it was built from.
func (m *Manifest) ToVSS(message string, body []byte) (map[string]any, error) {
	var decoded map[string]any
	if err := json.Unmarshal(body, &decoded); err != nil {
		return nil, fmt.Errorf("decode %s: %w", message, err)
	}

	resource, ok := m.Resources[message]
	if !ok {
		return nil, fmt.Errorf("unknown message %q; is the manifest from %s?", message, m.Spec)
	}

	out := map[string]any{}
	for name, value := range decoded {
		field, ok := resource.Fields[protoName(name)]
		if !ok || field.FQN == "" {
			continue
		}
		out[field.FQN] = m.toSource(field, value)
	}
	return out, nil
}

// FromVSS converts a VSS-shaped map back into protobuf JSON.
//
// The inverse of ToVSS, and lossy in exactly one direction: a signal the
// schema does not model has no field to land in and is dropped. Everything
// the schema does model round-trips.
func (m *Manifest) FromVSS(message string, signals map[string]any) ([]byte, error) {
	resource, ok := m.Resources[message]
	if !ok {
		return nil, fmt.Errorf("unknown message %q; is the manifest from %s?", message, m.Spec)
	}

	byFQN := make(map[string]struct {
		name  string
		field *Field
	}, len(resource.Fields))
	for name, f := range resource.Fields {
		if f.FQN == "" {
			continue
		}
		byFQN[f.FQN] = struct {
			name  string
			field *Field
		}{name, f}
	}

	out := map[string]any{}
	for fqn, value := range signals {
		hit, ok := byFQN[fqn]
		if !ok {
			continue
		}
		out[hit.name] = m.fromSource(hit.field, value)
	}
	return json.Marshal(out)
}

// toSource renders one value the way the source model writes it.
//
// Three cases. An enum is translated: the schema emits STATE_ON_VALUE where
// VSS writes "ON", and a peer expecting the latter cannot read the former.
// A nested message is recursed into, so the translation reaches the whole
// tree rather than stopping at the first boundary. Anything else passes
// through -- a number is a number in both.
func (m *Manifest) toSource(f *Field, value any) any {
	if f.Message != "" {
		if nested, ok := value.(map[string]any); ok {
			return m.nest(f.Message, nested, m.toSource)
		}
	}
	s, isString := value.(string)
	if !isString || len(f.Values) == 0 {
		return value
	}
	if source, ok := f.Values[s]; ok {
		return source
	}
	return value
}

// fromSource is the inverse: a source spelling becomes the constant the
// schema emits, recursing the same way.
func (m *Manifest) fromSource(f *Field, value any) any {
	if f.Message != "" {
		if nested, ok := value.(map[string]any); ok {
			return m.nest(f.Message, nested, m.fromSource)
		}
	}
	s, isString := value.(string)
	if !isString || len(f.Values) == 0 {
		return value
	}
	for emitted, source := range f.Values {
		if source == s {
			return emitted
		}
	}
	return value
}

// nest applies convert to every field of an embedded message.
//
// Keyed by the protobuf field name rather than the VSS name: an embedded
// message is not a branch a VSS consumer addresses on its own, so flattening
// it into fully qualified names would invent paths the specification does not
// declare.
func (m *Manifest) nest(message string, body map[string]any, convert func(*Field, any) any) map[string]any {
	resource, ok := m.Resources[message]
	if !ok {
		return body
	}
	out := make(map[string]any, len(body))
	for name, value := range body {
		field, ok := resource.Fields[protoName(name)]
		if !ok {
			out[name] = value
			continue
		}
		out[protoName(name)] = convert(field, value)
	}
	return out
}

// protoName normalises a protobuf JSON key to the field's declared name.
//
// protobuf JSON emits lowerCamelCase by default and accepts the declared
// snake_case on input, so both reach this package and both must resolve.
func protoName(key string) string {
	if !strings.ContainsAny(key, "ABCDEFGHIJKLMNOPQRSTUVWXYZ") {
		return key
	}
	var b strings.Builder
	for i, r := range key {
		if r >= 'A' && r <= 'Z' {
			if i > 0 {
				b.WriteByte('_')
			}
			r += 'a' - 'A'
		}
		b.WriteRune(r)
	}
	return b.String()
}
