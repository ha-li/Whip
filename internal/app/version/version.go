// Package version holds Whip's semantic version (https://semver.org):
// MAJOR.MINOR.PATCH with an optional pre-release label, e.g. "1.4.2" or
// "0.1.2-alpha".
package version

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// current is the app version. It defaults to Default; release builds
// override it at link time, e.g.:
//
//	go build -ldflags "-X github.com/ha-li/Whip/internal/app/version.current=0.2.0-beta"
//
// (make dmg / make distribute VERSION=... does this for you.)
var current = Default

// Default is the version reported by builds that don't set one.
const Default = "0.0.1"

// Current returns the app version this binary was built with. It fails only
// if the build injected a malformed version.
func Current() (Version, error) {
	return Parse(current)
}

// Version is a parsed semantic version.
type Version struct {
	Major, Minor, Patch int
	// Prerelease is the part after "-", e.g. "alpha" or "rc.1"; empty for a release.
	Prerelease string
	// Build is the part after "+", e.g. "20261125"; empty if not set.
	Build string
}

// semverPattern is the official SemVer 2.0.0 regex: no leading zeros in
// numeric parts; pre-release and build parts are dot-separated identifiers.
var semverPattern = regexp.MustCompile(`^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)` +
	`(?:-((?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?` +
	`(?:\+([0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?$`)

// Parse parses s (surrounding whitespace is ignored; a leading "v" is not allowed).
func Parse(s string) (Version, error) {
	s = strings.TrimSpace(s)
	m := semverPattern.FindStringSubmatch(s)
	if m == nil {
		return Version{}, fmt.Errorf("version %q is not valid semver (want MAJOR.MINOR.PATCH, optionally -prerelease)", s)
	}
	var v Version
	var err error
	for i, dst := range []*int{&v.Major, &v.Minor, &v.Patch} {
		if *dst, err = strconv.Atoi(m[i+1]); err != nil {
			return Version{}, fmt.Errorf("version %q: %w", s, err)
		}
	}
	v.Prerelease, v.Build = m[4], m[5]
	return v, nil
}

// MustParse is like Parse but panics on an invalid version. Use it only for
// versions fixed at build time.
func MustParse(s string) Version {
	v, err := Parse(s)
	if err != nil {
		panic(err)
	}
	return v
}

// String returns the full version, e.g. "0.1.2-alpha".
func (v Version) String() string {
	s := v.Core()
	if v.Prerelease != "" {
		s += "-" + v.Prerelease
	}
	if v.Build != "" {
		s += "+" + v.Build
	}
	return s
}

// Core returns just MAJOR.MINOR.PATCH, e.g. "0.1.2". macOS Info.plist version
// fields only accept this form.
func (v Version) Core() string {
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}

// IsPrerelease reports whether v is a pre-release such as an alpha or beta.
func (v Version) IsPrerelease() bool {
	return v.Prerelease != ""
}
