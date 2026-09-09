// Copyright 2026 The Protobuf Project authors.
// SPDX-License-Identifier: Apache-2.0

package vdm

import (
	"testing"

	"github.com/oh-tarnished/coversa-protobuf/sync/sdl"
	"github.com/oh-tarnished/coversa-protobuf/sync/vspec"
)

const source = `
type Person {
  id: ID!
  name: PersonName
  ageRange: AgeRange
  homeAddress: Address
}
type PersonName { first: String!  last: String! }
type Address { street: String  city: String }
enum AgeRange { CHILD  ADULT }

type ChargingStation { chargingPoints: [ChargingPoint] }
type ChargingPoint { label: String! }
type ChargingSession {
  vehicle: Vehicle!
  customer: Person!
  chargingPoint: ChargingPoint!
  startTime: String!
}
type Vehicle { speed: Float }
`

func convertAll(t *testing.T) map[string]*vspec.Node {
	t.Helper()
	defs, err := sdl.Parse("test.graphql", source)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	roots, err := Convert(defs)
	if err != nil {
		t.Fatalf("Convert: %v", err)
	}
	out := map[string]*vspec.Node{}
	for _, r := range roots {
		out[r.Name] = r
	}
	return out
}

// TestInlinesValueObjects checks a field naming a plain type becomes a branch
// with that type's own fields, since vspec has no type references.
func TestInlinesValueObjects(t *testing.T) {
	person := convertAll(t)["Person"]
	if person == nil {
		t.Fatal("no Person")
	}

	name, ok := person.Child("name")
	if !ok {
		t.Fatal("no name child")
	}
	if name.Kind != vspec.KindBranch {
		t.Errorf("name kind = %s, want branch", name.Kind)
	}
	if _, ok := name.Child("first"); !ok {
		t.Error("PersonName was not inlined")
	}
	if name.FQN != "Person.Name" {
		t.Errorf("fqn = %q, want Person.Name (VSS path casing)", name.FQN)
	}
}

// TestReferencesOtherResources is the rule that keeps a session from
// swallowing a vehicle: a field naming a resource points at it.
func TestReferencesOtherResources(t *testing.T) {
	session := convertAll(t)["ChargingSession"]
	for _, name := range []string{"vehicle", "customer", "chargingPoint"} {
		child, ok := session.Child(name)
		if !ok {
			t.Fatalf("no %s child", name)
		}
		if child.Ref == "" {
			t.Errorf("%s is inlined; want a reference", name)
		}
		if child.Datatype != "string" {
			t.Errorf("%s datatype = %q, want string", name, child.Datatype)
		}
		if len(child.Children) != 0 {
			t.Errorf("%s carries %d inlined children", name, len(child.Children))
		}
	}
}

// TestOwnerStillContains checks the other half: the station that holds a
// charging point inlines it, where the session that names one does not.
func TestOwnerStillContains(t *testing.T) {
	station := convertAll(t)["ChargingStation"]
	points, ok := station.Child("chargingPoints")
	if !ok {
		t.Fatal("no chargingPoints child")
	}
	if points.Ref != "" {
		t.Error("chargingPoints is a reference; its owner must contain it")
	}
	if !points.Repeated {
		t.Error("chargingPoints is not marked repeated")
	}
	if _, ok := points.Child("label"); !ok {
		t.Error("ChargingPoint was not inlined into its owner")
	}
}

// TestEnumBecomesAllowed checks a GraphQL enum arrives as an allowed-value
// set, which is how VSS states the same thing.
func TestEnumBecomesAllowed(t *testing.T) {
	person := convertAll(t)["Person"]
	age, _ := person.Child("ageRange")
	if age.EnumName != "AgeRange" {
		t.Errorf("enum name = %q", age.EnumName)
	}
	if len(age.Allowed) != 2 || age.Allowed[0] != "CHILD" {
		t.Errorf("allowed = %v", age.Allowed)
	}
}

// TestUnknownTypeIsAnError keeps the generator's rule: an unfamiliar type is
// an error, never a skipped field.
func TestUnknownTypeIsAnError(t *testing.T) {
	defs, err := sdl.Parse("t.graphql", `
type Person { thing: Mystery }
type ChargingStation { a: String }
type ChargingSession { b: String }
`)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Convert(defs); err == nil {
		t.Fatal("Convert accepted an unknown type")
	}
}
