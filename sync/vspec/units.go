// Copyright 2026 The Protobuf Project authors.
// SPDX-License-Identifier: Apache-2.0

package vspec

// units.go reads the unit and quantity catalogues VSS ships beside the tree.
//
// The generator used to carry these as a hand-written table, transcribed from
// the specification. VSS publishes them as data — `units.yaml` and
// `quantities.yaml` — so the table was a copy that could drift, and now it is
// read instead.
//
// Each unit names the quantity it measures and, for most, a QUDT identifier:
// a URI into a published ontology of units. Carrying that through means a
// consumer can resolve what a number means against something other than this
// schema's own vocabulary.

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Unit is one entry of the VSS unit catalogue.
type Unit struct {
	// Symbol is how VSS writes the unit in a signal: "mm", "km/h".
	Symbol string

	// Name is the unit spelled out: "millimeter".
	Name string `yaml:"unit"`

	// Definition is the one-line description the catalogue carries.
	Definition string `yaml:"definition"`

	// Quantity is the physical quantity measured: "length", "velocity".
	Quantity string `yaml:"quantity"`

	// AllowedDatatypes constrains which datatypes may carry the unit.
	AllowedDatatypes []string `yaml:"allowed-datatypes"`

	// Deprecation is the note VSS attaches to a unit on its way out.
	Deprecation string `yaml:"deprecation"`

	// QUDT identifies the unit and its quantity kind in the QUDT ontology,
	// where the catalogue gives them.
	QUDT struct {
		Unit         string `yaml:"unit"`
		QuantityKind string `yaml:"quantity-kind"`
	} `yaml:"qudt"`
}

// Quantity is one entry of the VSS quantity catalogue.
type Quantity struct {
	// Name is the quantity: "length".
	Name string

	// Definition and Remark are the catalogue's own prose, usually citing the
	// ISO 80000 clause the quantity is defined in.
	Definition string `yaml:"definition"`
	Remark     string `yaml:"remark"`
	Comment    string `yaml:"comment"`
}

// Catalogue is the unit and quantity vocabulary.
type Catalogue struct {
	Units      map[string]*Unit
	Quantities map[string]*Quantity
}

// LoadCatalogue reads units.yaml and quantities.yaml from a specification
// directory.
func LoadCatalogue(specDir string) (*Catalogue, error) {
	c := &Catalogue{
		Units:      map[string]*Unit{},
		Quantities: map[string]*Quantity{},
	}
	if err := load(filepath.Join(specDir, "units.yaml"), &c.Units); err != nil {
		return nil, err
	}
	if err := load(filepath.Join(specDir, "quantities.yaml"), &c.Quantities); err != nil {
		return nil, err
	}
	for symbol, u := range c.Units {
		u.Symbol = symbol
		if _, ok := c.Quantities[u.Quantity]; !ok {
			return nil, fmt.Errorf("unit %q measures %q, which quantities.yaml does not define",
				symbol, u.Quantity)
		}
	}
	for name, q := range c.Quantities {
		q.Name = name
	}
	return c, nil
}

// load decodes one catalogue file, rejecting a key the schema does not know.
func load[T any](path string, into *map[string]T) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	decoder := yaml.NewDecoder(bytes.NewReader(raw))
	decoder.KnownFields(true)
	if err := decoder.Decode(into); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	return nil
}
