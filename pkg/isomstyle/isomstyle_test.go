package isomstyle

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"go.yaml.in/yaml/v3"
)

const specPath = "../../isom.yaml"

func load(t *testing.T) *Spec {
	t.Helper()
	s, err := Load(specPath)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func styleLayers(t *testing.T) []map[string]any {
	t.Helper()
	b, err := load(t).StyleJSON()
	if err != nil {
		t.Fatal(err)
	}
	var style struct {
		Layers []map[string]any `json:"layers"`
	}
	if err := json.Unmarshal(b, &style); err != nil {
		t.Fatal(err)
	}
	return style.Layers
}

func TestDefinitionMatchesSchema(t *testing.T) {
	t.Parallel()
	c := jsonschema.NewCompiler()
	sch, err := c.Compile("../../isom.schema.json")
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatal(err)
	}
	var doc any
	if err := yaml.Unmarshal(b, &doc); err != nil {
		t.Fatal(err)
	}
	// Round-trip through JSON so the validator sees JSON types.
	j, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	inst, err := jsonschema.UnmarshalJSON(strings.NewReader(string(j)))
	if err != nil {
		t.Fatal(err)
	}
	if err := sch.Validate(inst); err != nil {
		t.Fatal(err)
	}
}

func TestParseRejectsBrokenDefinitions(t *testing.T) {
	t.Parallel()
	base, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatal(err)
	}
	cases := map[string][2]string{
		"colour outside palette": {`fill: { color: *yellow } }`, `fill: { color: "#123456" } }`},
		"unknown table":          {`table: vegetation_areas, fill: { color: *yellow } }`, `table: nope, fill: { color: *yellow } }`},
		"unknown field":          {`fill: { color: *yellow } }`, `fill: { color: *yellow }, colour: x }`},
		"two kinds":              {`fill: { color: *yellow } }`, `fill: { color: *yellow }, line: { color: *black, width: 1 } }`},
		"unknown image":          {`image: "111"`, `image: "999"`},
		"odd dash":               {`dash: [1.25, 0.25]`, `dash: [1.25, 0.25, 1]`},
	}
	for name, c := range cases {
		broken := strings.Replace(string(base), c[0], c[1], 1)
		if broken == string(base) {
			t.Fatalf("%s: fixture text %q not found", name, c[0])
		}
		if _, err := Parse([]byte(broken)); err == nil {
			t.Errorf("%s: Parse accepted it", name)
		}
	}
}

func TestPxMatchesConversionTable(t *testing.T) {
	t.Parallel()
	sc := load(t).Scale
	// mm @1:15,000 → CSS px @1:10,000, 96 dpi.
	cases := map[float64]float64{0.10: 0.567, 0.14: 0.794, 0.18: 1.021, 0.25: 1.417, 0.35: 1.984, 1.00: 5.669}
	for mm, want := range cases {
		if got := sc.Px(mm); math.Abs(got-want) > 0.001 {
			t.Errorf("Px(%v) = %v, want %v", mm, got, want)
		}
	}
}

// Line dimensions as ISOM 2017-2 gives them (mm @1:15,000), cross-checked
// against the OpenOrienteering Mapper ISOM 2017-2 symbol set. Kept apart from
// isom.yaml on purpose: this is the independent statement the definition must
// match.
var isomLines = map[string]struct {
	width float64
	dash  []float64
}{
	"101.000": {0.14, nil},
	"102.000": {0.25, nil},
	"103.000": {0.10, []float64{2.0, 0.2}},
	"104.000": {0.18, nil},
	"105.000": {0.18, nil},
	"201.000": {0.35, nil},
	"202.000": {0.25, nil},
	"301.000": {0.18, nil},
	"302.000": {0.10, nil},
	"304.000": {0.30, nil},
	"305.000": {0.18, nil},
	"306.000": {0.18, []float64{1.25, 0.25}},
	"415.000": {0.14, nil},
	"501.000": {0.14, nil},
	"503.000": {0.35, nil},
	"504.000": {0.35, []float64{3.0, 0.25}},
	"505.000": {0.25, []float64{2.0, 0.25}},
	"506.000": {0.18, []float64{1.0, 0.25}},
	"507.000": {0.18, []float64{1.0, 0.25, 1.0, 0.8}},
	"510.000": {0.14, nil},
	"515.000": {0.25, nil},
	"516.000": {0.14, nil},
	"521.001": {0.20, nil},
	"529.000": {0.25, nil},
}

func TestLineDimensionsMatchISOM(t *testing.T) {
	t.Parallel()
	seen := map[string]bool{}
	for _, sym := range load(t).Symbols() {
		want, ok := isomLines[sym.Code]
		if sym.Line == nil || !ok {
			continue
		}
		seen[sym.Code] = true
		if sym.Line.Width != want.width || !slices.Equal(sym.Line.Dash, want.dash) {
			t.Errorf("%s: width %v dash %v, want %v %v", sym.Code, sym.Line.Width, sym.Line.Dash, want.width, want.dash)
		}
	}
	for code := range isomLines {
		if !seen[code] {
			t.Errorf("%s: no single line layer in the definition", code)
		}
	}
}

