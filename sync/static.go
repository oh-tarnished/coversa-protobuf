// Copyright 2026 The Protobuf Project authors.
// SPDX-License-Identifier: Apache-2.0

package main

// static.go holds the two vocabulary files that are not transcriptions of
// the source model. They live here rather than as checked-in .proto files so
// that nothing under protobuf/ is maintained by hand.
//
// Functions rather than constants because each needs the spec revision for
// its banner, which is read at run time.

func vocabOptions(spec Spec) string {
	return vocabHeader(spec, "OptionsProto", []string{
		vocabRoot + "/quantity_kind.proto",
		vocabRoot + "/unit.proto",
	}) + `// BranchOptions is the payload of the ` + "`" + `branch` + "`" + ` message option.
//
// A VSS branch is a grouping node whose leaves are read and written
// independently; a struct is one whose members are read and written together.
// The distinction is not cosmetic -- it is what tells a transport whether it
// may deliver half of this message -- so it is recorded rather than inferred
// from shape.
//
// Reference: COVESA VSS rule set, "Signal branches".
// https://covesa.github.io/vehicle_signal_specification/rule_set/branches/
message BranchOptions {
  // Which kind of VSS node this message is, ELEMENT_BRANCH or ELEMENT_STRUCT.
  Element element = 1;

  // Fully qualified name in the source model, e.g. "Vehicle.Cabin.Door".
  //
  // This is the identity VSS itself uses, and it is stable across releases
  // where a protobuf message name is only stable across ours. A consumer
  // correlating this schema with a VSS-native datastore joins on this string.
  string fqn = 2;
}

// SignalOptions is the payload of the ` + "`" + `signal` + "`" + ` field option.
//
// Reference: COVESA VSS rule set, "Data entry types".
// https://covesa.github.io/vehicle_signal_specification/rule_set/data_entry/
message SignalOptions {
  // Whether the value is static, readable or also writable.
  Element element = 1;

  // Fully qualified name in the source model, e.g. "Vehicle.Speed".
  string fqn = 2;

  // The unit this field's value is expressed in.
  //
  // Fixed, not negotiated. The source model exposes a unit as a GraphQL field
  // argument so a caller may ask for miles instead of kilometres; protobuf has
  // no field arguments, and putting the unit on the wire beside the value
  // would let two producers of the same field disagree. So the schema pins the
  // source model's default and names it here.
  Unit unit = 3;

  // The physical quantity ` + "`" + `unit` + "`" + ` measures, e.g. QUANTITY_KIND_VELOCITY.
  //
  // Derivable from ` + "`" + `unit` + "`" + ` alone, and recorded anyway: a generator converting
  // between units needs to know which conversions are even meaningful, and
  // recomputing that from a table it does not have is how a length gets
  // converted into a mass.
  QuantityKind quantity_kind = 4;
}

// EnumerationOptions is the payload of the ` + "`" + `enumeration` + "`" + ` enum option.
//
// VSS states a signal's allowed values inline on the signal. Protobuf needs a
// named top-level enum, so one is synthesised per constrained signal and this
// records which signal it came from.
message EnumerationOptions {
  // Which kind of VSS node declared these allowed values.
  Element element = 1;

  // Fully qualified name of the signal that declares them.
  string fqn = 2;
}

// AllowedValueOptions is the payload of the ` + "`" + `allowed_value` + "`" + ` enum value option.
message AllowedValueOptions {
  // The value's spelling in the source model, before sanitisation.
  //
  // VSS allows values that no protobuf identifier can carry -- "Row1" is
  // merely cased wrongly, but values with spaces, hyphens and leading digits
  // all occur. The sanitised name is what the schema uses; this is what the
  // wire format of a VSS-native peer will actually contain, and a codec needs
  // both to translate between them.
  //
  // The source model calls this ` + "`" + `originalName` + "`" + `. AIP-122 <https://aip.dev/122>
  // bans the ` + "`" + `_name` + "`" + ` suffix on any field but display_name, given_name and
  // family_name, so that term survives only in this comment.
  string source_spelling = 1;
}

// Element is the kind of node an item maps to in the Vspec language.
//
// Reference: COVESA VSS rule set.
// https://covesa.github.io/vehicle_signal_specification/rule_set/
enum Element {
  // Not specified.
  ELEMENT_UNSPECIFIED = 0;

  // A grouping node whose properties are read and written independently.
  ELEMENT_BRANCH = 1;

  // A property whose value is static for the life of the vehicle.
  ELEMENT_ATTRIBUTE = 2;

  // A property whose value is dynamic and readable.
  ELEMENT_SENSOR = 3;

  // A property whose value is dynamic, readable and writable.
  ELEMENT_ACTUATOR = 4;

  // A grouping node whose properties are read and written in one operation.
  ELEMENT_STRUCT = 5;

  // A property declared inside a struct.
  ELEMENT_STRUCT_PROPERTY = 6;

  // A quantity kind grouping units of the same physical quantity.
  ELEMENT_QUANTITY_KIND = 7;

  // A unit of measurement.
  ELEMENT_UNIT = 8;
}
`
}

