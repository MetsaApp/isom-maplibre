// Package isomstyle turns the ISOM style definition (isom.yaml at the repository
// root, embedded and loaded by Default) into MapLibre styles and their
// pattern/point-symbol SVGs.
//
// The definition stores every symbol dimension in mm at the ISOM specification
// scale; Scale.Px is the one place they become CSS px.
package isomstyle

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"strings"
	"sync"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"go.yaml.in/yaml/v3"

	isommaplibre "github.com/MetsaApp/isom-maplibre"
)

// Spec is a parsed style definition. See isom.schema.json for field docs.
type Spec struct {
	Name   string            `yaml:"name"`
	Scale  Scale             `yaml:"scale"`
	Colors map[string]string `yaml:"colors"`
	Tables []string          `yaml:"tables"`
	Images map[string]Image  `yaml:"images"`
	Stack  []Group           `yaml:"stack"`
	Sprite Sprite            `yaml:"sprite"`
	Tiles  Tiles             `yaml:"tiles"`
}

type Scale struct {
	SpecDenominator float64 `yaml:"specDenominator"`
	MapDenominator  float64 `yaml:"mapDenominator"`
	CSSDPI          float64 `yaml:"cssDpi"`
	LockZoom        float64 `yaml:"lockZoom"`
	MaxZoom         float64 `yaml:"maxZoom"`
}

// Px converts a dimension in mm at the spec scale to CSS px at the map scale,
// rounded to 3 decimals.
func (s Scale) Px(mm float64) float64 {
	return round(mm*s.SpecDenominator/s.MapDenominator*s.CSSDPI/25.4, 3)
}

type Group struct {
	Group   string   `yaml:"group"`
	Symbols []Symbol `yaml:"symbols"`
}

// Symbol is one styled isom_code; exactly one of Fill, Line, Circle, Icon is set.
type Symbol struct {
	Code   string  `yaml:"code"`
	Table  string  `yaml:"table"`
	Fill   *Fill   `yaml:"fill"`
	Line   *Line   `yaml:"line"`
	Circle *Circle `yaml:"circle"`
	Icon   *Icon   `yaml:"icon"`
}

type Fill struct {
	Color   string `yaml:"color"`
	Pattern string `yaml:"pattern"`
}

type Line struct {
	Color string    `yaml:"color"`
	Width float64   `yaml:"width"`
	Dash  []float64 `yaml:"dash"`
}

type Circle struct {
	Color    string  `yaml:"color"`
	Diameter float64 `yaml:"diameter"`
}

type Icon struct {
	Image     string `yaml:"image"`
	Placement string `yaml:"placement"`
}

// Image is one drawn image; exactly one field is set.
type Image struct {
	LineRaster *LineRaster `yaml:"lineRaster"`
	HalfCircle *HalfCircle `yaml:"halfCircle"`
	Tick       *Tick       `yaml:"tick"`
}

type LineRaster struct {
	Angle   int     `yaml:"angle"`
	Width   float64 `yaml:"width"`
	Spacing float64 `yaml:"spacing"`
	Color   string  `yaml:"color"`
}

type HalfCircle struct {
	Diameter float64 `yaml:"diameter"`
	Width    float64 `yaml:"width"`
	Color    string  `yaml:"color"`
}

type Tick struct {
	Length float64 `yaml:"length"`
	Width  float64 `yaml:"width"`
	Color  string  `yaml:"color"`
}

type Sprite struct {
	ID  string `yaml:"id"`
	URL string `yaml:"url"`
}

type Tiles struct {
	URLPrefix     string    `yaml:"urlPrefix"`
	DetailMinZoom int       `yaml:"detailMinZoom"`
	Overview      *Overview `yaml:"overview"`
	Coverage      *Coverage `yaml:"coverage"`
}

type Overview struct {
	Source string   `yaml:"source"`
	Tables []string `yaml:"tables"`
}

type Coverage struct {
	Source       string `yaml:"source"`
	PaperMinZoom int    `yaml:"paperMinZoom"`
	Patch        struct {
		Color     string  `yaml:"color"`
		Opacity   float64 `yaml:"opacity"`
		LineWidth float64 `yaml:"lineWidth"`
	} `yaml:"patch"`
}

// Default parses the definition shipped with this module (isom.yaml).
func Default() (*Spec, error) { return Parse(isommaplibre.Definition) }

// Load reads and validates a definition file.
func Load(path string) (*Spec, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return Parse(b)
}

