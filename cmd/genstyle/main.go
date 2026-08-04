// genstyle writes the generated ISOM MapLibre style to the given path
// (default src/style.json). Run via `go generate ./...`.
package main

import (
	"fmt"
	"os"

	isomstyle "github.com/malpou/isom-maplibre"
)

func main() {
	out := "src/style.json"
	if len(os.Args) > 1 {
		out = os.Args[1]
	}
	b, err := isomstyle.JSON()
	if err == nil {
		err = os.WriteFile(out, b, 0o600)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "genstyle:", err)
		os.Exit(1)
	}
	fmt.Println("wrote", out)
}
