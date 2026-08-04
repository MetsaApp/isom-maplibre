package isomstyle

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"testing"
)

func TestPxMatchesSpecTable(t *testing.T) {
	t.Parallel()
	// ISOM 2017-2 §1.1 conversion table.
	cases := map[float64]float64{0.10: 0.567, 0.14: 0.794, 0.18: 1.021, 0.25: 1.417, 1.00: 5.669}
	for mm, want := range cases {
		if got := Px(mm); math.Abs(got-want) > 0.01 {
			t.Errorf("Px(%v) = %v, want %v", mm, got, want)
		}
	}
}

func styleLayers(t *testing.T) []map[string]any {
	t.Helper()
	b, err := JSON()
	if err != nil {
		t.Fatal(err)
	}
	var style struct {
		Layers []map[string]any `json:"layers"`
	}
	if uerr := json.Unmarshal(b, &style); uerr != nil {
		t.Fatal(uerr)
	}
	return style.Layers
}

// walk collects every string value in a JSON tree.
func walk(v any, out *[]string) {
	switch x := v.(type) {
	case string:
		*out = append(*out, x)
	case []any:
		for _, e := range x {
			walk(e, out)
		}
	case map[string]any:
		for _, e := range x {
			walk(e, out)
		}
	}
}

func TestOnlyPaletteColours(t *testing.T) {
	t.Parallel()
	allowed := map[string]bool{}
	for _, c := range Palette {
		allowed[c] = true
	}
	b, _ := JSON()
	var tree any
	_ = json.Unmarshal(b, &tree)
	var strs []string
	walk(tree, &strs)
	seen := 0
	for _, s := range strs {
		if strings.HasPrefix(s, "#") {
			seen++
			if !allowed[strings.ToUpper(s)] {
				t.Errorf("colour %q not in the §3.1 palette", s)
			}
		}
	}
	if seen == 0 {
		t.Fatal("no colours found: walk broken?")
	}
}

func TestLayerOrderFollowsColourStack(t *testing.T) {
	t.Parallel()
	layers := styleLayers(t)
	pos := func(code, ltype string) int {
		for i, l := range layers {
			id, _ := l["id"].(string)
			if strings.HasPrefix(id, "d") && strings.HasSuffix(id, code) && l["type"] == ltype {
				return i
			}
		}
		t.Fatalf("no detail %s layer for %s", ltype, code)
		return -1
	}
	// §3.2 bottom→top: yellow < green < blue fill < contours (brown) < black.
	if pos("401.000", "fill") >= pos("406.000", "fill") || pos("406.000", "fill") >= pos("410.000", "fill") {
		t.Error("yellow/green order wrong")
	}
	if pos("410.000", "fill") >= pos("301.000", "fill") {
		t.Error("blue 100% must be above green")
	}
	if pos("410.000", "fill") >= pos("308.000", "fill") {
		t.Error("marsh (blue tint) must be above green")
	}
	if pos("308.000", "fill") >= pos("301.000", "fill") {
		t.Error("marsh (blue tint) must sit below blue 100%")
	}
	if pos("301.000", "fill") >= pos("101.000", "line") {
		t.Error("brown contours must be above blue fill")
	}
	if pos("521.001", "fill") >= pos("301.000", "fill") {
		t.Error("building 65% infill (black-tints) must sit below blue 100%")
	}
	if pos("101.000", "line") >= pos("521.001", "line") {
		t.Error("building outline (black 100%) must be above contours")
	}
	if layers[0]["id"] != "background" {
		t.Error("background must be the bottom layer")
	}
}

// Zoom expressions are permitted ONLY as the canonical magnification wrapper:
// constant at/below lockZoom, exact ×2-per-level exponential above it (uniform
// enlargement preserves relative ISOM metrics). Anything else is a violation.
func TestOnlyCanonicalZoomMagnification(t *testing.T) {
	t.Parallel()
	for _, l := range styleLayers(t) {
		for _, key := range []string{"paint", "layout"} {
			props, _ := l[key].(map[string]any)
			for name, v := range props {
				checkZoomExpr(t, fmt.Sprintf("layer %v %s", l["id"], name), v)
			}
		}
	}
}

