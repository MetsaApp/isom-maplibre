// Pattern and point-symbol images the style references as "isom:<id>".
// Sized in CSS px at 1:10,000; register them with registerIsomIcons().
export const ICONS: Record<string, string> = {
  // ISOM 111 small depression: brown half-circle opening up, north-oriented
  // (1.2 mm wide, 0.27 mm stroke per the ISOM 2017-2 dimension tables).
  "isom:111": `<svg xmlns="http://www.w3.org/2000/svg" width="5" height="3" viewBox="0 0 5 3">
  <path d="M 0.51 0.6 A 1.99 1.99 0 0 0 4.49 0.6"
        fill="none" stroke="#D15C00" stroke-width="1.02"/>
</svg>`,
  // ISOM 308 marsh: horizontal blue line raster. Rendered coarser than the
  // printed spec (0.15 mm lines at 0.6 mm spacing) for screen legibility;
  // the true raster reads as a flat tint at 96 dpi. Tile of 5 lines, seamless.
  "isom:308": `<svg xmlns="http://www.w3.org/2000/svg" width="17" height="17" viewBox="0 0 17 17">
  <path d="M0 1.7H17M0 5.1H17M0 8.5H17M0 11.9H17M0 15.3H17"
        fill="none" stroke="#00FFFF" stroke-width="0.85"/>
</svg>`,
  // ISOM 407 slow running, good visibility: green vertical line raster,
  // north-oriented. One 0.7 px line on a 5 px tile so the pattern stays
  // seamless after rasterization.
  "isom:407": `<svg xmlns="http://www.w3.org/2000/svg" width="5" height="8" viewBox="0 0 5 8">
  <path d="M2.5 0V8" fill="none" stroke="#3DFF17" stroke-width="0.7"/>
</svg>`,
  // ISOM 409 walk, good visibility: like 407 but denser and heavier
  // (two 0.8 px lines per 5 px tile).
  "isom:409": `<svg xmlns="http://www.w3.org/2000/svg" width="5" height="8" viewBox="0 0 5 8">
  <path d="M1.25 0V8M3.75 0V8" fill="none" stroke="#3DFF17" stroke-width="0.8"/>
</svg>`,
  // ISOM 101 slope line: the tick on the lower side of a contour. Part of
  // symbol 101, so it carries contour brown at contour width. Drawn along +x;
  // the style places it with symbol-placement "line" on 2-point tick
  // geometries so MapLibre rotates it to run from the contour inwards.
  "isom:slope": `<svg xmlns="http://www.w3.org/2000/svg" width="2.268" height="0.794" viewBox="0 0 2.268 0.794">
  <line x1="0" y1="0.397" x2="2.268" y2="0.397"
        stroke="#D15C00" stroke-width="0.794"/>
</svg>`,
};
