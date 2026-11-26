package main

import (
	"embed"
	"log"

	"github.com/ha-li/Whip/internal/app"
)

// The built frontend, embedded into the binary. It has to live in package
// main: go:embed can only reach files at or below its own package directory.
//
//go:embed all:frontend/dist
var assets embed.FS

func main() {
	if err := app.New(assets).Run(); err != nil {
		log.Fatal(err)
	}
}