// Parse validates a definition against isom.schema.json, decodes it, and then
// checks the relations the schema cannot express (see validate).
func Parse(b []byte) (*Spec, error) {
	if err := matchSchema(b); err != nil {
		return nil, fmt.Errorf("isomstyle: %w", err)
	}
	// Unknown fields are an error so the struct cannot drift from the schema.
	dec := yaml.NewDecoder(bytes.NewReader(b))
	dec.KnownFields(true)
	var s Spec
	if err := dec.Decode(&s); err != nil {
		return nil, fmt.Errorf("isomstyle: %w", err)
	}
	if err := s.validate(); err != nil {
		return nil, fmt.Errorf("isomstyle: %w", err)
	}
	return &s, nil
}

var schema = sync.OnceValues(func() (*jsonschema.Schema, error) {
	doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(isommaplibre.Schema))
	if err != nil {
		return nil, err
	}
	const url = "https://github.com/MetsaApp/isom-maplibre/isom.schema.json" // its $id
	c := jsonschema.NewCompiler()
	if err := c.AddResource(url, doc); err != nil {
		return nil, err
	}
	return c.Compile(url)
})

func matchSchema(b []byte) error {
	sch, err := schema()
	if err != nil {
		return fmt.Errorf("schema: %w", err)
	}
	var doc any
	if err := yaml.Unmarshal(b, &doc); err != nil {
		return err
	}
	// Round-trip through JSON so the validator sees JSON types.
	j, err := json.Marshal(doc)
	if err != nil {
		return err
	}
	inst, err := jsonschema.UnmarshalJSON(bytes.NewReader(j))
	if err != nil {
		return err
	}
	return sch.Validate(inst)
}

// Symbols returns the stack flattened, bottom to top.
func (s *Spec) Symbols() []Symbol {
	var out []Symbol
	for _, g := range s.Stack {
		out = append(out, g.Symbols...)
	}
	return out
}

// validate checks what the schema cannot express: colours come from the
// palette, table and image references resolve, dashes come in pairs, and
// dimensions that bound each other are ordered.
func (s *Spec) validate() error {
	var errs []error
	fail := func(format string, a ...any) { errs = append(errs, fmt.Errorf(format, a...)) }

	if s.Scale.MaxZoom <= s.Scale.LockZoom {
		fail("scale: maxZoom must be above lockZoom")
	}
	palette := map[string]bool{}
	for _, c := range s.Colors {
		palette[strings.ToUpper(c)] = true
	}
	color := func(where, c string) {
		if !palette[strings.ToUpper(c)] {
			fail("%s: colour %q is not in the palette", where, c)
		}
	}
	tables := map[string]bool{}
	for _, t := range s.Tables {
		tables[t] = true
	}

	for key, img := range s.Images {
		where := "image " + key
		switch {
		case img.LineRaster != nil:
			color(where, img.LineRaster.Color)
			if img.LineRaster.Spacing <= img.LineRaster.Width {
				fail("%s: width must be below spacing", where)
			}
		case img.HalfCircle != nil:
			color(where, img.HalfCircle.Color)
			if img.HalfCircle.Diameter <= img.HalfCircle.Width {
				fail("%s: width must be below diameter", where)
			}
		case img.Tick != nil:
			color(where, img.Tick.Color)
		}
	}

	for _, sym := range s.Symbols() {
		where := "symbol " + sym.Code
		if !tables[sym.Table] {
			fail("%s: table %q is not in tables", where, sym.Table)
		}
		switch {
		case sym.Fill != nil && sym.Fill.Color != "":
			color(where, sym.Fill.Color)
		case sym.Fill != nil:
			if s.Images[sym.Fill.Pattern].LineRaster == nil {
				fail("%s: pattern %q is not a lineRaster image", where, sym.Fill.Pattern)
			}
		case sym.Line != nil:
			color(where, sym.Line.Color)
			if len(sym.Line.Dash)%2 == 1 {
				fail("%s: dash needs dash/gap pairs", where)
			}
		case sym.Circle != nil:
			color(where, sym.Circle.Color)
		case sym.Icon != nil:
			if _, ok := s.Images[sym.Icon.Image]; !ok {
				fail("%s: image %q is not defined", where, sym.Icon.Image)
			}
		}
	}

	if o := s.Tiles.Overview; o != nil {
		for _, t := range o.Tables {
			if !tables[t] {
				fail("tiles.overview: table %q is not in tables", t)
			}
		}
	}
	if c := s.Tiles.Coverage; c != nil {
		color("tiles.coverage.patch", c.Patch.Color)
	}
	return errors.Join(errs...)
}

func round(v float64, places int) float64 {
	p := math.Pow(10, float64(places))
	return math.Round(v*p) / p
}
