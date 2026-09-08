// sync turns the COVESA Vehicle Data Model into protobuf.
//
// Deliberately dependency-free: the `require` block is empty and should stay
// that way. The generator reads GraphQL SDL, a four-key YAML pin and writes
// text, none of which needs a library -- and a generator with no dependencies
// is one that still builds when someone returns to it in two years.
module github.com/the-protobuf-project/vdm/sync

go 1.24
