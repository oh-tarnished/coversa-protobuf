// Copyright 2026 The Protobuf Project authors.
// SPDX-License-Identifier: Apache-2.0

package main

// domains.go is the one hand-maintained classification in this generator.
//
// The source model states the vehicle tree but never says what any part of it
// is *for*, so the functional grouping cannot be derived and has to be
// asserted. A branch the table does not name lands in "platform" and is
// reported by -survey, so a VSS release adding one is visible rather than
// silently miscategorised.

// domains groups the vehicle tree's top-level branches into the functional
// areas a reader navigates by. Every deeper resource inherits its ancestor's
// domain, so only the top level is listed.
//
// A hand-maintained table, and the only one in this generator: the source
// model states the tree but not what any part of it is *for*, so the grouping
// cannot be derived from it. A branch the table does not name lands in
// "platform" and is reported by `-survey`, so a VSS release adding one is
// visible rather than silently miscategorised.
var domains = map[string]string{
	// How the vehicle moves, and what carries it.
	"Acceleration":     "motion",
	"AngularVelocity":  "motion",
	"MotionManagement": "motion",
	"Chassis":          "motion",

	// What moves it, and what stores the energy.
	"Powertrain":        "propulsion",
	"LowVoltageBattery": "propulsion",

	// The space people occupy, and the people in it.
	"Cabin":    "interior",
	"Occupant": "interior",
	"Driver":   "interior",

	// The shell, and what hangs off it.
	"Body":     "exterior",
	"Exterior": "exterior",
	"Trailer":  "exterior",

	// Driver assistance and automated control.
	"ADAS": "assistance",

	// Where the vehicle is.
	"CurrentLocation": "location",

	// The vehicle as a managed system: what it is, what it runs, how it is
	// reached, how it is serviced.
	"ControlUnit":           "platform",
	"Connectivity":          "platform",
	"Diagnostics":           "platform",
	"Service":               "platform",
	"VersionVSS":            "platform",
	"VehicleIdentification": "platform",
}

// defaultDomain receives any branch the table does not name.
const defaultDomain = "platform"

// sharedValueTypes are value objects that more than one package references,
// and that are therefore *copied into* each rather than claimed by the first.
//
// The alternative was google.type: AIP-215 <https://aip.dev/215> forbids a
// field from naming a message in another proto package and exempts only
// `google.*`, which makes google.type.PostalAddress and google.type.Date the
// obvious homes for these two. Both were used, and both were withdrawn --
// see docs/decisions.md.
//
// The reason is the serialization layer. protoc-gen-buffers maps only
// google.protobuf.Timestamp and Duration; any other google.* message is
// **silently dropped** from the emitted schema, or emitted as an include of a
// file it never writes. A Person carrying a PostalAddress lost its address
// entirely in the .fbs, with no diagnostic. A schema that quietly means
// something different in one of its own target formats is worse than a
// duplicated message, so the duplication is accepted and the copies are
// generated rather than maintained.
var sharedValueTypes = map[string]bool{
	"Address": true,
	"Date":    true,
}

// skipTypes are source types with no protobuf counterpart.
var skipTypes = map[string]bool{
	// The GraphQL entry point. It declares no data, only where a query starts.
	"Query": true,
}

// vdmRoots are the resources VDM declares outside the vehicle tree, in
// declaration order. Each becomes a package of its own.
var vdmRoots = []string{"Person", "ChargingStation", "ChargingSession"}
