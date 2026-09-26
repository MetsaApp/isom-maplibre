// Package isommaplibre embeds the ISOM style definition so Go programs need no
// path to it. It lives at the repository root only because go:embed cannot
// reach parent directories; pkg/isomstyle parses it.
package isommaplibre

import _ "embed"

// Definition is isom.yaml, the whole style: scale, palette, images, symbol stack.
//
//go:embed isom.yaml
var Definition []byte

// Schema is isom.schema.json, the structural rules Definition must satisfy.
//
//go:embed isom.schema.json
var Schema []byte
