package isomstyle

import (
	"encoding/json"
	"fmt"
	"math"
)

// tileSources is every vector-tile source the style reads: the seven detail tables
// (z13-16) + the combined overview function source (z10-12).
//
//nolint:gochecknoglobals // fixed config mirror
var tileSources = []string{
	"contours", "cliffs", "knolls_points", "vegetation_areas", "water", "paths", "manmade",
	"overview",
}

// detailMinZoom / overviewMaxZoom are layer visibility bounds. They stop the
// overview source from overzoom-rendering underneath detail tiles at z13+.
const (
	detailMinZoom   = 13
	overviewMaxZoom = 13
	// overviewMinZoom mirrors the tile server's overview source (z10): below it
	// nothing of the map itself is served, which is where the coverage patch
	// stands in for it.
	overviewMinZoom = 10
)

// lockZoom is the web-mercator zoom where the map is true 1:10,000 at Danish
// latitude (2.6458 m/px at ~56°N, ISOM 2017-2 §1.2). At or below it every
// dimension is a constant; above it all dimensions magnify uniformly by
// 2^(z-lockZoom) — the equivalent of §8's blessed browser magnification, so
// relative ISOM metrics are preserved while zoomed in.
const (
	lockZoom  = 15.0
	maxStyleZ = 22.0
	magAtMaxZ = 128.0 // 2^(22-15)
)

// zoomScaled wraps a pixel dimension in the canonical magnification expression:
// constant v up to lockZoom, then exponential base-2 growth (exact ×2 per zoom
// level). line-dasharray needs no wrapping — it is measured in line widths.
func zoomScaled(v float64) any {
	return []any{
		"interpolate", []any{"exponential", 2}, []any{"zoom"},
		lockZoom, v, maxStyleZ, round2(v * magAtMaxZ),
	}
}

// BuildStyle returns the complete MapLibre style as a JSON-marshalable tree.
// Source URLs are root-relative; the viewer resolves them against its origin.
func BuildStyle() map[string]any {
	sources := map[string]any{}
	for _, s := range tileSources {
		sources[s] = map[string]any{"type": "vector", "url": "/tiles/" + s}
	}
	// The CARTO vector basemap (navigation context outside coverage) is merged
	// in by the viewer at load time (viewer.js) so this file stays
	// self-contained and offline-usable. The white background layer below is
	// dropped during that merge.

	// Job coverage (white base): opaque paper under every generated area so the
	// basemap never bleeds through gaps between feature fills (405 forest is
	// only emitted where KP classified it; coverage is the union of job areas).
	sources["coverage"] = map[string]any{"type": "vector", "url": "/tiles/coverage"}

	layers := []any{
		map[string]any{
			"id":    "background",
			"type":  "background",
			"paint": map[string]any{"background-color": White},
		},
		map[string]any{
			"id":           "coverage-base",
			"type":         "fill",
			"source":       "coverage",
			"source-layer": "coverage",
			// White paper from the first zoom that renders map data — the overview
			// stack, not just the detail one — so the basemap's grey forest never
			// bleeds through the gaps in a low-fidelity render. Below that nothing
			// is drawn on top and a white patch on a white-ish basemap is invisible
			// anyway; the coverage patch takes over there.
			"minzoom": float64(overviewMinZoom),
			"paint":   map[string]any{"fill-color": White},
		},
		// Zoomed out, coverage is the answer to "where do maps exist?", so it is
		// drawn as a visible patch rather than as paper. It stops where the
		// overview render begins: a brown wash over the real map only makes it
		// look muddy, and by then the map itself answers the question.
		map[string]any{
			"id":           "coverage-overview",
			"type":         "fill",
			"source":       "coverage",
			"source-layer": "coverage",
			"maxzoom":      float64(overviewMinZoom),
			// ISOM brown at low opacity: the same colour the dashboard already
			// uses for generated areas, so it needs no new palette entry.
			"paint": map[string]any{
				"fill-color":   Brown,
				"fill-opacity": 0.35,
			},
		},
		map[string]any{
			"id":           "coverage-overview-line",
			"type":         "line",
			"source":       "coverage",
			"source-layer": "coverage",
			"maxzoom":      float64(overviewMinZoom),
			"paint": map[string]any{
				"line-color": Brown,
				"line-width": 1.2,
			},
		},
	}
	// Overview first (it only renders below z13, i.e. visually alone), then the
	// detail stack; within each pass, Stack order = §3.2 bottom → top.
	for i, s := range Stack {
		if overviewTables[s.table] && s.kind != "symbol" && s.kind != "circle" {
			layers = append(layers, layerFor(s, fmt.Sprintf("ov%02d-%s", i, s.code), "overview", 0, overviewMaxZoom))
		}
	}
	for i, s := range Stack {
		layers = append(layers, layerFor(s, fmt.Sprintf("d%02d-%s", i, s.code), s.table, detailMinZoom, 0))
	}

	return map[string]any{
		"version": 8,
		"name":    "ISOM 2017-2 (1:10,000)",
		// Multi-sprite form: coexists with the basemap's own sprite after the
		// viewer merge; icon-image references use the "isom:" prefix.
		"sprite":  []any{map[string]any{"id": "isom", "url": "/sprite/isom"}},
		"sources": sources,
		"layers":  layers,
	}
}

