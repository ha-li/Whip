package palette_test

import (
	"math"
	"regexp"
	"strconv"
	"testing"

	"github.com/ha-li/Whip/internal/palette"
)

var hexColor = regexp.MustCompile(`^#[0-9a-f]{6}$`)

// luminance returns the WCAG relative luminance of a "#rrggbb" colour.
func luminance(t *testing.T, hex string) float64 {
	t.Helper()
	var rgb [3]float64
	for i := range rgb {
		v, err := strconv.ParseUint(hex[1+2*i:3+2*i], 16, 8)
		if err != nil {
			t.Fatal(err)
		}
		c := float64(v) / 255
		if c <= 0.04045 {
			rgb[i] = c / 12.92
		} else {
			rgb[i] = math.Pow((c+0.055)/1.055, 2.4)
		}
	}
	return 0.2126*rgb[0] + 0.7152*rgb[1] + 0.0722*rgb[2]
}

func TestColors(t *testing.T) {
	t.Parallel()

	colors := palette.New().Colors()
	if len(colors) == 0 {
		t.Fatal("Colors() is empty")
	}
	seen := map[string]bool{}
	for _, c := range colors {
		t.Run(c.Name, func(t *testing.T) {
			t.Parallel()
			if c.Name == "" || c.Label == "" {
				t.Errorf("%+v has no name or label", c)
			}
			if !hexColor.MatchString(c.Background) || !hexColor.MatchString(c.Text) {
				t.Fatalf("%+v: colours must be lower-case #rrggbb", c)
			}
			hi, lo := luminance(t, c.Background), luminance(t, c.Text)
			if hi < lo {
				hi, lo = lo, hi
			}
			if ratio := (hi + 0.05) / (lo + 0.05); ratio < 4.5 {
				t.Errorf("text on background contrast = %.2f, want at least 4.5", ratio)
			}
		})
		if seen[c.Name] {
			t.Errorf("colour %q appears twice", c.Name)
		}
		seen[c.Name] = true
	}
}

func TestLookup(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		lookup string
		found  bool
	}{
		{"known", "blue", true},
		{"unknown", "chartreuse", false},
		{"empty", "", false},
		{"wrong case", "Blue", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c, ok := palette.Lookup(tt.lookup)
			if ok != tt.found || (ok && c.Name != tt.lookup) {
				t.Errorf("Lookup(%q) = %+v, %v; want found %v", tt.lookup, c, ok, tt.found)
			}
		})
	}
}

func TestAllIsACopy(t *testing.T) {
	t.Parallel()

	all := palette.All()
	all[0].Name = "changed"
	if palette.All()[0].Name == "changed" {
		t.Error("changing All()'s result changed the palette")
	}
}
