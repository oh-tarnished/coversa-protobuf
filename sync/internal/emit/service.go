// Copyright 2026 The Protobuf Project authors.
// SPDX-License-Identifier: Apache-2.0

package emit

// service.go writes each package's service.proto: the RPCs, with the HTTP
// mapping, method signatures and per-method error documentation AIP requires.

import (
	"sort"
	"strings"

	"github.com/the-protobuf-project/vdm/sync/model"
	"github.com/the-protobuf-project/vdm/sync/naming"
)

// Service renders service.proto for a package.
func (e *Emitter) Service(pkg *model.Package) string {
	imports := []string{
		"google/api/annotations.proto",
		"google/api/client.proto",
		pkg.ImportPath("messages.proto"),
		pkg.ImportPath(model.TypeFile(pkg.Root.Name)),
	}
	if !pkg.Singleton() {
		imports = append(imports, "google/protobuf/empty.proto")
	}
	sort.Strings(imports)

	var sb strings.Builder
	sb.WriteString(e.fileHeader(pkg, "ServiceProto", imports))
	sb.WriteString(docBlock(serviceDoc(pkg), ""))
	sb.WriteString("service " + pkg.ServiceName() + " {\n")

	e.rpcGet(pkg, &sb)
	if !pkg.Singleton() {
		e.rpcList(pkg, &sb)
		e.rpcCreate(pkg, &sb)
	}
	e.rpcUpdate(pkg, &sb)
	if !pkg.Singleton() {
		e.rpcDelete(pkg, &sb)
		e.rpcUndelete(pkg, &sb)
	}
	sb.WriteString("}\n")
	return sb.String()
}

// rpcGet writes the Get method, AIP-131.
func (e *Emitter) rpcGet(pkg *model.Package, sb *strings.Builder) {
	res := pkg.ResourceName()
	sb.WriteString(docBlock("Retrieves one "+res+".\n\n"+
		"  GET "+pkg.HTTPPath()+"\n\n"+
		"Safe and idempotent. Read the `etag` here and pass it back on update "+
		"to make the write conditional.\n\n"+
		"  NOT_FOUND          no such "+res+"\n"+
		"  INVALID_ARGUMENT   name does not match the resource pattern\n"+
		"  PERMISSION_DENIED  caller may not read it", "  "))
	sb.WriteString("  rpc Get" + res + "(Get" + res + "Request) returns (" + res + ") {\n")
	sb.WriteString("    option (google.api.http) = {get: \"" + pkg.HTTPPath() + "\"};\n")
	sb.WriteString("    option (google.api.method_signature) = \"name\";\n  }\n\n")
}

// rpcList writes the List method, AIP-132.
func (e *Emitter) rpcList(pkg *model.Package, sb *strings.Builder) {
	plural := pkg.PluralType()
	sb.WriteString(docBlock("Lists the "+pkg.Plural+" of one vehicle.\n\n"+
		"  GET "+pkg.CollectionPath()+"\n\n"+
		"Paginated per AIP-158 <https://aip.dev/158>: pass `page_size` (defaults "+
		"to 50, capped at 1000) and the `next_page_token` from the previous "+
		"response. Page tokens are opaque and may expire; do not construct or "+
		"store them.\n\n"+
		"  INVALID_ARGUMENT   malformed filter, or page_size out of range\n"+
		"  PERMISSION_DENIED  caller may not list this collection", "  "))
	sb.WriteString("  rpc List" + plural + "(List" + plural +
		"Request) returns (List" + plural + "Response) {\n")
	sb.WriteString("    option (google.api.http) = {get: \"" + pkg.CollectionPath() + "\"};\n")
	if !pkg.TopLevel() {
		sb.WriteString("    option (google.api.method_signature) = \"parent\";\n")
	}
	sb.WriteString("  }\n\n")
}

