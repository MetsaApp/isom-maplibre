package isomstyle

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// rasterAlong is the length in CSS px of a line-raster tile along its lines;
// any value works since the lines run edge to edge.
const rasterAlong = 4

// Icons returns every image as SVG, keyed by its full style id ("isom:111").
// Images are sized in CSS px at the map scale; canvases are whole pixels so a
// renderer never resamples them.
func (s *Spec) Icons() map[string]string {
	out := map[string]string{}
	for key, img := range s.Images {
		out[s.imageID(key)] = s.svg(img)
	}
	return out
}

// IconsJSON renders Icons as an indented JSON object.
func (s *Spec) IconsJSON() ([]byte, error) { return marshal(s.Icons()) }

func (s *Spec) svg(img Image) string {
	px := s.Scale.Px
	switch {
	case img.LineRaster != nil:
		r := img.LineRaster
		n, tile := rasterTile(px(r.Spacing))
		step := tile / float64(n)
		var d strings.Builder
		for i := range n {
			at := f((float64(i) + 0.5) * step)
			if r.Angle == 0 {
				fmt.Fprintf(&d, "M0 %sH%d", at, rasterAlong)
			} else {
				fmt.Fprintf(&d, "M%s 0V%d", at, rasterAlong)
			}
		}
		w, h := int(tile), rasterAlong
		if r.Angle == 0 {
			w, h = h, w
		}
		return svgDoc(w, h, fmt.Sprintf(`<path d=%q fill="none" stroke=%q stroke-width="%s"/>`,
			d.String(), r.Color, f(px(r.Width))))
	case img.HalfCircle != nil:
		c := img.HalfCircle
		sw, outer := px(c.Width), px(c.Diameter)
		r := (outer - sw) / 2
		w, h := int(math.Ceil(outer)), int(math.Ceil(r+sw))
		cx, y := float64(w)/2, (float64(h)-r-sw)/2+sw/2
		return svgDoc(w, h, fmt.Sprintf(`<path d="M%s %sA%s %s 0 0 0 %s %s" fill="none" stroke=%q stroke-width="%s"/>`,
			f(cx-r), f(y), f(r), f(r), f(cx+r), f(y), c.Color, f(sw)))
	default:
		t := img.Tick
		length, sw := px(t.Length), px(t.Width)
		w, h := int(math.Ceil(length)), int(math.Ceil(sw))
		x0, y := (float64(w)-length)/2, float64(h)/2
		return svgDoc(w, h, fmt.Sprintf(`<path d="M%s %sH%s" stroke=%q stroke-width="%s"/>`,
			f(x0), f(y), f(x0+length), t.Color, f(sw)))
	}
}

// rasterTile picks the smallest number of line periods whose total length is
// within 0.2% of a whole pixel, so the pattern tiles seamlessly while the
// spacing stays true to the spec.
func rasterTile(spacing float64) (int, float64) {
	for n := 1; n <= 64; n++ {
		total := float64(n) * spacing
		if math.Abs(total-math.Round(total)) <= 0.002*total {
			return n, math.Round(total)
		}
	}
	return 1, math.Max(1, math.Round(spacing))
}

func svgDoc(w, h int, body string) string {
	return fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d">%s</svg>`,
		w, h, w, h, body)
}

func f(v float64) string { return strconv.FormatFloat(round(v, 3), 'f', -1, 64) }
