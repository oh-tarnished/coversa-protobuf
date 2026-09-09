// Copyright 2026 Srikanth Kandarpa.
// SPDX-License-Identifier: Apache-2.0

package emit

// messages.go writes each package's messages.proto: the request and response
// types for its standard methods.
//
// Which messages exist follows the resource's shape. A singleton has no
// Create, Delete, List or Undelete -- AIP-156 does not define them for a
// resource that exists because its parent does.

import (
	"fmt"
	"strings"

	"github.com/oh-tarnished/coversa-protobuf/sync/model"
	"github.com/oh-tarnished/coversa-protobuf/sync/naming"
)

// Messages renders messages.proto for a package.
func (e *Emitter) Messages(pkg *model.Package) string {
	var sb strings.Builder
	sb.WriteString(e.fileHeader(pkg, "MessagesProto", []string{
		"buf/validate/validate.proto",
		"google/api/field_behavior.proto",
		"google/api/resource.proto",
		"google/protobuf/field_mask.proto",
		pkg.ImportPath(model.TypeFile(pkg.Root.Name)),
	}))

	e.getRequest(pkg, &sb)
	if !pkg.Singleton() {
		e.listMessages(pkg, &sb)
		e.createRequest(pkg, &sb)
	}
	e.updateRequest(pkg, &sb)
	if !pkg.Singleton() {
		e.deleteRequests(pkg, &sb)
	}
	// Exactly one trailing newline, whichever branch above ran last.
	return strings.TrimRight(sb.String(), "\n") + "\n"
}

// getRequest writes GetXRequest, AIP-131.
func (e *Emitter) getRequest(pkg *model.Package, sb *strings.Builder) {
	res := pkg.ResourceName()
	sb.WriteString(docBlock("Request message for "+pkg.ServiceName()+".Get"+res+
		", AIP-131 <https://aip.dev/131>.", ""))
	sb.WriteString("message Get" + res + "Request {\n")
	sb.WriteString(docBlock("Name of the "+res+" to retrieve, \""+pkg.Pattern+"\".", "  "))
	sb.WriteString("  string name = 1 [\n")
	sb.WriteString("    (google.api.field_behavior) = REQUIRED,\n")
	sb.WriteString("    (google.api.resource_reference) = {type: \"vdm.covesa.org/" + res + "\"}\n")
	sb.WriteString("  ];\n}\n\n")
}

// listMessages writes ListXRequest and ListXResponse, AIP-132.
func (e *Emitter) listMessages(pkg *model.Package, sb *strings.Builder) {
	res, plural := pkg.ResourceName(), pkg.PluralType()

	sb.WriteString(docBlock("Request message for "+pkg.ServiceName()+".List"+plural+
		", AIP-132 <https://aip.dev/132>.", ""))
	sb.WriteString("message List" + plural + "Request {\n")

	n := 0
	if !pkg.TopLevel() {
		n = 1
		e.parentField(pkg, res, sb)
	}
	sb.WriteString(docBlock("Maximum "+pkg.Plural+" to return. Defaults to 50, capped at 1000.", "  "))
	fmt.Fprintf(sb, "  int32 page_size = %d [\n    (google.api.field_behavior) = OPTIONAL,\n", n+1)
	sb.WriteString("    (buf.validate.field).int32 = {\n      gte: 0\n      lte: 1000\n    }\n  ];\n\n")
	sb.WriteString(docBlock("Page token from a previous List"+plural+"Response.next_page_token.", "  "))
	fmt.Fprintf(sb, "  string page_token = %d [(google.api.field_behavior) = OPTIONAL];\n\n", n+2)
	sb.WriteString(docBlock("Filter expression, AIP-160 <https://aip.dev/160>.", "  "))
	fmt.Fprintf(sb, "  string filter = %d [(google.api.field_behavior) = OPTIONAL];\n\n", n+3)
	sb.WriteString(docBlock("Include soft-deleted "+pkg.Plural+". AIP-164 <https://aip.dev/164>.", "  "))
	fmt.Fprintf(sb, "  bool show_deleted = %d [(google.api.field_behavior) = OPTIONAL];\n}\n\n", n+4)

	sb.WriteString(docBlock("Response message for "+pkg.ServiceName()+".List"+plural+
		", AIP-132 <https://aip.dev/132>.", ""))
	sb.WriteString("message List" + plural + "Response {\n")
	sb.WriteString(docBlock("The "+pkg.Plural+" on this page.", "  "))
	sb.WriteString("  repeated " + res + " " + naming.Snake(pkg.Plural) + " = 1;\n\n")
	sb.WriteString(docBlock("Token for the next page, empty when this is the last page.", "  "))
	sb.WriteString("  string next_page_token = 2;\n}\n\n")
}