func vocabAnnotations(spec Spec) string {
	return vocabHeader(spec, "AnnotationsProto", []string{
		"google/protobuf/descriptor.proto",
		vocabRoot + "/options.proto",
	}) + `// annotations.proto is the single import a VSS package needs from this
// vocabulary:
//
//	import "protobuf/covesa/vss/annotations/v1/annotations.proto";
//
// It carries the part of a VSS signal that protobuf has no native place for.
// A ` + "`" + `.proto` + "`" + ` field says a vehicle's speed is a ` + "`" + `double` + "`" + `; it cannot say that the
// number is kilometres per hour, that the value is a *sensor* rather than a
// static attribute, or that COVESA calls it ` + "`" + `Vehicle.Speed` + "`" + `. All three are
// load-bearing -- a consumer that reads the number without the unit has read a
// different quantity -- so they are recorded here in a form a generator can
// read, and restated in each field's doc comment for a human.
//
// Reference: COVESA Vehicle Signal Specification, Vspec rule set.
// https://covesa.github.io/vehicle_signal_specification/rule_set/
//
// # Why a custom option and not a comment alone
//
// The upstream model is GraphQL SDL, where this information lives in a
// ` + "`" + `@vspec` + "`" + ` directive and a ` + "`" + `unit:` + "`" + ` field argument. Protobuf has neither
// directives nor field arguments, and a comment is not readable by the
// FlatBuffers and Cap'n Proto generators that consume this schema. Dropping to
// comments alone would make the unit unrecoverable downstream, which is the
// one thing the source model is careful never to do.
//
// See docs/decisions.md for why this departs from protobuf-rfc's rule 5.
//
// # Extension number range
//
// Field numbers 50000-99999 are reserved for non-Google options. This
// vocabulary uses 54000-54003, clear of buffers.v1's 53000-53007, so a file
// may carry both -- which every VSS package here does.

extend google.protobuf.MessageOptions {
  // branch marks a message as a VSS branch or struct and records its fully
  // qualified name in the source model.
  BranchOptions branch = 54000;
}

extend google.protobuf.FieldOptions {
  // signal records what a field is in VSS terms: attribute, sensor or
  // actuator, the unit its value is expressed in, and its fully qualified
  // name.
  SignalOptions signal = 54001;
}

extend google.protobuf.EnumOptions {
  // enumeration records the fully qualified name of the VSS signal whose
  // allowed values an enum encodes.
  EnumerationOptions enumeration = 54002;
}

extend google.protobuf.EnumValueOptions {
  // allowed_value records an enum value's spelling in the source model, which
  // is frequently not a legal protobuf identifier.
  AllowedValueOptions allowed_value = 54003;
}
`
}
