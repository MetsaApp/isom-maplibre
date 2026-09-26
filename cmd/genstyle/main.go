// genstyle generates the MapLibre style and its images from an ISOM style
// definition. Each output flag is optional; at least one is required.
//
//	genstyle -spec isom.yaml -style style.json -icons icons.json -sprites sprites/
package main

//go:generate go run . -spec ../../isom.yaml -style ../../src/style.json -icons ../../src/icons.json

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/malpou/isom-maplibre/pkg/isomstyle"
)

func main() {
	specPath := flag.String("spec", "isom.yaml", "style definition to read")
	stylePath := flag.String("style", "", "write the MapLibre style JSON here")
	iconsPath := flag.String("icons", "", "write the image SVGs as a JSON object here")
	spritesDir := flag.String("sprites", "", "write one <key>.svg per image into this directory")
	flag.Parse()
	if err := run(*specPath, *stylePath, *iconsPath, *spritesDir); err != nil {
		fmt.Fprintln(os.Stderr, "genstyle:", err)
		os.Exit(1)
	}
}

func run(specPath, stylePath, iconsPath, spritesDir string) error {
	if stylePath == "" && iconsPath == "" && spritesDir == "" {
		return errors.New("nothing to write: pass -style, -icons and/or -sprites")
	}
	spec, err := isomstyle.Load(specPath)
	if err != nil {
		return err
	}
	if stylePath != "" {
		if err := write(stylePath, spec.StyleJSON); err != nil {
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
