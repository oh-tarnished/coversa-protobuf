// Copyright 2026 The Protobuf Project authors.
// SPDX-License-Identifier: Apache-2.0

package main

// rpc.go renders service.proto: the RPCs themselves, with the HTTP mapping,
// method signatures and per-method error documentation AIP requires.

import "strings"

// ServiceName is the collection name the service is called by, AIP-131.
//
// Where a branch's plural equals its singular -- Adas, Diagnostics,
// Connectivity, all mass nouns or already-plural names -- the collection name
// would collide with the message name in the same package, which protobuf
// rejects. Those services take a `Service` suffix: the collection name is
// still the resource's own plural, so AIP-123's tie between plural and URL
// segment is untouched, and only the gRPC service identifier differs.
func (p *Package) ServiceName() string {
	if p.PluralType() == messageName(p.Root.Name) {
		return p.PluralType() + "Service"
	}
	return p.PluralType()
}

// PluralType renders the plural as a type-cased name, e.g. "ControlUnits".
func (p *Package) PluralType() string { return pascal(p.Plural) }

// Singleton reports whether a vehicle holds exactly one of this branch, in
// which case AIP-156 governs and there is no Create, Delete or List.
func (p *Package) Singleton() bool { return p.Parent != nil && !p.IsList }

// httpPath renders the resource's REST path from its own pattern.
//
// Derived rather than assembled from the family: a ChargingPoint hangs beneath
// a ChargingStation, not beneath a vehicle, and AIP-127 checks the HTTP
// template against the resource's declared pattern.
func (p *Package) httpPath() string {
	return "/v1/{name=" + patternToPath(p.Pattern) + "}"
}

// collectionPath renders the collection's REST path, for List and Create.
//
// A root resource's collection is addressed directly; a child's hangs beneath
// its parent, whose name becomes the `parent` variable.
func (p *Package) collectionPath() string {
	if p.TopLevel() {
		return "/v1/" + p.Plural
	}
	return "/v1/{parent=" + patternToPath(p.NameParent.Pattern) + "}/" + p.Plural
}

// TopLevel reports whether the resource has no parent resource above it.
func (p *Package) TopLevel() bool { return p.Parent == nil }

// patternToPath turns a resource name pattern into an HTTP template body:
// every `{segment}` becomes a `*` wildcard.
func patternToPath(pattern string) string {
	parts := strings.Split(pattern, "/")
	for i, s := range parts {
		if strings.HasPrefix(s, "{") {
			parts[i] = "*"
		}
	}
	return strings.Join(parts, "/")
}

