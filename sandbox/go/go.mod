// Compile-only module for the generated Go, copied over gen/go in CI.
//
// Not a library anyone imports: it exists so a build failure in the generated
// code is caught here rather than by the first consumer. `buf lint` cannot
// tell you whether a schema produces code that compiles.
module github.com/oh-tarnished/coversa-protobuf/gen/go

go 1.24.0
