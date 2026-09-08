// Compile-only module for the generated Go, copied over gen/go in CI.
//
// Not a library anyone imports: it exists so a build failure in the generated
// code is caught here rather than by the first consumer. `buf lint` cannot
// tell you whether a schema produces code that compiles.
module github.com/the-protobuf-project/vdm/gen/go

go 1.24

require (
	buf.build/go/protovalidate v0.14.0
	google.golang.org/genproto/googleapis/api v0.0.0-20250811230008-5f3141c8851a
	google.golang.org/grpc v1.76.0
	google.golang.org/protobuf v1.36.10
)