func TestDashArrayInWidthUnits(t *testing.T) {
	t.Parallel()
	// 505: width 0.25, dash 2.0/0.25 → 8 widths dash, 1 width gap.
	for _, l := range styleLayers(t) {
		if l["id"] == nil || !strings.HasPrefix(l["id"].(string), "d") || !strings.HasSuffix(l["id"].(string), "505.000") {
			continue
		}
		d := l["paint"].(map[string]any)["line-dasharray"].([]any)
		if math.Abs(d[0].(float64)-8) > 0.05 || math.Abs(d[1].(float64)-1) > 0.05 {
			t.Errorf("505 dasharray = %v, want ~[8 1]", d)
		}
		return
	}
	t.Fatal("no 505 layer")
}

func TestOnlyPaletteColours(t *testing.T) {
	t.Parallel()
	s := load(t)
	allowed := map[string]bool{}
	for _, c := range s.Colors {
		allowed[strings.ToUpper(c)] = true
	}
	style, _ := s.StyleJSON()
	icons, _ := s.IconsJSON()
	hex := regexp.MustCompile(`#[0-9A-Fa-f]{6}`)
	found := hex.FindAllString(string(style)+string(icons), -1)
	if len(found) == 0 {
		t.Fatal("no colours found")
	}
	for _, c := range found {
		if !allowed[strings.ToUpper(c)] {
			t.Errorf("colour %s is not in the palette", c)
		}
	}
}

func TestLayerOrderFollowsColourStack(t *testing.T) {
	t.Parallel()
	layers := styleLayers(t)
	pos := func(code, typ string) int {
		for i, l := range layers {
			id, _ := l["id"].(string)
			if strings.HasPrefix(id, "d") && strings.HasSuffix(id, code) && l["type"] == typ {
				return i
			}
		}
		t.Fatalf("no detail %s layer for %s", typ, code)
		return -1
	}
	below := [][4]string{
		{"401.000", "fill", "405.000", "fill"}, // yellow < white
		{"405.000", "fill", "406.000", "fill"}, // white < green 30%
		{"406.000", "fill", "408.000", "fill"}, // green 30% < green 60%
		{"408.000", "fill", "410.000", "fill"}, // green 60% < green
		{"410.000", "fill", "308.000", "fill"}, // green < blue tints
		{"521.001", "fill", "501.000", "fill"}, // black tints < brown 50%
		{"501.000", "fill", "301.000", "fill"}, // brown 50% < blue
		{"301.000", "fill", "101.000", "line"}, // blue < brown
		{"101.000", "line", "521.001", "line"}, // brown < black
		{"202.000", "line", "206.000", "fill"}, // cliff edges < massive cliff face
	}
	for _, b := range below {
		if pos(b[0], b[1]) >= pos(b[2], b[3]) {
			t.Errorf("%s %s must be below %s %s", b[0], b[1], b[2], b[3])
		}
	}
	if layers[0]["id"] != "background" {
		t.Error("background must be the bottom layer")
	}
}

// The only permitted zoom expression is uniform magnification: constant up to
// lockZoom, then exactly ×2 per zoom level.
func TestOnlyUniformMagnification(t *testing.T) {
	t.Parallel()
	for _, l := range styleLayers(t) {
		for _, key := range []string{"paint", "layout"} {
			props, _ := l[key].(map[string]any)
			for name, v := range props {
				b, _ := json.Marshal(v)
				if !strings.Contains(string(b), `"zoom"`) {
					continue
				}
				e, ok := v.([]any)
				ok = ok && len(e) == 7 && e[0] == "interpolate" &&
					fmt.Sprint(e[1]) == "[exponential 2]" && fmt.Sprint(e[2]) == "[zoom]" &&
					e[3] == 15.0 && e[5] == 22.0 && math.Abs(e[6].(float64)-e[4].(float64)*128) < 0.5
				if !ok {
					t.Errorf("layer %v %s: non-uniform zoom expression %s", l["id"], name, b)
				}
			}
		}
	}
}

func TestLineRastersTileSeamlessly(t *testing.T) {
	t.Parallel()
	s := load(t)
	dims := regexp.MustCompile(`width="(\d+)" height="(\d+)"`)
	for key, img := range s.Images {
		r := img.LineRaster
		if r == nil {
			continue
		}
		svg := s.Icons()[s.Sprite.ID+":"+key]
		m := dims.FindStringSubmatch(svg)
		if m == nil {
			t.Fatalf("%s: canvas is not whole pixels: %s", key, svg)
		}
		across, _ := strconv.Atoi(m[1])
		if r.Angle == 0 {
			across, _ = strconv.Atoi(m[2])
		}
		lines := strings.Count(svg, "M")
		got, want := float64(across)/float64(lines), s.Scale.Px(r.Spacing)
		if math.Abs(got-want)/want > 0.005 {
			t.Errorf("%s: spacing %.3f px, spec %.3f px", key, got, want)
		}
	}
}