// rpcCreate writes the Create method, AIP-133.
func (e *Emitter) rpcCreate(pkg *model.Package, sb *strings.Builder) {
	res, lower := pkg.ResourceName(), naming.Snake(pkg.Root.Name)
	sb.WriteString(docBlock("Creates a "+res+".\n\n"+
		"  POST "+pkg.CollectionPath()+"\n"+
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
	sb.WriteString("      post: \"" + pkg.CollectionPath() + "\"\n")
	sb.WriteString("      body: \"" + lower + "\"\n    };\n")
	if pkg.TopLevel() {
		sb.WriteString("    option (google.api.method_signature) = \"" + lower + "," + lower + "_id\";\n")
	} else {
		sb.WriteString("    option (google.api.method_signature) = \"parent," + lower + "," + lower + "_id\";\n")
	}
	sb.WriteString("  }\n\n")
}

// rpcUpdate writes the Update method, AIP-134.
func (e *Emitter) rpcUpdate(pkg *model.Package, sb *strings.Builder) {
	res, lower := pkg.ResourceName(), naming.Snake(pkg.Root.Name)
	path := strings.Replace(pkg.HTTPPath(), "{name=", "{"+lower+".name=", 1)

	sb.WriteString(docBlock("Updates a "+res+".\n\n"+
		"  PATCH "+path+"\n"+
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
	sb.WriteString("      patch: \"" + path + "\"\n")
	sb.WriteString("      body: \"" + lower + "\"\n    };\n")
	sb.WriteString("    option (google.api.method_signature) = \"" + lower + ",update_mask\";\n  }\n")
}

// rpcDelete writes the Delete method, AIP-135.
func (e *Emitter) rpcDelete(pkg *model.Package, sb *strings.Builder) {
	res := pkg.ResourceName()
	sb.WriteString("\n")
	sb.WriteString(docBlock("Deletes a "+res+".\n\n"+
		"  DELETE "+pkg.HTTPPath()+"\n\n"+
		"Soft delete: the "+res+" is marked deleted, disappears from List unless "+
		"`show_deleted` is set, and is purged at `expire_time`. Recover it with "+
		"Undelete"+res+" before then.\n\n"+
		"Idempotent in effect but not in reporting: a second delete returns "+
		"NOT_FOUND rather than succeeding silently.\n\n"+
		"  NOT_FOUND          no such "+res+"\n"+
		"  ABORTED            etag does not match\n"+
		"  PERMISSION_DENIED  caller may not delete it", "  "))
	sb.WriteString("  rpc Delete" + res + "(Delete" + res + "Request) returns (google.protobuf.Empty) {\n")
	sb.WriteString("    option (google.api.http) = {delete: \"" + pkg.HTTPPath() + "\"};\n")
	sb.WriteString("    option (google.api.method_signature) = \"name\";\n  }\n\n")
}

// rpcUndelete writes the Undelete method, AIP-164.
func (e *Emitter) rpcUndelete(pkg *model.Package, sb *strings.Builder) {
	res := pkg.ResourceName()
	sb.WriteString(docBlock("Restores a soft-deleted "+res+".\n\n"+
		"  POST "+pkg.HTTPPath()+":undelete\n\n"+
		"Valid only before `expire_time`; after that the "+res+" is gone and "+
		"this returns NOT_FOUND. Clears `delete_time` and `expire_time`.\n\n"+
		"  NOT_FOUND          no such "+res+", or already purged\n"+
		"  ALREADY_EXISTS     the "+res+" is not deleted\n"+
		"  PERMISSION_DENIED  caller may not restore it", "  "))
	sb.WriteString("  rpc Undelete" + res + "(Undelete" + res + "Request) returns (" + res + ") {\n")
	sb.WriteString("    option (google.api.http) = {\n")
	sb.WriteString("      post: \"" + pkg.HTTPPath() + ":undelete\"\n")
	sb.WriteString("      body: \"*\"\n    };\n")
	sb.WriteString("    option (google.api.method_signature) = \"name\";\n  }\n")
}

// serviceDoc builds the service's documentation block.
func serviceDoc(pkg *model.Package) string {
	res := pkg.ResourceName()
	doc := pkg.ServiceName() + " manages the " + res + " branch of a vehicle.\n\n"

	switch {
	case pkg.TopLevel() && pkg.Family == model.FamilyVSS:
		doc = "Vehicles manages vehicles.\n\n" +
			"A Vehicle here carries only its own signals -- speed, weight, trip " +
			"distance. Its branches are separate resources under it, each with its " +
			"own service: Cabins, Powertrains, Chassis and the rest. AIP-215 " +
			"<https://aip.dev/215> forbids a field naming a message in another " +
			"package, so the tree is navigated by resource name rather than read " +
			"whole in one call.\n\n" +
			"That is a real cost, and it is accepted: it is what lets a consumer " +
			"read a cabin without transferring a powertrain.\n\n"
	case pkg.Singleton():
		doc += "A **singleton**, AIP-156 <https://aip.dev/156>: a vehicle has exactly " +
			"one, its name has no id segment, and it cannot be created or deleted " +
			"independently of the vehicle. Get and Update are the whole surface.\n\n"
	default:
		doc += "A vehicle may hold several, so this is an ordinary collection with " +
			"the AIP-131..135 standard methods plus AIP-164 <https://aip.dev/164> " +
			"undelete.\n\n"
	}
	return doc + "Reference: COVESA Vehicle Signal Specification.\n" +
		"https://covesa.github.io/vehicle_signal_specification/"
}
