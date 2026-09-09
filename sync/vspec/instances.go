// Copyright 2026 Srikanth Kandarpa.
// SPDX-License-Identifier: Apache-2.0

package vspec

// instances.go expands VSS's instance shorthand.
//
// An instance declaration says a branch occurs more than once and names the
// occurrences. It takes three forms:
//
//	instances: ["Front", "Rear"]        one axis, listed
//	instances: Row[1,2]                 one axis, a range
//	instances:                          two axes, combined
//	  - Row[1,2]
//	  - ["DriverSide", "Middle", "PassengerSide"]
//
// The last is why a seat is Row1/DriverSide rather than seat number four:
// the axes multiply, and the pair together names one occurrence.
//
// `Row[1,2]` is not YAML. It is VSS shorthand for a numeric range with a
// prefix, and expanding it here is what lets everything downstream work in
// explicit names.

import (
	"fmt"
	"regexp"
	"strconv"
)

// rangePattern matches VSS's `Prefix[low,high]` shorthand.
var rangePattern = regexp.MustCompile(`^([A-Za-z]*)\[(\d+),(\d+)\]$`)

// expandInstances turns the `instances` value into explicit axes.
//
// The result is a list of axes, each a list of names. A single axis is
// returned as one entry, so a caller need not distinguish the forms.
func expandInstances(v any) ([][]string, error) {
	switch t := v.(type) {
	case nil:
		return nil, nil
	case string:
		axis, err := expandAxis(t)
		return [][]string{axis}, err
	case []any:
		return expandList(t)
	default:
		return nil, fmt.Errorf("instances: unsupported form %T", v)
	}
}

// expandList handles both a single listed axis and a list of axes.
//
// The two are told apart by their contents: a list of strings is one axis,
// a list containing a nested list is several. A list of strings that are
// *themselves* ranges is also several axes, which is the `- Row[1,2]` form.
func expandList(items []any) ([][]string, error) {
	var (
		axes  [][]string
		plain []string
	)
	for _, item := range items {
		switch t := item.(type) {
		case []any:
			axis, err := names(t)
			if err != nil {
				return nil, err
			}
			axes = append(axes, axis)
		case string:
			if rangePattern.MatchString(t) {
				axis, err := expandAxis(t)
				if err != nil {
					return nil, err
				}
				axes = append(axes, axis)
				continue
			}
			plain = append(plain, t)
		default:
			return nil, fmt.Errorf("instances: unsupported entry %T", item)
		}
	}
	if len(plain) > 0 {
		axes = append(axes, plain)
	}
	return axes, nil
}

// expandAxis turns one axis declaration into its names.
func expandAxis(s string) ([]string, error) {
	m := rangePattern.FindStringSubmatch(s)
	if m == nil {
		return []string{s}, nil
	}

	prefix, lo, hi := m[1], m[2], m[3]
	low, err := strconv.Atoi(lo)
	if err != nil {
		return nil, fmt.Errorf("instances: %q: %w", s, err)
	}
	high, err := strconv.Atoi(hi)
	if err != nil {
		return nil, fmt.Errorf("instances: %q: %w", s, err)
	}
	if high < low {
		return nil, fmt.Errorf("instances: %q counts backwards", s)
	}

	out := make([]string, 0, high-low+1)
	for i := low; i <= high; i++ {
		out = append(out, prefix+strconv.Itoa(i))
	}
	return out, nil
}

// names converts a decoded YAML sequence into instance names.
func names(items []any) ([]string, error) {
	out := make([]string, 0, len(items))
	for _, item := range items {
		s, ok := item.(string)
		if !ok {
			return nil, fmt.Errorf("instances: expected a name, got %T", item)
		}
		out = append(out, s)
	}
	return out, nil
}
