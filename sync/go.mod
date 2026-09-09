// sync turns the COVESA Vehicle Signal Specification and Vehicle Data Model
// into protobuf.
//
// One dependency, and it is deliberate. The generator was dependency-free
// while its input was GraphQL SDL -- a small, regular grammar a hand-written
// scanner handles safely. VSS is written in `.vspec`, which is YAML: block
// scalars, multi-line plain scalars, flow and block sequences. Hand-rolling
// that is precisely the "parser that quietly accepts an unfamiliar shape"
// this repository's rules warn against, so the parsing is delegated to a
// library that has been getting it right for a decade.
//
// Everything else stays hand-written. `#include` resolution and instance
// expansion are VSS's own preprocessor, not YAML, and no library implements
// them.
module github.com/oh-tarnished/coversa-protobuf/sync

go 1.24

require gopkg.in/yaml.v3 v3.0.1