// createRequest writes CreateXRequest, AIP-133.
func (e *Emitter) createRequest(pkg *model.Package, sb *strings.Builder) {
	res, lower := pkg.ResourceName(), naming.Snake(pkg.Root.Name)

	sb.WriteString(docBlock("Request message for "+pkg.ServiceName()+".Create"+res+
		", AIP-133 <https://aip.dev/133>.", ""))
	sb.WriteString("message Create" + res + "Request {\n")

	n := 0
	if !pkg.TopLevel() {
		n = 1
		sb.WriteString(docBlock("The "+pkg.ParentName()+" to create the "+res+" under, \""+
			pkg.NameParent.Pattern+"\".", "  "))
		sb.WriteString("  string parent = 1 [\n    (google.api.field_behavior) = REQUIRED,\n")
		sb.WriteString("    (google.api.resource_reference) = {child_type: \"vdm.covesa.org/" + res + "\"}\n  ];\n\n")
	}
	sb.WriteString(docBlock("The "+res+" to create.", "  "))
	fmt.Fprintf(sb, "  %s %s = %d [\n", res, lower, n+1)
	sb.WriteString("    (google.api.field_behavior) = REQUIRED,\n    (buf.validate.field).required = true\n  ];\n\n")
	sb.WriteString(docBlock("Client-assigned id, becoming the last segment of the resource name. "+
		"The server generates one when empty.", "  "))
	fmt.Fprintf(sb, "  string %s_id = %d [(google.api.field_behavior) = OPTIONAL];\n}\n\n", lower, n+2)
}

// parentField writes the `parent` field a child collection's requests carry.
func (e *Emitter) parentField(pkg *model.Package, res string, sb *strings.Builder) {
	sb.WriteString(docBlock("The "+pkg.ParentName()+" whose "+pkg.Plural+" to list, \""+
		pkg.NameParent.Pattern+"\".", "  "))
	sb.WriteString("  string parent = 1 [\n")
	sb.WriteString("    (google.api.field_behavior) = REQUIRED,\n")
	sb.WriteString("    (google.api.resource_reference) = {child_type: \"vdm.covesa.org/" + res + "\"}\n")
	sb.WriteString("  ];\n\n")
}

// updateRequest writes UpdateXRequest, AIP-134.
func (e *Emitter) updateRequest(pkg *model.Package, sb *strings.Builder) {
	res, lower := pkg.ResourceName(), naming.Snake(pkg.Root.Name)

	sb.WriteString(docBlock("Request message for "+pkg.ServiceName()+".Update"+res+
		", AIP-134 <https://aip.dev/134>.", ""))
	sb.WriteString("message Update" + res + "Request {\n")
	sb.WriteString(docBlock("The "+res+" to update. Its name field identifies the resource.", "  "))
	sb.WriteString("  " + res + " " + lower + " = 1 [\n")
	sb.WriteString("    (google.api.field_behavior) = REQUIRED,\n    (buf.validate.field).required = true\n  ];\n\n")
	sb.WriteString(docBlock("Fields to update. An empty mask updates all mutable fields.\n\n"+
		"Only an actuator is writable. A sensor or attribute named in the mask is "+
		"rejected: both are OUTPUT_ONLY, because a vehicle reports them and an API "+
		"cannot set them.", "  "))
	sb.WriteString("  google.protobuf.FieldMask update_mask = 2 [(google.api.field_behavior) = OPTIONAL];\n}\n\n")
}

// deleteRequests writes DeleteXRequest and UndeleteXRequest, AIP-135 and 164.
func (e *Emitter) deleteRequests(pkg *model.Package, sb *strings.Builder) {
	res := pkg.ResourceName()

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
	sb.WriteString("    (buf.validate.field).string.pattern = \"" + model.PatternRegex(pkg.Pattern) + "\"\n  ];\n}\n")
}

// fileHeader renders the banner, package clause, imports and Java options for
// a file the emitter builds rather than plans.
func (e *Emitter) fileHeader(pkg *model.Package, class string, imports []string) string {
	var sb strings.Builder
	sb.WriteString(e.M.Spec.Banner())
	sb.WriteString("syntax = \"proto3\";\n\n")
	sb.WriteString("package " + pkg.ProtoPackage() + ";\n\n")
	for _, i := range imports {
		sb.WriteString("import \"" + i + "\";\n")
	}
	sb.WriteString("\noption java_multiple_files = true;\n")
	fmt.Fprintf(&sb, "option java_outer_classname = %q;\n", class)
	sb.WriteString("option java_package = \"" + pkg.JavaPackage() + "\";\n\n")
	return sb.String()
}
