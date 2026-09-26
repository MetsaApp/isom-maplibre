// genstyle generates the MapLibre styles and their images from an ISOM style
// definition, by default the one embedded in this module. Each output flag is
// optional; at least one is required.
//
//	genstyle -style style.json -geojson-style style.geojson.json -icons icons.json -sprites sprites/
package main

//go:generate go run . -style ../../src/style.json -geojson-style ../../src/style.geojson.json -icons ../../src/icons.json

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/MetsaApp/isom-maplibre/pkg/isomstyle"
)

func main() {
	specPath := flag.String("spec", "", "style definition to read (default: the embedded isom.yaml)")
	stylePath := flag.String("style", "", "write the vector-tile MapLibre style JSON here")
	geojsonPath := flag.String("geojson-style", "", "write the GeoJSON-source MapLibre style JSON here")
	iconsPath := flag.String("icons", "", "write the image SVGs as a JSON object here")
	spritesDir := flag.String("sprites", "", "write one <key>.svg per image into this directory")
	flag.Parse()
	if err := run(*specPath, *stylePath, *geojsonPath, *iconsPath, *spritesDir); err != nil {
		fmt.Fprintln(os.Stderr, "genstyle:", err)
		os.Exit(1)
	}
}

func run(specPath, stylePath, geojsonPath, iconsPath, spritesDir string) error {
	if stylePath == "" && geojsonPath == "" && iconsPath == "" && spritesDir == "" {
		return errors.New("nothing to write: pass -style, -geojson-style, -icons and/or -sprites")
	}
	var spec *isomstyle.Spec
	var err error
	if specPath == "" {
		spec, err = isomstyle.Default()
	} else {
		spec, err = isomstyle.Load(specPath)
	}
	if err != nil {
		return err
	}
	if stylePath != "" {
		if err := write(stylePath, spec.StyleJSON); err != nil {
			return err
		}
	}
	if geojsonPath != "" {
		if err := write(geojsonPath, spec.GeojsonStyleJSON); err != nil {
			return err
		}
	}
	if iconsPath != "" {
		if err := write(iconsPath, spec.IconsJSON); err != nil {
			return err
		}
	}
	if spritesDir != "" {
		if err := os.MkdirAll(spritesDir, 0o755); err != nil {
			return err
		}
		icons := spec.Icons()
		for key := range spec.Images {
			svg := icons[spec.Sprite.ID+":"+key]
			path := filepath.Join(spritesDir, key+".svg")
			if err := os.WriteFile(path, []byte(svg+"\n"), 0o644); err != nil {
				return err
			}
			fmt.Println("wrote", path)
		}
	}
	return nil
}

func write(path string, render func() ([]byte, error)) error {
	b, err := render()
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, b, 0o644); err != nil {
		return err
	}
	fmt.Println("wrote", path)
	return nil
}
