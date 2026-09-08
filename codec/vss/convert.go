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
		return nil, fmt.Errorf("unknown message %q; is the manifest from spec %s?", message, m.Version)
	}

	out := map[string]any{}
	for name, value := range decoded {
		field, ok := resource.Fields[protoName(name)]
		if !ok || field.FQN == "" {
			continue
		}
		out[field.FQN] = toSourceValue(field, value)
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
		return nil, fmt.Errorf("unknown message %q; is the manifest from spec %s?", message, m.Version)
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
		out[hit.name] = fromSourceValue(hit.field, value)
	}
	return json.Marshal(out)
}

// toSourceValue renders one value the way the source model writes it.
//
// Only enums differ: the schema emits SEAT_INSTANCE_TAG_DIMENSION1_ROW1 where
// VSS writes "Row1", and a peer expecting the latter cannot read the former.
func toSourceValue(f *Field, value any) any {
	s, isString := value.(string)
	if !isString || len(f.Values) == 0 {
		return value
	}
	if source, ok := f.Values[s]; ok {
		return source
	}
	return value
}

// fromSourceValue is the inverse: a source spelling becomes the constant the
// schema emits.
func fromSourceValue(f *Field, value any) any {
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