// checkZoomExpr fails unless v is zoom-independent or the canonical
// magnification expression (exponential base 2, stops at z15 and z22=×128).
func checkZoomExpr(t *testing.T, where string, v any) {
	t.Helper()
	b, _ := json.Marshal(v)
	str := string(b)
	zoomish := strings.Contains(str, `"zoom"`) ||
		strings.Contains(str, `"interpolate"`) || strings.Contains(str, `"step"`)
	if !zoomish {
		return
	}
	expr, ok := v.([]any)
	if !ok || len(expr) != 7 || expr[0] != "interpolate" {
		t.Errorf("%s: non-canonical zoom expression %s", where, str)
		return
	}
	base, at15, at22 := expr[1].([]any), expr[4].(float64), expr[6].(float64)
	if base[0] != "exponential" || base[1].(float64) != 2 ||
		expr[2].([]any)[0] != "zoom" || expr[3].(float64) != 15 ||
		expr[5].(float64) != 22 || math.Abs(at22-at15*128) > 0.5 {
		t.Errorf("%s: magnification stops wrong: %s", where, str)
	}
}

func TestFiltersOnlyKnownCodes(t *testing.T) {
	t.Parallel()
	known := map[string]bool{}
	for _, s := range Stack {
		known[s.code] = true
	}
	for _, l := range styleLayers(t) {
		f, ok := l["filter"].([]any)
		if !ok {
			continue // background
		}
		code := f[2].(string)
		if !known[code] {
			t.Errorf("layer %v filters on unknown code %q", l["id"], code)
		}
	}
}

// ISOM 206 massive cliff: the style must carry the derived area's rule as the
// canonical black fill, stacked above the 201/202 cliff lines so a face reads over
// its own edges (isom206-massive-cliffs design D4).
func TestMassiveCliffAreaRendersAboveCliffLines(t *testing.T) {
	t.Parallel()
	layers := styleLayers(t)
	pos, fill := -1, map[string]any(nil)
	linePos := -1
	for i, l := range layers {
		id, _ := l["id"].(string)
		if !strings.HasPrefix(id, "d") {
			continue
		}
		switch {
		case strings.HasSuffix(id, "206.000") && l["type"] == "fill":
			pos, fill = i, l
		case (strings.HasSuffix(id, "201.000") || strings.HasSuffix(id, "202.000")) && l["type"] == "line":
			if i > linePos {
				linePos = i
			}
		}
	}
	if fill == nil {
		t.Fatal("no 206 fill rule: a stored massive-cliff area would be invisible")
	}
	if c := fill["paint"].(map[string]any)["fill-color"]; c != Black {
		t.Errorf("206 fill colour = %v, want the canonical black %s", c, Black)
	}
	if linePos < 0 {
		t.Fatal("no 201/202 line layers found")
	}
	if pos <= linePos {
		t.Errorf("206 fill (layer %d) must stack above the cliff lines (layer %d)", pos, linePos)
	}
}

func TestDashArrayInWidthUnits(t *testing.T) {
	t.Parallel()
	// 505: W 0.25, dash 2.0/0.25 → dash = 2.0/0.25 = 8 widths, gap = 1 width.
	for _, l := range styleLayers(t) {
		if !strings.HasSuffix(l["id"].(string), "505.000") || l["type"] != "line" {
			continue
		}
		paint := l["paint"].(map[string]any)
		d := paint["line-dasharray"].([]any)
		if math.Abs(d[0].(float64)-8) > 0.05 || math.Abs(d[1].(float64)-1) > 0.05 {
			t.Errorf("505 dasharray = %v, want ~[8 1]", d)
		}
		return
	}
	t.Fatal("no 505 line layer found")
}
