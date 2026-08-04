// Package isomstyle generates the ISOM 2017-2 MapLibre style (src/style.json).
// Single source of truth: symbol dimensions in mm @1:15,000 (ISOM 2017-2 §5),
// converted once (§1.1: ×1.5 to 1:10,000, ×3.779528 mm→CSS px @96 dpi), the §3.1
// palette, and the §3.2 layer stack. Every visual property is a constant — no
// zoom expressions (§1's no-dynamic-scaling rule; z≈15 at DK latitude is true
// 1:10,000, other zooms are browsing aids).
//
//go:generate go run ./cmd/genstyle src/style.json
package isomstyle

import "math"

// PxPerMM15 converts an ISOM dimension in mm @1:15,000 to CSS px (§1.1).
const PxPerMM15 = 1.5 * 3.779528 // 5.669291

// Px converts mm @1:15k to CSS px, rounded to 3 decimals.
func Px(mm float64) float64 {
	return math.Round(mm*PxPerMM15*1000) / 1000
}

// §3.1 canonical sRGB palette. Do not introduce colours outside this table.
const (
	Black    = "#000000"
	Black65  = "#595959"
	Black50  = "#808080"
	Black25  = "#BFBFBF"
	Black20  = "#CCCCCC"
	Brown    = "#D15C00"
	Brown50  = "#E8AD80"
	Yellow   = "#FFBA36"
	Yellow75 = "#FFCB68"
	Yellow50 = "#FFDD9B"
	Yellow35 = "#FFE7B9"
	Green    = "#3DFF17"
	Green60  = "#8BFF74"
	Green30  = "#C5FFB9"
	Blue     = "#00FFFF"
	Blue70   = "#4DFFFF"
	Blue50   = "#80FFFF"
	Purple   = "#A626FF"
	Olive    = "#9EBA1D"
	White    = "#FFFFFF"
)

// Palette lists every permitted colour (isom-style spec: palette audit).
//
//nolint:gochecknoglobals // fixed spec table
var Palette = []string{
	Black, Black65, Black50, Black25, Black20, Brown, Brown50,
	Yellow, Yellow75, Yellow50, Yellow35, Green, Green60, Green30,
	Blue, Blue70, Blue50, Purple, Olive, White,
}
