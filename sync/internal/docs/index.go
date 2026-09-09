// Copyright 2026 The Protobuf Project authors.
// SPDX-License-Identifier: Apache-2.0

package docs

// index.go writes the root README: every resource, grouped by the functional
// domain it belongs to, with its name pattern and API shape.

import (
	"fmt"
	"sort"
	"strings"

	"github.com/the-protobuf-project/vdm/sync/model"
)

// index renders the top-level reference index.
func (g *Generator) index() string {
	var sb strings.Builder

	sb.WriteString("# COVESA protobuf reference\n\n")
	fmt.Fprintf(&sb, "Generated from the COVESA Vehicle Data Model, spec revision `%s` (`%s`).\n\n",
		g.M.Spec.VSS.Version, g.M.Spec.VSS.Short())
	sb.WriteString(g.banner())

	counts := g.counts()
	sb.WriteString("| | |\n| --- | --- |\n")
	fmt.Fprintf(&sb, "| Packages | %d |\n", len(g.M.Packages))
	fmt.Fprintf(&sb, "| Resources | %d |\n", len(g.M.Packages))
	fmt.Fprintf(&sb, "| Collections / singletons | %d / %d |\n", counts.collections, counts.singletons)
	fmt.Fprintf(&sb, "| RPCs | %d |\n\n", counts.rpcs)

	g.relationships(&sb)
	g.dependencies(&sb)

	for _, group := range g.byDomain() {
		fmt.Fprintf(&sb, "## %s\n\n", group.title)
		sb.WriteString("| Resource | Name | Shape | Reference |\n")
		sb.WriteString("| --- | --- | --- | --- |\n")
		for _, pkg := range group.packages {
			shape := "collection"
			if pkg.Singleton() {
				shape = "singleton"
			}
			fmt.Fprintf(&sb, "| `%s` | `%s` | %s | [%s](%s/README.md) |\n",
				pkg.ResourceName(), pkg.Pattern, shape,
				pkg.Name, strings.Join(pkg.Segments(), "/"))
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

// tally is the summary shown at the head of the index.
type tally struct{ collections, singletons, rpcs int }

// counts totals the resources by shape and the RPCs they add up to.
func (g *Generator) counts() tally {
	var t tally
	for _, pkg := range g.M.Packages {
		if pkg.Singleton() {
			t.singletons++
			t.rpcs += 2 // Get, Update
			continue
		}
		t.collections++
		t.rpcs += 6 // Get, List, Create, Update, Delete, Undelete
	}
	return t
}

// group is one domain's packages, in the order the index lists them.
type group struct {
	title    string
	packages []*model.Package
}

// byDomain groups the packages for the index.
//
// The vehicle root comes first, then the VSS domains alphabetically, then the
// VDM resources -- which is the order a reader meets them: the vehicle, its
// parts, and then the world it interacts with.
func (g *Generator) byDomain() []group {
	byKey := map[string][]*model.Package{}
	for _, pkg := range g.M.Packages {
		byKey[groupKey(pkg)] = append(byKey[groupKey(pkg)], pkg)
	}

	keys := make([]string, 0, len(byKey))
	for k := range byKey {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	groups := make([]group, 0, len(keys))
	for _, k := range keys {
		groups = append(groups, group{title: titleOf(k), packages: byKey[k]})
	}
	sort.SliceStable(groups, func(i, j int) bool {
		return rank(groups[i].title) < rank(groups[j].title)
	})
	return groups
}

// groupKey is the domain a package is listed under.
func groupKey(pkg *model.Package) string {
	if pkg.Family == model.FamilyVDM {
		return "vdm"
	}
	if pkg.Domain == "" {
		return "vehicle"
	}
	return pkg.Domain
}

// titleOf renders a group key as a heading.
func titleOf(key string) string {
	switch key {
	case "vehicle":
		return "The vehicle"
	case "vdm":
		return "Beyond the vehicle"
	default:
		return strings.ToUpper(key[:1]) + key[1:]
	}
}

// rank orders the groups: the vehicle first, the VDM resources last.
func rank(title string) int {
	switch title {
	case "The vehicle":
		return 0
	case "Beyond the vehicle":
		return 2
	default:
		return 1
	}
}
