// Smallest check that fails if the generated styles, the loader or the icon
// set break.
import { readFileSync } from "node:fs";
import { isomGeojsonStyle, ICONS, DETAIL_TABLES } from "../dist/index.js";

const assert = (cond, msg) => { if (!cond) { console.error("FAIL:", msg); process.exit(1); } };
const tiled = JSON.parse(readFileSync(new URL("../dist/style.json", import.meta.url)));

const contours = { type: "FeatureCollection", features: [] };
const s = isomGeojsonStyle({ contours });
assert(s.sources.contours.data === contours, "caller data not attached");
assert(isomGeojsonStyle().sources.contours.data !== contours, "loader mutated the shared style");
assert(DETAIL_TABLES.join() === tiled.metadata["isom:tables"].join(), "DETAIL_TABLES differs from the definition");
assert(DETAIL_TABLES.every((t) => s.sources[t]?.type === "geojson"), "missing geojson source");
assert(s.layers[0].type === "background", "background layer missing");
assert(s.layers.slice(1).every((l) => l.metadata?.["isom:pass"] === "detail" && l.metadata["isom:code"]),
  "non-detail or untagged layer");
assert(s.layers.some((l) => l.source === "contours"), "no contour layers");
assert(s.layers.every((l) => !("source-layer" in l) && !("minzoom" in l)), "source-layer or minzoom leaked");
assert(!("sprite" in s), "sprite reference leaked");
assert(tiled.layers.slice(1).every((l) => l.metadata?.["isom:pass"]), "untagged tile style layer");
for (const id of ["isom:111", "isom:308", "isom:407", "isom:409", "isom:slope"]) {
  assert(ICONS[id]?.includes("<svg"), `missing icon ${id}`);
}
console.log("ok:", s.layers.length, "layers,", Object.keys(s.sources).length, "sources");
