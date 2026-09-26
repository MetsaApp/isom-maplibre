package isomstyle

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
)

// Layer metadata keys. Every generated layer except the background carries
// PassKey; symbol layers also carry CodeKey and GroupKey. Consumers select
// layers by these rather than by id.
const (
	CodeKey  = "isom:code"  // the isom_code the layer filters on
	GroupKey = "isom:group" // the stack colour group
	PassKey  = "isom:pass"  // PassDetail, PassOverview or PassCoverage
	// TablesKey is style-level metadata: the tables in definition order.
	TablesKey = "isom:tables"
)

const (
	PassDetail   = "detail"
	PassOverview = "overview"
	PassCoverage = "coverage"
)

// Style returns the vector-tile MapLibre style as a JSON-marshalable tree.
func (s *Spec) Style() map[string]any {
	t := s.Tiles
	sources := map[string]any{}
	source := func(name string) {
		sources[name] = map[string]any{"type": "vector", "url": t.URLPrefix + name}
	}
	for _, name := range s.Tables {
		source(name)
	}

	layers := []any{s.background()}
	if c := t.Coverage; c != nil {
		source(c.Source)
		base := func(id, typ string) map[string]any {
			return map[string]any{
				"id": id, "type": typ, "source": c.Source, "source-layer": c.Source,
				"metadata": map[string]any{PassKey: PassCoverage},
			}
		}
		paper := base("coverage-base", "fill")
		paper["minzoom"] = c.PaperMinZoom
		paper["paint"] = map[string]any{"fill-color": s.Colors["white"]}
		patch := base("coverage-overview", "fill")
		patch["maxzoom"] = c.PaperMinZoom
		patch["paint"] = map[string]any{"fill-color": c.Patch.Color, "fill-opacity": c.Patch.Opacity}
		outline := base("coverage-overview-line", "line")
		outline["maxzoom"] = c.PaperMinZoom
		outline["paint"] = map[string]any{"line-color": c.Patch.Color, "line-width": c.Patch.LineWidth}
		layers = append(layers, paper, patch, outline)
	}

	// The overview pass renders alone below detailMinZoom, so it goes first; the
	// detail pass follows. Within each, stack order is ISOM colour order.
	if o := t.Overview; o != nil {
		source(o.Source)
		in := map[string]bool{}
		for _, name := range o.Tables {
			in[name] = true
		}
		for i, e := range s.stacked() {
			if in[e.Table] && (e.Fill != nil || e.Line != nil) {
				l := s.layer(e, PassOverview, fmt.Sprintf("ov%02d-%s", i, e.Code), o.Source)
				l["source-layer"] = e.Table
				l["maxzoom"] = t.DetailMinZoom
				layers = append(layers, l)
			}
		}
	}
	for _, l := range s.detailLayers() {
		l["source-layer"] = l["source"]
		if t.DetailMinZoom > 0 {
			l["minzoom"] = t.DetailMinZoom
		}
		layers = append(layers, l)
	}

	return map[string]any{
		"version":  8,
		"name":     s.Name,
		"metadata": map[string]any{TablesKey: s.Tables},
		"sprite":   []any{map[string]any{"id": s.Sprite.ID, "url": s.Sprite.URL}},
		"sources":  sources,
		"layers":   layers,
	}
}

// GeojsonStyle returns the style for in-memory data: one empty GeoJSON source
// per table and the detail pass at every zoom. It has no sprite; the caller
// supplies the images (see Icons).
func (s *Spec) GeojsonStyle() map[string]any {
	sources := map[string]any{}
	for _, name := range s.Tables {
		sources[name] = map[string]any{
			"type": "geojson",
			"data": map[string]any{"type": "FeatureCollection", "features": []any{}},
		}
	}
	layers := []any{s.background()}
	for _, l := range s.detailLayers() {
		layers = append(layers, l)
	}
	return map[string]any{
		"version":  8,
		"name":     s.Name,
		"metadata": map[string]any{TablesKey: s.Tables},
		"sources":  sources,
		"layers":   layers,
	}
}

