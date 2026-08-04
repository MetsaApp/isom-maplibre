package isomstyle

// A sym is one styled ISOM code: its source table, geometry treatment, and the
// §5 dimensions in mm @1:15,000. The emitter turns each into one MapLibre layer.
// Only codes store.Translate can currently emit are listed; anything else stays
// invisible by spec (isom-style: unknown code stays invisible).
type sym struct {
	code   string // stored isom_code (NNN.000)
	table  string // detail source / overview source-layer
	kind   string // "fill" | "line" | "circle" | "symbol"
	color  string
	wMM    float64   // line width mm @1:15k
	dashMM []float64 // dash,gap mm @1:15k (line-dasharray = px/width per MapLibre)
	diaMM  float64   // circle diameter mm @1:15k
	icon   string    // sprite image id for kind "symbol"
	// Place the symbol along its line geometry instead of at a point, so the glyph
	// takes its bearing from the feature. Only meaningful for kind "symbol".
	alongLine bool
}

// Stack is the §3.2 colour order, bottom → top, restricted to what the pipeline
// ingests today (no course purple, no screens/pattern fills yet — flat-fill
// approximations per PRD 3's honest limit, marked below).
//
//nolint:gochecknoglobals // fixed spec table
var Stack = []sym{
	// -- yellow group (§3.2 #13) --
	{code: "401.000", table: "vegetation_areas", kind: "fill", color: Yellow},
	{code: "402.000", table: "vegetation_areas", kind: "fill", color: Yellow}, // ponytail: hole pattern deferred
	{code: "403.000", table: "vegetation_areas", kind: "fill", color: Yellow50},
	{code: "404.000", table: "vegetation_areas", kind: "fill", color: Yellow50}, // ponytail: hole pattern deferred
	{code: "412.000", table: "vegetation_areas", kind: "fill", color: Yellow},   // ponytail: dot grid deferred
	{code: "413.000", table: "vegetation_areas", kind: "fill", color: Yellow50}, // ponytail: green dot rows deferred
	// 520 olive sits in the yellow group (§3.2 #13 lists it there), not with the
	// 5xx black symbols its code number suggests. Olive is the spec's first option;
	// the vertical-black-stripe alternative is a pattern.
	// ponytail: no 0.14 black boundary — ISOM draws it only where the border is
	// clear on the ground, which the source cannot tell us
	{code: "520.000", table: "manmade", kind: "fill", color: Olive},
	// -- white holes (§3.2 #12) --
	{code: "405.000", table: "vegetation_areas", kind: "fill", color: White},
	// -- green screens/fills (§3.2 #9-11, bottom-up: 30 → 60 → 100) --
	{code: "406.000", table: "vegetation_areas", kind: "fill", color: Green30},
	// 407/409 stripe rasters (§5.3): green vertical lines on white, north-oriented —
	// the viewport axis IS north here (ISOM 2017-2 §"Pattern fills").
	{code: "407.000", table: "vegetation_areas", kind: "fill", icon: "407"},
	{code: "408.000", table: "vegetation_areas", kind: "fill", color: Green60},
	{code: "409.000", table: "vegetation_areas", kind: "fill", icon: "409"},
	{code: "410.000", table: "vegetation_areas", kind: "fill", color: Green},
	// -- blue tints (§3.2 #8) --
	{code: "302.000", table: "water", kind: "fill", color: Blue50},
	// 308 marsh: horizontal blue line raster via fill-pattern sprite
	// ponytail: coarser than spec (0.15 @ 0.6 CC vs 0.10 @ 0.3) — spec raster reads flat on screen
	{code: "308.000", table: "water", kind: "fill", icon: "308"},
	// -- black tints (§3.2 #6): 521 alt. rendering (outline + 65% infill, §5.5) --
	{code: "521.001", table: "manmade", kind: "fill", color: Black65},
	// -- brown-50 (§3.2 #7): paved fill + wide-road casing/infill --
	{code: "501.000", table: "manmade", kind: "fill", color: Brown50},
	// 502: edges 0.14 + inner ≥0.3 @15k (§5.5). Casing under infill (adjacent
	// layers) rather than split across colour groups — reads identically.
	{code: "502.000", table: "paths", kind: "line", color: Black, wMM: 0.58},  // casing: 0.3 + 2×0.14
	{code: "502.000", table: "paths", kind: "line", color: Brown50, wMM: 0.3}, // infill
	// -- blue 100% (§3.2 #4) --
	{code: "301.000", table: "water", kind: "fill", color: Blue},
	{code: "304.000", table: "water", kind: "line", color: Blue, wMM: 0.30},
	{code: "305.000", table: "water", kind: "line", color: Blue, wMM: 0.18},
	{code: "306.000", table: "water", kind: "line", color: Blue, wMM: 0.18, dashMM: []float64{1.25, 0.25}},
	// -- brown 100% (§3.2 #3): contours + brown point symbols --
	{code: "101.000", table: "contours", kind: "line", color: Brown, wMM: 0.14},
	// The slope line on a depression: part of 101, carried as 101.001 so it can be
	// styled as a glyph rather than a contour. Its geometry is the 2-point tick the
	// generator places from the contour inwards, so placing the symbol along the line
	// gives MapLibre the bearing — no per-feature rotation attribute needed (and a
	// jsonb attrs column reaches the tile as a string, which a filter cannot read).
	{code: "101.001", table: "contours", kind: "symbol", icon: "slope", alongLine: true},
	{code: "102.000", table: "contours", kind: "line", color: Brown, wMM: 0.25},
	// 103 form line (§5.1): same brown, 0.14, dashed. The generator emits these at
	// half the contour interval and they were ingested from the first run — they
	// were simply missing here, so the viewer dropped what OCAD draws.
	// ponytail: plain dasharray, not the spec's balanced dashes (ISOM 2017-2 §4
	// pre-computed-dash note) — indistinguishable at 1:10,000 until it isn't.
	{code: "103.000", table: "contours", kind: "line", color: Brown, wMM: 0.14, dashMM: []float64{2.0, 0.25}},
	// 104/105 are landforms, so brown with the contours rather than black with the
	// 5xx structures. Both carry ornaments sym cannot express (§5.1), the same
	// deliberate deferral 201 and 516 already carry.
	{code: "104.000", table: "contours", kind: "line", color: Brown, wMM: 0.18}, // ponytail: bank tags deferred
	{code: "105.000", table: "contours", kind: "line", color: Brown, wMM: 0.18}, // ponytail: wall dots deferred
	{code: "109.000", table: "knolls_points", kind: "circle", color: Brown, diaMM: 0.5},
	{code: "111.000", table: "knolls_points", kind: "symbol", icon: "111"},
	// -- black 100% (§3.2 #2) --
	{code: "301.000", table: "water", kind: "line", color: Black, wMM: 0.18},   // bank line
	{code: "501.000", table: "manmade", kind: "line", color: Black, wMM: 0.14}, // paved frame
	{code: "503.000", table: "paths", kind: "line", color: Black, wMM: 0.35},
	{code: "504.000", table: "paths", kind: "line", color: Black, wMM: 0.35, dashMM: []float64{3.0, 0.25}},
	{code: "505.000", table: "paths", kind: "line", color: Black, wMM: 0.25, dashMM: []float64{2.0, 0.25}},
	{code: "506.000", table: "paths", kind: "line", color: Black, wMM: 0.18, dashMM: []float64{1.0, 0.25}},
	// ponytail: 507's double-dash rendered as plain dash
	{code: "507.000", table: "paths", kind: "line", color: Black, wMM: 0.18, dashMM: []float64{0.8, 0.25}},
	// 509 railway: black envelope + dashed white core = §5.5 double line with
	// dashes (base/core pair like 502). Envelope 0.45 = core 0.35 + borders;
	// core 0.25 = window left after the 0.10 borders overlap the core edges.
	// ponytail: white core masks underlay — offset rails if it ever matters
	{code: "509.000", table: "manmade", kind: "line", color: Black, wMM: 0.45},
	{code: "509.000", table: "manmade", kind: "line", color: White, wMM: 0.25, dashMM: []float64{1.0, 1.5}},
	{code: "510.000", table: "manmade", kind: "line", color: Black, wMM: 0.14}, // ponytail: pylon bars deferred
	{code: "516.000", table: "manmade", kind: "line", color: Black, wMM: 0.14}, // ponytail: fence tags deferred
	// 415 is a plain black line in §5.4 — no ornament to defer. It routes to
	// vegetation_areas because tableFor keys on the leading 4, not the geometry.
	{code: "415.000", table: "vegetation_areas", kind: "line", color: Black, wMM: 0.14},
	// 511 double line (§5.5: two 0.14 lines, gap >= 0.3): the casing/core idiom 502
	// and 509 already use, so this one gets its real drawing rather than a deferral.
	{code: "511.000", table: "manmade", kind: "line", color: Black, wMM: 0.58}, // 0.3 gap + 2x0.14
	{code: "511.000", table: "manmade", kind: "line", color: White, wMM: 0.30}, // the gap
	{code: "515.000", table: "manmade", kind: "line", color: Black, wMM: 0.35}, // ponytail: wall dots deferred
	{code: "529.000", table: "manmade", kind: "line", color: Black, wMM: 0.25}, // ponytail: tick pairs deferred
	{
		code:  "521.001",
		table: "manmade",
		kind:  "line",
		color: Black,
		wMM:   0.2,
	}, // building outline (infill in black-tints)
	{code: "201.000", table: "cliffs", kind: "line", color: Black, wMM: 0.35}, // ponytail: cliff tags deferred
	{code: "202.000", table: "cliffs", kind: "line", color: Black, wMM: 0.25},
	// 206 gigantic boulder / rock pillar / massive cliff: black plan shape, derived
	// from cliff linework at ingest (isom206-massive-cliffs). Above 201/202 so a face
	// reads over its own edges (design D4).
	{code: "206.000", table: "cliffs", kind: "fill", color: Black},
}

// overviewTables are the source-layers the overview function source emits
// combined at low zoom; must match what the tile server publishes there.
//
//nolint:gochecknoglobals // fixed config mirror
var overviewTables = map[string]bool{
	"vegetation_areas": true, "water": true, "paths": true, "manmade": true,
}
