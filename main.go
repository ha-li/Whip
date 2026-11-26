package main

import (
	"embed"
	"log"

	"github.com/ha-li/Whip/internal/app"
	"github.com/ha-li/Whip/internal/app/version"
)

// The built frontend, embedded into the binary. It has to live in package
// main: go:embed can only reach files at or below its own package directory.
//
//go:embed all:frontend/dist
var assets embed.FS

func main() {
	v, err := version.Current()
	if err != nil {
		log.Fatal(err)
	}
	a, err := app.New(assets, v)
	if err != nil {
		log.Fatal(err)
	}
	if err := a.Run(); err != nil {
		log.Fatal(err)
	}
}
