// Smallest check that fails if the transform or icon set breaks.
import { isomGeojsonStyle, ICONS, DETAIL_TABLES } from "../dist/index.js";

const s = isomGeojsonStyle();
const assert = (cond, msg) => { if (!cond) { console.error("FAIL:", msg); process.exit(1); } };

assert(s.layers.some((l) => l.source === "contours"), "no contour layers survived");
assert(s.layers.every((l) => !("source-layer" in l) && !("minzoom" in l)), "source-layer or minzoom leaked");
assert(s.layers[0].type === "background", "background layer missing");
assert(DETAIL_TABLES.every((t) => s.sources[t]?.type === "geojson"), "missing geojson source");
assert(!("sprite" in s), "sprite reference leaked");
for (const id of ["isom:111", "isom:308", "isom:407", "isom:409", "isom:slope"]) {
  assert(ICONS[id]?.includes("<svg"), `missing icon ${id}`);
}
console.log("ok:", s.layers.length, "layers,", Object.keys(s.sources).length, "sources");
