package version_test

import (
	"testing"

	"github.com/ha-li/Whip/internal/app/version"
)

func TestParseValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name, in string
		want     version.Version
		core     string
	}{
		{"release", "1.4.2", version.Version{Major: 1, Minor: 4, Patch: 2}, "1.4.2"},
		{"zeros", "0.0.0", version.Version{}, "0.0.0"},
		{"alpha", "0.1.2-alpha", version.Version{Minor: 1, Patch: 2, Prerelease: "alpha"}, "0.1.2"},
		{"numbered prerelease", "2.0.0-rc.1", version.Version{Major: 2, Prerelease: "rc.1"}, "2.0.0"},
		{"build metadata", "1.0.0-beta+20261125", version.Version{Major: 1, Prerelease: "beta", Build: "20261125"}, "1.0.0"},
		{"surrounding whitespace", " 1.2.3\n", version.Version{Major: 1, Minor: 2, Patch: 3}, "1.2.3"},
		{"large numbers", "10.20.30", version.Version{Major: 10, Minor: 20, Patch: 30}, "10.20.30"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := version.Parse(tt.in)
			if err != nil {
				t.Fatalf("Parse(%q) error = %v", tt.in, err)
			}
			if got != tt.want {
				t.Errorf("Parse(%q) = %+v, want %+v", tt.in, got, tt.want)
			}
			if got.Core() != tt.core {
				t.Errorf("Core() = %q, want %q", got.Core(), tt.core)
			}
		})
	}
}

func TestParseInvalid(t *testing.T) {
	t.Parallel()

	for _, in := range []string{
		"", "1", "1.2", "1.2.3.4", "v1.2.3", "01.2.3", "1.02.3", "1.2.03",
		"1.2.3-", "1.2.3-alpha..1", "1.2.3-01", "1.2.3+", "a.b.c", "1.2.3 alpha",
	} {
		t.Run(in, func(t *testing.T) {
			t.Parallel()
			if v, err := version.Parse(in); err == nil {
				t.Errorf("Parse(%q) = %+v, want error", in, v)
			}
		})
	}
}

func TestStringRoundTrip(t *testing.T) {
	t.Parallel()

	for _, in := range []string{"1.4.2", "0.1.2-alpha", "2.0.0-rc.1", "1.0.0-beta+20261125"} {
		t.Run(in, func(t *testing.T) {
			t.Parallel()
			if got := mustParse(t, in).String(); got != in {
				t.Errorf("Parse(%q).String() = %q", in, got)
			}
		})
	}
}

// mustParse parses s, failing the test if it isn't a valid version.
func mustParse(t *testing.T, s string) version.Version {
	t.Helper()
	v, err := version.Parse(s)
	if err != nil {
		t.Fatalf("Parse(%q) error = %v", s, err)
	}
	return v
}

func TestCurrentUnstamped(t *testing.T) {
	t.Parallel()

	// Unit tests aren't built with -ldflags -X, so current keeps its dev value.
	v, err := version.Current()
	if err != nil {
		t.Fatalf("Current() error = %v", err)
	}
	if got, want := v.String(), "0.0.0-dev"; got != want {
		t.Errorf("Current() = %q, want %q", got, want)
	}
}
