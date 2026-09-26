import type { GeoJSONSourceSpecification, Map, StyleSpecification } from "maplibre-gl";
import geojsonStyle from "./style.geojson.json" with { type: "json" };
import icons from "./icons.json" with { type: "json" };

/** Pattern and point-symbol SVGs keyed by style image id ("isom:111"), sized
 * in CSS px at 1:10,000. registerIsomIcons() rasterizes them for you. */
export const ICONS: Record<string, string> = icons;

export type IsomTable = keyof typeof geojsonStyle.sources;

/** Source layers the style expects, in definition order; each feature carries
 * a string `isom_code` property ("401.000", "509.000", ...) selecting the ISOM
 * symbol. */
export const DETAIL_TABLES = geojsonStyle.metadata["isom:tables"] as readonly IsomTable[];

/** The generated GeoJSON style (one source per table, the full-detail symbol
 * pass at every zoom, no sprite: registerIsomIcons supplies the images) with
 * the given FeatureCollections attached. Tables left out start empty. */
export function isomGeojsonStyle(
  data?: Partial<Record<IsomTable, GeoJSON.FeatureCollection>>,
): StyleSpecification {
  const style = structuredClone(geojsonStyle) as unknown as StyleSpecification;
  for (const t of DETAIL_TABLES) {
    const fc = data?.[t];
    if (fc) (style.sources[t] as GeoJSONSourceSpecification).data = fc;
  }
  return style;
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