// renderService writes service.proto for a package.
func (m *Model) renderService(pkg *Package) string {
	res := messageName(pkg.Root.Name)
	lower := snake(pkg.Root.Name)
	sb := &strings.Builder{}

	imports := []string{
		"google/api/annotations.proto",
		"google/api/client.proto",
		pkg.ImportPath("messages.proto"),
		pkg.ImportPath(typeFile(pkg.Root.Name)),
	}
	if !pkg.Singleton() {
		imports = append(imports, "google/protobuf/empty.proto")
	}
	sb.WriteString(fileHeader(m.Spec, pkg, "ServiceProto", sortedStrings(imports)))

	sb.WriteString(docBlock(m.serviceDoc(pkg), ""))
	sb.WriteString("service " + pkg.ServiceName() + " {\n")

	// Get
	sb.WriteString(docBlock("Retrieves one "+res+".\n\n"+
		"  GET "+pkg.httpPath()+"\n\n"+
		"Safe and idempotent. Read the `etag` here and pass it back on update "+
		"to make the write conditional.\n\n"+
		"  NOT_FOUND          no such "+res+"\n"+
		"  INVALID_ARGUMENT   name does not match the resource pattern\n"+
		"  PERMISSION_DENIED  caller may not read it", "  "))
	sb.WriteString("  rpc Get" + res + "(Get" + res + "Request) returns (" + res + ") {\n")
	sb.WriteString("    option (google.api.http) = {get: \"" + pkg.httpPath() + "\"};\n")
	sb.WriteString("    option (google.api.method_signature) = \"name\";\n  }\n\n")

	if !pkg.Singleton() {
		// List
		sb.WriteString(docBlock("Lists the "+pkg.Plural+" of one vehicle.\n\n"+
			"  GET "+pkg.collectionPath()+"\n\n"+
			"Paginated per AIP-158 <https://aip.dev/158>: pass `page_size` (defaults "+
			"to 50, capped at 1000) and the `next_page_token` from the previous "+
			"response. Page tokens are opaque and may expire; do not construct or "+
			"store them.\n\n"+
			"  INVALID_ARGUMENT   malformed filter, or page_size out of range\n"+
			"  PERMISSION_DENIED  caller may not list this collection", "  "))
		sb.WriteString("  rpc List" + pkg.PluralType() + "(List" + pkg.PluralType() +
			"Request) returns (List" + pkg.PluralType() + "Response) {\n")
		sb.WriteString("    option (google.api.http) = {get: \"" + pkg.collectionPath() + "\"};\n")
		if !pkg.TopLevel() {
			sb.WriteString("    option (google.api.method_signature) = \"parent\";\n")
		}
		sb.WriteString("  }\n\n")

		// Create
		sb.WriteString(docBlock("Creates a "+res+".\n\n"+
			"  POST "+pkg.collectionPath()+"\n"+
			"  body: the "+res+" itself\n\n"+
			"Server-assigned fields on the request body are ignored: `name`, `uid`, "+
			"`etag` and the server timestamps are set regardless of what is sent.\n\n"+
			"Not idempotent. Retrying after a timeout may create a second "+res+
			"; supply `"+lower+"_id` if the caller needs retry safety.\n\n"+
			"  ALREADY_EXISTS     that id is taken\n"+
			"  INVALID_ARGUMENT   a field violates its buf.validate constraint\n"+
			"  PERMISSION_DENIED  caller may not create here", "  "))
		sb.WriteString("  rpc Create" + res + "(Create" + res + "Request) returns (" + res + ") {\n")
		sb.WriteString("    option (google.api.http) = {\n")
		sb.WriteString("      post: \"" + pkg.collectionPath() + "\"\n")
		sb.WriteString("      body: \"" + lower + "\"\n    };\n")
		if pkg.TopLevel() {
			sb.WriteString("    option (google.api.method_signature) = \"" + lower + "," + lower + "_id\";\n")
		} else {
			sb.WriteString("    option (google.api.method_signature) = \"parent," + lower + "," + lower + "_id\";\n")
		}
		sb.WriteString("  }\n\n")
	}

	// Update
	updatePath := strings.Replace(pkg.httpPath(), "{name=", "{"+lower+".name=", 1)
	sb.WriteString(docBlock("Updates a "+res+".\n\n"+
		"  PATCH "+updatePath+"\n"+
		"  body: the "+res+" itself\n\n"+
		"A partial update: only fields named in `update_mask` are written. An "+
		"empty mask writes every mutable field, which will clear anything the "+
		"caller left unset -- pass a mask unless that is the intent.\n\n"+
		"**Only an actuator is writable.** VSS marks each signal an attribute, a "+
		"sensor or an actuator; the first two are OUTPUT_ONLY here, because a "+
		"vehicle reports them and no API call can change them. A mask naming one "+
		"is rejected rather than silently ignored.\n\n"+
		"Idempotent. Pass the `etag` read from Get to make the write conditional.\n\n"+
		"  NOT_FOUND          no such "+res+"\n"+
		"  ABORTED            etag does not match the stored resource\n"+
		"  INVALID_ARGUMENT   mask names an unknown, immutable or read-only field\n"+
		"  PERMISSION_DENIED  caller may not modify it", "  "))
	sb.WriteString("  rpc Update" + res + "(Update" + res + "Request) returns (" + res + ") {\n")
	sb.WriteString("    option (google.api.http) = {\n")
	sb.WriteString("      patch: \"" + updatePath + "\"\n")
	sb.WriteString("      body: \"" + lower + "\"\n    };\n")
	sb.WriteString("    option (google.api.method_signature) = \"" + lower + ",update_mask\";\n  }\n")

	if !pkg.Singleton() {
		// Delete
		sb.WriteString("\n")
		sb.WriteString(docBlock("Deletes a "+res+".\n\n"+
			"  DELETE "+pkg.httpPath()+"\n\n"+
			"Soft delete: the "+res+" is marked deleted, disappears from List unless "+
			"`show_deleted` is set, and is purged at `expire_time`. Recover it with "+
			"Undelete"+res+" before then.\n\n"+
			"Idempotent in effect but not in reporting: a second delete returns "+
			"NOT_FOUND rather than succeeding silently.\n\n"+
			"  NOT_FOUND          no such "+res+"\n"+
			"  ABORTED            etag does not match\n"+
			"  PERMISSION_DENIED  caller may not delete it", "  "))
		sb.WriteString("  rpc Delete" + res + "(Delete" + res + "Request) returns (google.protobuf.Empty) {\n")
		sb.WriteString("    option (google.api.http) = {delete: \"" + pkg.httpPath() + "\"};\n")
		sb.WriteString("    option (google.api.method_signature) = \"name\";\n  }\n\n")

		// Undelete
		sb.WriteString(docBlock("Restores a soft-deleted "+res+".\n\n"+
			"  POST "+pkg.httpPath()+":undelete\n\n"+
			"Valid only before `expire_time`; after that the "+res+" is gone and "+
			"this returns NOT_FOUND. Clears `delete_time` and `expire_time`.\n\n"+
			"  NOT_FOUND          no such "+res+", or already purged\n"+
			"  ALREADY_EXISTS     the "+res+" is not deleted\n"+
			"  PERMISSION_DENIED  caller may not restore it", "  "))
		sb.WriteString("  rpc Undelete" + res + "(Undelete" + res + "Request) returns (" + res + ") {\n")
		sb.WriteString("    option (google.api.http) = {\n")
		sb.WriteString("      post: \"" + pkg.httpPath() + ":undelete\"\n")
		sb.WriteString("      body: \"*\"\n    };\n")
		sb.WriteString("    option (google.api.method_signature) = \"name\";\n  }\n")
	}
	sb.WriteString("}\n")
	return sb.String()
}

