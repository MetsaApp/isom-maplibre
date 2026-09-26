// Package isomstyle turns the ISOM style definition (isom.yaml at the repository
// root) into a MapLibre style and its pattern/point-symbol SVGs.
//
// The definition stores every symbol dimension in mm at the ISOM specification
// scale; Scale.Px is the one place they become CSS px.
package isomstyle

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"os"
	"regexp"
	"strings"

	"go.yaml.in/yaml/v3"
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

// Load reads and validates a definition file.
func Load(path string) (*Spec, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return Parse(b)
}

// Parse decodes a definition (unknown fields are an error) and validates what
// the schema cannot: colours come from the palette and every reference resolves.
func Parse(b []byte) (*Spec, error) {
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

// Symbols returns the stack flattened, bottom to top.
func (s *Spec) Symbols() []Symbol {
	var out []Symbol
	for _, g := range s.Stack {
		out = append(out, g.Symbols...)
	}
	return out
}

var codeRE = regexp.MustCompile(`^[1-7][0-9]{2}\.[0-9]{3}$`)

func (s *Spec) validate() error {
	var errs []error
	fail := func(format string, a ...any) { errs = append(errs, fmt.Errorf(format, a...)) }

	sc := s.Scale
	if sc.SpecDenominator <= 0 || sc.MapDenominator <= 0 || sc.CSSDPI <= 0 || sc.MaxZoom <= sc.LockZoom {
		fail("scale: denominators and cssDpi must be positive and maxZoom above lockZoom")
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
		case countSet(img.LineRaster != nil, img.HalfCircle != nil, img.Tick != nil) != 1:
			fail("%s: set exactly one of lineRaster, halfCircle, tick", where)
		case img.LineRaster != nil:
			r := img.LineRaster
			color(where, r.Color)
			if r.Angle != 0 && r.Angle != 90 {
				fail("%s: angle must be 0 or 90", where)
			}
			if r.Width <= 0 || r.Spacing <= r.Width {
				fail("%s: need 0 < width < spacing", where)
			}
		case img.HalfCircle != nil:
			color(where, img.HalfCircle.Color)
			if img.HalfCircle.Width <= 0 || img.HalfCircle.Diameter <= img.HalfCircle.Width {
				fail("%s: need 0 < width < diameter", where)
			}
		case img.Tick != nil:
			color(where, img.Tick.Color)
			if img.Tick.Width <= 0 || img.Tick.Length <= 0 {
				fail("%s: length and width must be positive", where)
			}
		}
	}

	for _, sym := range s.Symbols() {
		where := "symbol " + sym.Code
		if !codeRE.MatchString(sym.Code) {
			fail("%s: code must look like 401.000", where)
		}
		if !tables[sym.Table] {
			fail("%s: table %q is not in tables", where, sym.Table)
		}
		switch {
		case countSet(sym.Fill != nil, sym.Line != nil, sym.Circle != nil, sym.Icon != nil) != 1:
			fail("%s: set exactly one of fill, line, circle, icon", where)
		case sym.Fill != nil:
			if (sym.Fill.Color == "") == (sym.Fill.Pattern == "") {
				fail("%s: fill needs exactly one of color, pattern", where)
			} else if sym.Fill.Color != "" {
				color(where, sym.Fill.Color)
			} else if s.Images[sym.Fill.Pattern].LineRaster == nil {
				fail("%s: pattern %q is not a lineRaster image", where, sym.Fill.Pattern)
			}
		case sym.Line != nil:
			color(where, sym.Line.Color)
			if sym.Line.Width <= 0 {
				fail("%s: width must be positive", where)
			}
			if n := len(sym.Line.Dash); n == 1 || n%2 == 1 {
				fail("%s: dash needs dash/gap pairs", where)
			}
		case sym.Circle != nil:
			color(where, sym.Circle.Color)
			if sym.Circle.Diameter <= 0 {
				fail("%s: diameter must be positive", where)
			}
		case sym.Icon != nil:
			if _, ok := s.Images[sym.Icon.Image]; !ok {
				fail("%s: image %q is not defined", where, sym.Icon.Image)
			}
			if p := sym.Icon.Placement; p != "" && p != "point" && p != "line-center" {
				fail("%s: placement must be point or line-center", where)
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

func countSet(bs ...bool) int {
	n := 0
	for _, b := range bs {
		if b {
			n++
		}
	}
	return n
}

func round(v float64, places int) float64 {
	p := math.Pow(10, float64(places))
	return math.Round(v*p) / p
}
