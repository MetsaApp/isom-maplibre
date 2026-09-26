# isom-maplibre

A [MapLibre GL](https://maplibre.org) style that renders orienteering maps per
[ISOM 2017-2](https://orienteering.sport/iof/mapping/), the IOF International
Specification for Orienteering Maps, at 1:10,000.

Every line width, dash, pattern and symbol is computed from the ISOM dimensions
(mm at 1:15,000, enlarged ×1.5 and converted at 96 dpi), uses the ISOM colours,
and is stacked in the ISOM colour order. The dimensions are cross-checked against
the [OpenOrienteering Mapper](https://www.openorienteering.org/apps/mapper/)
ISOM 2017-2 symbol set.

## Install

The package is published to GitHub Packages, which requires a token for
`npm install` even for public packages. Create a classic personal access token
with `read:packages` and add to your `.npmrc`:

```
@metsaapp:registry=https://npm.pkg.github.com
//npm.pkg.github.com/:_authToken=${GITHUB_TOKEN}
```

```sh
npm install @metsaapp/isom-maplibre maplibre-gl
```

## Usage

### In-memory GeoJSON

`isomGeojsonStyle()` returns the GeoJSON style (one GeoJSON source per table,
no sprite) with your data attached, and `registerIsomIcons()` supplies the
pattern and symbol images at runtime:

```ts
import maplibregl from "maplibre-gl";
import { isomGeojsonStyle, registerIsomIcons } from "@metsaapp/isom-maplibre";

const map = new maplibregl.Map({
  container: "map",
  style: isomGeojsonStyle({ contours, water, paths }), // FeatureCollections, lng/lat
});
registerIsomIcons(map);
```

Tables you leave out start empty; fill them later with
`map.getSource(table).setData(featureCollection)`. `DETAIL_TABLES` lists them.
The style without data is also exported as
`@metsaapp/isom-maplibre/style.geojson.json`.

### Vector tiles

```ts
import style from "@metsaapp/isom-maplibre/style.json";
```

The style reads one vector source per table from `/tiles/<table>` (TileJSON)
and its images from the sprite `/sprite/isom`. Point those URLs at your server
and build the sprite from `ICONS` (SVGs keyed by image id), or pass `-sprites`
to the generator (see [Development](#development)). Two optional sources serve
large tile sets:

- `overview`: one source whose layers carry `vegetation_areas`, `water`,
  `paths` and `manmade`, drawn below zoom 13 in place of the per-table sources.
- `coverage`: polygons outlining where data exists. From zoom 10 they are white
  paper under the map, so a basemap merged underneath never shows through; below
  zoom 10 they are a translucent brown patch marking where maps are.

Every layer except the background carries `metadata` for selecting layers
without parsing ids: `isom:pass` (`detail`, `overview` or `coverage`), and on
symbol layers `isom:code` and `isom:group` (the ISOM colour group). The style's
own `metadata["isom:tables"]` lists the tables in definition order.

## Data

Each feature carries a string `isom_code` property. Codes without a symbol stay
invisible.

| Table              | Geometry        | `isom_code`                                                          |
|--------------------|-----------------|----------------------------------------------------------------------|
| `contours`         | lines           | 101.000, 101.001 (slope line), 102.000, 103.000, 104.000, 105.000    |
| `cliffs`           | lines, polygons | 201.000, 202.000, 206.000                                            |
| `knolls_points`    | points          | 109.000, 111.000                                                     |
| `vegetation_areas` | polygons, lines | 401.000 to 410.000, 412.000, 413.000, 415.000                        |
| `water`            | lines, polygons | 301.000, 302.000, 304.000, 305.000, 306.000, 308.000                 |
| `paths`            | lines           | 502.000 to 507.000                                                   |
| `manmade`          | lines, polygons | 501.000, 509.000, 510.000, 511.000, 515.000, 516.000, 520.000, 521.000, 521.001 (large building: outline and 65% infill), 529.000 |

A 101.001 slope line is a two-point line from the contour downhill; the symbol
takes its bearing from it. Coordinates are lng/lat, as for any MapLibre source.

## Scale

Up to zoom 15 every dimension is constant; above it the whole map magnifies ×2
per zoom level, which keeps ISOM proportions intact. Zoom 15 is true 1:10,000
at about 56° latitude. At latitude φ the true-scale zoom is
`log2(156543.03 · cos φ / 2.6458)`.

## Limitations

MapLibre cannot draw some ISOM ornaments, so these symbols are simplified:

- Plain line without its ornament: 104 tags, 105 dots, 201 tags, 510 pylon
  bars, 515 dots, 516 tags, 529 tick pairs.
- Flat fill without its pattern: 402 and 404 holes, 412 dots, 413 dot rows.
- Dashes are not balanced per feature as ISOM asks.
- The white core of 509 and 511 masks what lies beneath it.
- Fill patterns (308, 407, 409) keep their density above zoom 15.
- The map must be displayed north-up for the north-oriented patterns to be
  correct.

## Development

[`isom.yaml`](isom.yaml) defines the whole style: scale, palette (referenced
through YAML anchors), images, and the symbol stack. It is validated by
[`isom.schema.json`](isom.schema.json), which editors with the YAML language
server pick up automatically; the generator enforces the schema too, plus the
cross-references it cannot express (palette colours, table and image names).
`src/style.json`, `src/style.geojson.json` and `src/icons.json` are generated
from it:

```sh
go generate ./...   # regenerate src/
go test ./...       # schema, ISOM dimension and colour-order checks
npm run build && npm test
```

The generator is also usable directly, for example to write a sprite directory
for a tile server. It reads the definition embedded in the module unless
`-spec` names another file:

```sh
go run github.com/MetsaApp/isom-maplibre/cmd/genstyle@latest \
  -style style.json -geojson-style style.geojson.json -icons icons.json -sprites sprites/
```

Go programs get the parsed definition from `isomstyle.Default()` in
`github.com/MetsaApp/isom-maplibre/pkg/isomstyle` (`Load` and `Parse` take
another one), and render it with `Style`, `GeojsonStyle` and `Icons`.

Commits follow [Conventional Commits](https://www.conventionalcommits.org);
release-please opens the release PR, and merging it tags the release and
publishes the package.

## License

MIT. ISOM is a specification of the International Orienteering Federation;
this project is not affiliated with the IOF.
