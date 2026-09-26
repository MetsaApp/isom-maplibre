import type { Map, StyleSpecification } from "maplibre-gl";
import base from "./style.json" with { type: "json" };
import icons from "./icons.json" with { type: "json" };

/** Pattern and point-symbol SVGs keyed by style image id ("isom:111"), sized
 * in CSS px at 1:10,000. registerIsomIcons() rasterizes them for you. */
export const ICONS: Record<string, string> = icons;

/** Source layers the style expects; each feature carries a string `isom_code`
 * property ("401.000", "509.000", ...) selecting the ISOM symbol. */
export const DETAIL_TABLES = [
  "contours",
  "cliffs",
  "knolls_points",
  "vegetation_areas",
  "water",
  "paths",
  "manmade",
] as const;
export type IsomTable = (typeof DETAIL_TABLES)[number];

/** The shipped vector-tile style re-sourced onto per-layer GeoJSON sources.
 * Keeps the white background and the full-detail symbol pass; drops the
 * low-zoom overview pass and coverage layers (tiled-serving concepts), strips
 * minzoom so the map renders at any zoom, and drops the sprite reference
 * (registerIsomIcons supplies the images instead). */
export function isomGeojsonStyle(
  data?: Partial<Record<IsomTable, GeoJSON.FeatureCollection>>,
): StyleSpecification {
  const style = structuredClone(base) as any;
  style.sources = Object.fromEntries(
    DETAIL_TABLES.map((t) => [t, {
      type: "geojson",
      data: data?.[t] ?? { type: "FeatureCollection", features: [] },
    }]),
  );
  style.layers = style.layers
    // Detail layers have source === their table; overview/coverage don't.
    .filter((l: any) => l.type === "background" || DETAIL_TABLES.includes(l.source))
    .map((l: any) => {
      if (l.type === "background") return l;
      const { ["source-layer"]: _sl, minzoom: _mz, ...rest } = l;
      return rest;
    });
  delete style.sprite;
  return style as StyleSpecification;
}

/** Rasterizes the ISOM pattern/symbol images on demand at 2x (fill patterns
 * and point symbols reference them as "isom:<id>"). Call once right after
 * constructing the Map. */
export function registerIsomIcons(map: Map): void {
  const pending = new Set<string>();
  map.on("styleimagemissing", async ({ id }) => {
    const svg = ICONS[id];
    // styleimagemissing refires every frame until the image exists.
    if (!svg || pending.has(id) || map.hasImage(id)) return;
    pending.add(id);
    const img = new Image();
    img.src = "data:image/svg+xml;charset=utf-8," + encodeURIComponent(svg);
    await img.decode();
    const c = document.createElement("canvas");
    c.width = img.width * 2;
    c.height = img.height * 2;
    const ctx = c.getContext("2d")!;
    ctx.drawImage(img, 0, 0, c.width, c.height);
    if (!map.hasImage(id)) {
      map.addImage(id, ctx.getImageData(0, 0, c.width, c.height), { pixelRatio: 2 });
    }
  });
}