func layerFor(s sym, id, source string, minzoom, maxzoom int) map[string]any {
	l := map[string]any{
		"id":           id,
		"source":       source,
		"source-layer": s.table,
		"filter":       []any{"==", []any{"get", "isom_code"}, s.code},
	}
	if minzoom > 0 {
		l["minzoom"] = minzoom
	}
	if maxzoom > 0 {
		l["maxzoom"] = maxzoom
	}
	switch s.kind {
	case "fill":
		l["type"] = "fill"
		if s.icon != "" {
			// ponytail: fill-pattern can't zoom-magnify — raster density stays fixed past lockZoom
			l["paint"] = map[string]any{"fill-pattern": "isom:" + s.icon}
		} else {
			l["paint"] = map[string]any{"fill-color": s.color}
		}
	case "line":
		l["type"] = "line"
		paint := map[string]any{"line-color": s.color, "line-width": Px(s.wMM)}
		if len(s.dashMM) == 2 {
			// line-dasharray is in units of line width.
			w := Px(s.wMM)
			paint["line-dasharray"] = []any{round2(Px(s.dashMM[0]) / w), round2(Px(s.dashMM[1]) / w)}
		}
		l["paint"] = paint
		l["layout"] = map[string]any{"line-cap": "butt", "line-join": "round"}
	case "circle":
		l["type"] = "circle"
		l["paint"] = map[string]any{"circle-color": s.color, "circle-radius": zoomScaled(round2(Px(s.diaMM) / 2))}
	case "symbol":
		l["type"] = "symbol"
		layout := map[string]any{
			"icon-image":              "isom:" + s.icon,
			"icon-size":               zoomScaled(1),
			"icon-allow-overlap":      true,
			"icon-ignore-placement":   true,
			"icon-rotation-alignment": "map",
		}
		if s.alongLine {
			// The feature is a short line whose direction is the symbol's bearing.
			// symbol-placement "line" makes MapLibre rotate the glyph to follow it;
			// the spacing is larger than any such feature so exactly one is drawn.
			layout["symbol-placement"] = "line-center"
		}
		l["layout"] = layout
	}
	return l
}

func round2(v float64) float64 { return math.Round(v*100) / 100 }

// JSON renders the style, indented, trailing newline.
func JSON() ([]byte, error) {
	b, err := json.MarshalIndent(BuildStyle(), "", "  ")
	if err != nil {
		return nil, err
	}
	return append(b, '\n'), nil
}
