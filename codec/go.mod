// codec converts between this schema and the COVESA Vehicle Data Model's own
// shape, so a consumer who knows VSS can read data produced against the
// protobuf and vice versa.
//
// Its own module for the reason mermaid is: it is a *consumer* of the schema,
// not part of producing it. Nothing in the generation pipeline may depend on
// a codec, and a module boundary is what makes that true rather than merely
// intended.
//
// It depends on sync only to build the manifest. The runtime conversion
// reads the manifest, not the generator, which is what lets a third party in
// any language do the same thing.
module github.com/the-protobuf-project/vdm/codec

go 1.24

require github.com/the-protobuf-project/vdm/sync v0.0.0

replace github.com/the-protobuf-project/vdm/sync => ../sync
