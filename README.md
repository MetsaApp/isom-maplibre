# @kortmyre/isom-maplibre

ISOM 2017-2 map styling for MapLibre GL: correct orienteering-map rendering for
vector data carrying ISOM symbol codes.

The package ships a generated MapLibre style (`style.json`) whose every visual
property is derived from the ISOM 2017-2 specification at a fixed 1:10,000
scale: symbol dimensions in mm converted once to CSS px at 96 dpi, the ISOM
color palette, and the ISOM layer stacking order. Around zoom 15 (at mid
European latitudes) the map is true 1:10,000; other zooms magnify uniformly
(x2 per zoom level) rather than restyling, per ISOM's no-dynamic-scaling rule.

## Data contract

The style renders vector data organized into these source layers, where each
feature carries a string `isom_code` property selecting its ISOM symbol:

| Source layer       | Geometry        | `isom_code` values |
|--------------------|-----------------|--------------------|
| `contours`         | lines           | 101.000, 101.001 (index), 102.000, 103.000, 104.000, 105.000 |
| `cliffs`           | lines, polygons | 201.000, 202.000, 206.000 |
| `knolls_points`    | points          | 109.000, 111.000 |
| `vegetation_areas` | polygons        | 401.000, 402.000, 403.000, 404.000, 405.000, 406.000, 407.000, 408.000, 409.000, 410.000, 412.000, 413.000, 415.000 |
| `water`            | lines, polygons | 301.000, 302.000, 304.000, 305.000, 306.000, 308.000 |
| `paths`            | lines           | 502.000, 503.000, 504.000, 505.000, 506.000, 507.000 |
| `manmade`          | lines, polygons | 501.000, 509.000, 510.000, 511.000, 515.000, 516.000, 520.000, 521.001, 529.000 |

## Usage A: vector tiles

Serve MVT tiles with the source-layer names above and use `style.json`
directly. Its sources point at root-relative `/tiles/<name>` URLs and its
sprite at `/sprite/isom`; rewrite those to your tile server, or host the
sprite yourself (the SVGs are in `ICONS` if you want to build a spritesheet).

```ts
import style from "@kortmyre/isom-maplibre/style.json";
```

## Usage B: in-memory GeoJSON

For a single map held client-side (no tile server), `isomGeojsonStyle()`
returns the same symbol stack re-sourced onto per-layer GeoJSON sources, and
`registerIsomIcons()` supplies the pattern/point images at runtime:

```ts
import maplibregl from "maplibre-gl";
import { isomGeojsonStyle, registerIsomIcons, DETAIL_TABLES } from "@kortmyre/isom-maplibre";

const style = isomGeojsonStyle({ contours, water /* FeatureCollections in lng/lat */ });
const map = new maplibregl.Map({ container, style });
registerIsomIcons(map);
```

Coordinates must be lng/lat (EPSG:4326), as with any MapLibre GeoJSON source.
You can also pass nothing and feed data later via
`map.getSource(table).setData(fc)` for each table in `DETAIL_TABLES`.

## Installing

Published to GitHub Packages. Note that GitHub Packages requires an auth token
for `npm install` even on public packages:

```
@kortmyre:registry=https://npm.pkg.github.com
//npm.pkg.github.com/:_authToken=<a token with read:packages>
```

Vendoring the repo (git submodule plus a `file:` dependency) works without any
token and is a legitimate way to consume it.

## License

MIT
