// Copyright 2026 Srikanth Kandarpa.
// SPDX-License-Identifier: Apache-2.0

package model

// paths.go answers where a package's files live and what they are called.

import (
	"strings"

	"github.com/oh-tarnished/coversa-protobuf/sync/naming"
)

// CovesaRoot is the import prefix every emitted file lives under, matching
// the buf module rooted at the repository root.
const CovesaRoot = "protobuf/covesa"

// javaBase is the Java package prefix, per AIP-191.
const javaBase = "io.github.ohtarnished.protobuf.covesa"

// VocabName is the directory and package segment the VSS annotation
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
const VocabName = "annotations"

// VocabRoot is where the VSS annotation vocabulary lives. It is imported by
// every VSS package and by none of the VDM ones, which carry no signals.
const VocabRoot = CovesaRoot + "/vss/" + VocabName + "/v1"

// VocabPackage is the vocabulary's proto package, used to qualify every
// annotation the generator writes.
const VocabPackage = "protobuf.covesa.vss." + VocabName + ".v1"

// Segments is the path from the repository root down to this package,
// excluding the version directory.
//
// Vehicle sits directly under its family rather than in a domain, because it
// is the root every domain hangs from rather than a member of one.
func (p *Package) Segments() []string {
	if p.Domain == "" {
		return []string{string(p.Family), p.Dir}
	}
	return []string{string(p.Family), p.Domain, p.Dir}
}

// ProtoPackage is the full proto package name, e.g.
// protobuf.covesa.vss.interior.seat.v1.
func (p *Package) ProtoPackage() string {
	return "protobuf.covesa." + strings.Join(p.Segments(), ".") + ".v1"
}

// JavaPackage is the AIP-191 java_package for this package.
func (p *Package) JavaPackage() string {
	return javaBase + "." + strings.Join(p.Segments(), ".") + ".v1"
}

// ImportPath is the import path of one file in this package.
func (p *Package) ImportPath(base string) string {
	return CovesaRoot + "/" + strings.Join(p.Segments(), "/") + "/v1/" + base
}

// TypeFile is the file one object type lives in.
//
// `messages.proto` and `service.proto` are reserved for the method surface,
// so a branch whose own name collides with one -- VSS has a branch called
// Service, for maintenance scheduling -- takes a `_resource` suffix instead.
// Renaming the message was the alternative and is worse: the message name is
// the mapping into every other IDL this schema generates.
func TypeFile(name string) string {
	base := naming.Snake(name)
	if base == "service" || base == "messages" {
		base += "_resource"
	}
	return base + ".proto"
}

// OuterClassname renders the AIP-191 java_outer_classname for a file.
func OuterClassname(base string) string {
	return naming.Pascal(strings.TrimSuffix(base, ".proto")) + "Proto"
}
