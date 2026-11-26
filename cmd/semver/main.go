// Command semver validates an app version for the build.
//
//	go run ./cmd/semver [-full] VERSION
//
// It prints MAJOR.MINOR.PATCH (or, with -full, the whole version) and exits
// non-zero if VERSION isn't valid semver.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/ha-li/Whip/internal/app/version"
)

func main() {
	full := flag.Bool("full", false, "print the full version, including any -prerelease/+build part")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: semver [-full] VERSION")
		flag.PrintDefaults()
	}
	flag.Parse()
	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(2)
	}

	v, err := version.Parse(flag.Arg(0))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if *full {
		fmt.Println(v.String())
	} else {
		fmt.Println(v.Core())
	}
}