// serviceDoc builds the service's documentation block.
func (m *Model) serviceDoc(pkg *Package) string {
	res := messageName(pkg.Root.Name)
	doc := pkg.ServiceName() + " manages the " + res + " branch of a vehicle.\n\n"
	if pkg.TopLevel() && pkg.Family == FamilyVSS {
		doc = "Vehicles manages vehicles.\n\n" +
			"A Vehicle here carries only its own signals -- speed, weight, trip " +
			"distance. Its branches are separate resources under it, each with its " +
			"own service: Cabins, Powertrains, Chassis and the rest. AIP-215 " +
			"<https://aip.dev/215> forbids a field naming a message in another " +
			"package, so the tree is navigated by resource name rather than read " +
			"whole in one call.\n\n" +
			"That is a real cost, and it is accepted: it is what lets a consumer " +
			"read a cabin without transferring a powertrain.\n\n"
	} else if pkg.Singleton() {
		doc += "A **singleton**, AIP-156 <https://aip.dev/156>: a vehicle has exactly " +
			"one, its name has no id segment, and it cannot be created or deleted " +
			"independently of the vehicle. Get and Update are the whole surface.\n\n"
	} else {
		doc += "A vehicle may hold several, so this is an ordinary collection with " +
			"the AIP-131..135 standard methods plus AIP-164 <https://aip.dev/164> " +
			"undelete.\n\n"
	}
	doc += "Reference: COVESA Vehicle Signal Specification.\n" +
		"https://covesa.github.io/vehicle_signal_specification/"
	return doc
}

// sortedStrings returns a sorted copy of s.
func sortedStrings(s []string) []string {
	out := append([]string(nil), s...)
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j] < out[j-1]; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}
