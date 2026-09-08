// Copyright 2026 The Protobuf Project authors.
// SPDX-License-Identifier: Apache-2.0

package main

// paths.go answers where a package's files live and what they are called:
// the import prefix, the proto package, the Java package, and the file one
// type lands in.
import "strings"

// covesaRoot is the import prefix every emitted file lives under, matching the
// buf module rooted at the repository root.
const covesaRoot = "protobuf/covesa"

// javaBase is the Java package prefix, per AIP-191.
const javaBase = "io.github.theprotobufproject.protobuf.covesa"

// vocabName is the directory and package segment the VSS annotation
// vocabulary lives under.
//
// Named, rather than sitting directly at protobuf/covesa/vss/v1. Two reasons,
// and the second is not cosmetic:
//
//   - Every other package here is <area>/<name>/v1, so a bare version
//     directory beside the domain directories reads as a mistake.
//   - "protobuf.covesa.vss.v1" would be a proper *prefix* of
//     "protobuf.covesa.vss.interior.cabin.v1". A package that prefixes other
//     packages is a resolution hazard: a reference to `vss.v1.Unit` from
//     inside one of them can bind to the wrong scope, and the generated code
//     in several languages nests the packages as though one contained the
//     other.
const vocabName = "annotations"

// vocabRoot is where the VSS annotation vocabulary lives. It is imported by
// every VSS package and by none of the VDM ones, which carry no signals.
const vocabRoot = covesaRoot + "/vss/" + vocabName + "/v1"

// vocabPackage is the vocabulary's proto package, used to qualify every
// annotation this generator writes.
const vocabPackage = "protobuf.covesa.vss." + vocabName + ".v1"

// typeFile is the file one object type lives in.
//
// `messages.proto` and `service.proto` are reserved for the method surface, so
// a branch whose own name collides with one of them -- VSS has a branch called
// Service, for maintenance scheduling -- gets a `_resource` suffix instead.
// Renaming the message rather than the file was the alternative and is worse:
// the message name is the mapping into every other IDL this schema generates.
func typeFile(name string) string {
	base := snake(name)
	if base == "service" || base == "messages" {
		base += "_resource"
	}
	return base + ".proto"
}

// protoPackage is the full proto package name for a VSS package.
func protoPackage(pkg *Package) string {
	return "protobuf.covesa." + strings.Join(pkg.segments(), ".") + ".v1"
}

// outerClassname renders the AIP-191 java_outer_classname for a file.
func outerClassname(base string) string {
	return pascal(strings.TrimSuffix(base, ".proto")) + "Proto"
}
