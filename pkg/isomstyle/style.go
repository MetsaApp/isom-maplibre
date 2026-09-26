package isomstyle

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
)

// Style returns the complete MapLibre style as a JSON-marshalable tree.
func (s *Spec) Style() map[string]any {
	t := s.Tiles
	sources := map[string]any{}
	source := func(name string) {
		sources[name] = map[string]any{"type": "vector", "url": t.URLPrefix + name}
	}
	for _, name := range s.Tables {
		source(name)
	}

	layers := []any{map[string]any{
		"id":    "background",
		"type":  "background",
		"paint": map[string]any{"background-color": s.Colors["white"]},
	}}
	if c := t.Coverage; c != nil {
		source(c.Source)
		base := func(id, typ string) map[string]any {
			return map[string]any{"id": id, "type": typ, "source": c.Source, "source-layer": c.Source}
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
	symbols := s.Symbols()
	if o := t.Overview; o != nil {
		source(o.Source)
		in := map[string]bool{}
		for _, name := range o.Tables {
			in[name] = true
		}
		for i, sym := range symbols {
			if in[sym.Table] && (sym.Fill != nil || sym.Line != nil) {
				l := s.layer(sym, fmt.Sprintf("ov%02d-%s", i, sym.Code), o.Source)
				l["maxzoom"] = t.DetailMinZoom
				layers = append(layers, l)
			}
		}
	}
	for i, sym := range symbols {
		l := s.layer(sym, fmt.Sprintf("d%02d-%s", i, sym.Code), sym.Table)
		if t.DetailMinZoom > 0 {
			l["minzoom"] = t.DetailMinZoom
		}
		layers = append(layers, l)
	}

	return map[string]any{
		"version": 8,
		"name":    s.Name,
		"sprite":  []any{map[string]any{"id": s.Sprite.ID, "url": s.Sprite.URL}},
		"sources": sources,
		"layers":  layers,
	}
}

func (s *Spec) layer(sym Symbol, id, source string) map[string]any {
	l := map[string]any{
		"id":           id,
		"source":       source,
		"source-layer": sym.Table,
		"filter":       []any{"==", []any{"get", "isom_code"}, sym.Code},
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

// StyleJSON renders the style, indented, with a trailing newline.
func (s *Spec) StyleJSON() ([]byte, error) { return marshal(s.Style()) }

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
