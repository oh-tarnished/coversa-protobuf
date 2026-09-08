// Copyright 2026 The Protobuf Project authors.
// SPDX-License-Identifier: Apache-2.0

package main

// service.go emits each package's messages.proto and service.proto.
//
// Two shapes, chosen by the source model rather than by preference:
//
//   - A branch Vehicle holds once -- one cabin, one powertrain, one chassis --
//     is a **singleton**, AIP-156. Its name has no id segment, and it has no
//     Create, Delete or List: it exists because its vehicle does, and nothing
//     can make a second one or remove the only one.
//   - A branch Vehicle holds many of -- occupants, control units -- is an
//     ordinary collection with the full AIP-131..135 set plus AIP-164
//     undelete.
//
// Rule 8 says every resource gets full CRUD, and a singleton getting four
// methods rather than six is not an exception to it: AIP-156 defines the
// complete method set for a singleton, and Create and Delete are not in it.

import (
	"fmt"
	"strings"
)

// renderMessages writes messages.proto for a package.
func (m *Model) renderMessages(pkg *Package) string {
	res := messageName(pkg.Root.Name)
	sb := &strings.Builder{}

	sb.WriteString(fileHeader(m.Spec, pkg, "MessagesProto", []string{
		"buf/validate/validate.proto",
		"google/api/field_behavior.proto",
		"google/api/resource.proto",
		"google/protobuf/field_mask.proto",
		pkg.ImportPath(typeFile(pkg.Root.Name)),
	}))

	sb.WriteString(docBlock("Request message for "+pkg.ServiceName()+".Get"+res+
		", AIP-131 <https://aip.dev/131>.", ""))
	sb.WriteString("message Get" + res + "Request {\n")
	sb.WriteString(docBlock("Name of the "+res+" to retrieve, \""+pkg.Pattern+"\".", "  "))
	sb.WriteString("  string name = 1 [\n")
	sb.WriteString("    (google.api.field_behavior) = REQUIRED,\n")
	sb.WriteString("    (google.api.resource_reference) = {type: \"vdm.covesa.org/" + res + "\"}\n")
	sb.WriteString("  ];\n}\n\n")

	if !pkg.Singleton() {
		sb.WriteString(docBlock("Request message for "+pkg.ServiceName()+".List"+pkg.PluralType()+
			", AIP-132 <https://aip.dev/132>.", ""))
		sb.WriteString("message List" + pkg.PluralType() + "Request {\n")
		n := 0
		if !pkg.TopLevel() {
			n++
			sb.WriteString(docBlock("The "+parentName(pkg)+" whose "+pkg.Plural+" to list, \""+pkg.NameParent.Pattern+"\".", "  "))
			sb.WriteString("  string parent = 1 [\n")
			sb.WriteString("    (google.api.field_behavior) = REQUIRED,\n")
			sb.WriteString("    (google.api.resource_reference) = {child_type: \"vdm.covesa.org/" + res + "\"}\n")
			sb.WriteString("  ];\n\n")
		}
		sb.WriteString(docBlock("Maximum "+pkg.Plural+" to return. Defaults to 50, capped at 1000.", "  "))
		sb.WriteString(fmt.Sprintf("  int32 page_size = %d [\n    (google.api.field_behavior) = OPTIONAL,\n", n+1))
		sb.WriteString("    (buf.validate.field).int32 = {\n      gte: 0\n      lte: 1000\n    }\n  ];\n\n")
		sb.WriteString(docBlock("Page token from a previous List"+pkg.PluralType()+"Response.next_page_token.", "  "))
		sb.WriteString(fmt.Sprintf("  string page_token = %d [(google.api.field_behavior) = OPTIONAL];\n\n", n+2))
		sb.WriteString(docBlock("Filter expression, AIP-160 <https://aip.dev/160>.", "  "))
		sb.WriteString(fmt.Sprintf("  string filter = %d [(google.api.field_behavior) = OPTIONAL];\n\n", n+3))
		sb.WriteString(docBlock("Include soft-deleted "+pkg.Plural+". AIP-164 <https://aip.dev/164>.", "  "))
		sb.WriteString(fmt.Sprintf("  bool show_deleted = %d [(google.api.field_behavior) = OPTIONAL];\n}\n\n", n+4))

		sb.WriteString(docBlock("Response message for "+pkg.ServiceName()+".List"+pkg.PluralType()+
			", AIP-132 <https://aip.dev/132>.", ""))
		sb.WriteString("message List" + pkg.PluralType() + "Response {\n")
		sb.WriteString(docBlock("The "+pkg.Plural+" on this page.", "  "))
		sb.WriteString("  repeated " + res + " " + snake(pkg.Plural) + " = 1;\n\n")
		sb.WriteString(docBlock("Token for the next page, empty when this is the last page.", "  "))
		sb.WriteString("  string next_page_token = 2;\n}\n\n")

		sb.WriteString(docBlock("Request message for "+pkg.ServiceName()+".Create"+res+
			", AIP-133 <https://aip.dev/133>.", ""))
		sb.WriteString("message Create" + res + "Request {\n")
		c := 0
		if !pkg.TopLevel() {
			c++
			sb.WriteString(docBlock("The "+parentName(pkg)+" to create the "+res+" under, \""+pkg.NameParent.Pattern+"\".", "  "))
			sb.WriteString("  string parent = 1 [\n    (google.api.field_behavior) = REQUIRED,\n")
			sb.WriteString("    (google.api.resource_reference) = {child_type: \"vdm.covesa.org/" + res + "\"}\n  ];\n\n")
		}
		sb.WriteString(docBlock("The "+res+" to create.", "  "))
		sb.WriteString(fmt.Sprintf("  %s %s = %d [\n", res, snake(pkg.Root.Name), c+1))
		sb.WriteString("    (google.api.field_behavior) = REQUIRED,\n    (buf.validate.field).required = true\n  ];\n\n")
		sb.WriteString(docBlock("Client-assigned id, becoming the last segment of the resource name. "+
			"The server generates one when empty.", "  "))
		sb.WriteString(fmt.Sprintf("  string %s_id = %d [(google.api.field_behavior) = OPTIONAL];\n}\n\n", snake(pkg.Root.Name), c+2))
	}

	sb.WriteString(docBlock("Request message for "+pkg.ServiceName()+".Update"+res+
		", AIP-134 <https://aip.dev/134>.", ""))
	sb.WriteString("message Update" + res + "Request {\n")
	sb.WriteString(docBlock("The "+res+" to update. Its name field identifies the resource.", "  "))
	sb.WriteString("  " + res + " " + snake(pkg.Root.Name) + " = 1 [\n")
	sb.WriteString("    (google.api.field_behavior) = REQUIRED,\n    (buf.validate.field).required = true\n  ];\n\n")
	sb.WriteString(docBlock("Fields to update. An empty mask updates all mutable fields.\n\n"+
		"Only an actuator is writable. A sensor or attribute named in the mask is "+
		"rejected: both are OUTPUT_ONLY, because a vehicle reports them and an API "+
		"cannot set them.", "  "))
	sb.WriteString("  google.protobuf.FieldMask update_mask = 2 [(google.api.field_behavior) = OPTIONAL];\n}\n\n")

	if !pkg.Singleton() {
		sb.WriteString(docBlock("Request message for "+pkg.ServiceName()+".Delete"+res+
			", AIP-135 <https://aip.dev/135>.", ""))
		sb.WriteString("message Delete" + res + "Request {\n")
		sb.WriteString(docBlock("Name of the "+res+" to delete, \""+pkg.Pattern+"\".", "  "))
		sb.WriteString("  string name = 1 [\n    (google.api.field_behavior) = REQUIRED,\n")
		sb.WriteString("    (google.api.resource_reference) = {type: \"vdm.covesa.org/" + res + "\"}\n  ];\n}\n\n")

		sb.WriteString(docBlock("Request message for "+pkg.ServiceName()+".Undelete"+res+
			", AIP-164 <https://aip.dev/164>.", ""))
		sb.WriteString("message Undelete" + res + "Request {\n")
		sb.WriteString(docBlock("Name of the soft-deleted "+res+" to restore, \""+pkg.Pattern+"\".", "  "))
		sb.WriteString("  string name = 1 [\n    (google.api.field_behavior) = REQUIRED,\n")
		sb.WriteString("    (google.api.resource_reference).type = \"vdm.covesa.org/" + res + "\",\n")
		sb.WriteString("    (buf.validate.field).string.pattern = \"" + patternToRegex(pkg.Pattern) + "\"\n  ];\n}\n")
	}
	// Exactly one trailing newline, whichever branch above ran last.
	return strings.TrimRight(sb.String(), "\n") + "\n"
}

// fileHeader renders the licence, syntax, package, imports and Java options.
func fileHeader(spec Spec, pkg *Package, class string, imports []string) string {
	sb := &strings.Builder{}
	sb.WriteString(spec.banner())
	sb.WriteString("syntax = \"proto3\";\n\n")
	sb.WriteString("package " + protoPackage(pkg) + ";\n\n")
	for _, i := range imports {
		sb.WriteString("import \"" + i + "\";\n")
	}
	sb.WriteString("\noption java_multiple_files = true;\n")
	sb.WriteString(fmt.Sprintf("option java_outer_classname = %q;\n", class))
	sb.WriteString("option java_package = \"" + pkg.JavaPackage() + "\";\n\n")
	return sb.String()
}

// parentName is the human-readable name of a package's parent resource.
func parentName(p *Package) string {
	if p.NameParent == nil {
		return ""
	}
	return messageName(p.NameParent.Root.Name)
}
