// Package docs содержит сгенерированную единую спецификацию публичного API.
package docs

import "embed"

// Files содержит swagger.json и swagger.yaml, созданные командой task generate:swagger.
//
//go:embed swagger.json swagger.yaml
var Files embed.FS