// detailLayers returns one layer per symbol, each on its table's source.
func (s *Spec) detailLayers() []map[string]any {
	var out []map[string]any
	for i, e := range s.stacked() {
		out = append(out, s.layer(e, PassDetail, fmt.Sprintf("d%02d-%s", i, e.Code), e.Table))
	}
	return out
}

func (s *Spec) background() map[string]any {
	return map[string]any{
		"id":    "background",
		"type":  "background",
		"paint": map[string]any{"background-color": s.Colors["white"]},
	}
}

// entry is a symbol with the colour group it is stacked in.
type entry struct {
	Group string
	Symbol
}

// stacked returns the stack flattened with each symbol's group, bottom to top.
func (s *Spec) stacked() []entry {
	var out []entry
	for _, g := range s.Stack {
		for _, sym := range g.Symbols {
			out = append(out, entry{g.Group, sym})
		}
	}
	return out
}

func (s *Spec) layer(e entry, pass, id, source string) map[string]any {
	sym := e.Symbol
	l := map[string]any{
		"id":       id,
		"source":   source,
		"filter":   []any{"==", []any{"get", "isom_code"}, sym.Code},
		"metadata": map[string]any{CodeKey: sym.Code, GroupKey: e.Group, PassKey: pass},
	}
	px := s.Scale.Px
	switch {
	case sym.Fill != nil:
		l["type"] = "fill"
		if sym.Fill.Pattern != "" {
			// fill-pattern cannot be zoom-scaled, so patterns keep their
			// density past lockZoom.
			l["paint"] = map[string]any{"fill-pattern": s.imageID(sym.Fill.Pattern)}
		} else {
			l["paint"] = map[string]any{"fill-color": sym.Fill.Color}
		}
	case sym.Line != nil:
		w := px(sym.Line.Width)
		paint := map[string]any{"line-color": sym.Line.Color, "line-width": s.magnified(w)}
		if len(sym.Line.Dash) > 0 {
			// line-dasharray is in units of line width, so it follows the
			// magnified width without its own zoom expression.
			dash := make([]any, len(sym.Line.Dash))
			for i, d := range sym.Line.Dash {
				dash[i] = round(px(d)/w, 2)
			}
			paint["line-dasharray"] = dash
		}
		l["type"] = "line"
		l["paint"] = paint
		l["layout"] = map[string]any{"line-cap": "butt", "line-join": "round"}
	case sym.Circle != nil:
		l["type"] = "circle"
		l["paint"] = map[string]any{
			"circle-color":  sym.Circle.Color,
			"circle-radius": s.magnified(round(px(sym.Circle.Diameter)/2, 2)),
		}
	case sym.Icon != nil:
		layout := map[string]any{
			"icon-image":              s.imageID(sym.Icon.Image),
			"icon-size":               s.magnified(1),
			"icon-allow-overlap":      true,
			"icon-ignore-placement":   true,
			"icon-rotation-alignment": "map",
		}
		if sym.Icon.Placement == "line-center" {
			layout["symbol-placement"] = "line-center"
		}
		l["type"] = "symbol"
		l["layout"] = layout
	}
	return l
}

// magnified wraps a px value in the one zoom expression the style uses: constant
// up to lockZoom, then exactly ×2 per zoom level up to maxZoom. Uniform
// magnification keeps every ISOM proportion intact.
func (s *Spec) magnified(v float64) any {
	sc := s.Scale
	return []any{
		"interpolate", []any{"exponential", 2}, []any{"zoom"},
		sc.LockZoom, v, sc.MaxZoom, round(v*math.Exp2(sc.MaxZoom-sc.LockZoom), 2),
	}
}

func (s *Spec) imageID(key string) string { return s.Sprite.ID + ":" + key }

// StyleJSON renders Style, indented, with a trailing newline.
func (s *Spec) StyleJSON() ([]byte, error) { return marshal(s.Style()) }

// GeojsonStyleJSON renders GeojsonStyle, indented, with a trailing newline.
func (s *Spec) GeojsonStyleJSON() ([]byte, error) { return marshal(s.GeojsonStyle()) }

func marshal(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
