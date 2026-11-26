package app

import (
	"testing"

	"github.com/wailsapp/wails/v3/pkg/application"
)

func TestWindowBoundsUsable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		bounds *windowBounds
		want   bool
	}{
		{"none saved", nil, false},
		{"normal", &windowBounds{Width: 1450, Height: 925}, true},
		{"smallest", &windowBounds{Width: minWidth, Height: minHeight}, true},
		{"too narrow", &windowBounds{Width: minWidth - 1, Height: 925}, false},
		{"too short", &windowBounds{Width: 1450, Height: minHeight - 1}, false},
		{"zero", &windowBounds{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.bounds.usable(); got != tt.want {
				t.Errorf("%+v usable() = %v, want %v", tt.bounds, got, tt.want)
			}
		})
	}
}

func TestWindowBoundsVisibleOn(t *testing.T) {
	t.Parallel()

	// A 1440×875 work area below a 25-point menu bar.
	area := application.Rect{X: 0, Y: 25, Width: 1440, Height: 875}
	tests := []struct {
		name   string
		bounds windowBounds
		want   bool
	}{
		{"inside", windowBounds{X: 100, Y: 100, Width: 800, Height: 600}, true},
		{"overhangs the right edge", windowBounds{X: 1000, Y: 100, Width: 800, Height: 600}, true},
		{"just enough on the left", windowBounds{X: -700, Y: 100, Width: 800, Height: 600}, true},
		{"too little on the left", windowBounds{X: -701, Y: 100, Width: 800, Height: 600}, false},
		{"off the right edge", windowBounds{X: 1500, Y: 100, Width: 800, Height: 600}, false},
		{"below the display", windowBounds{X: 100, Y: 900, Width: 800, Height: 600}, false},
		{"on a display that's gone", windowBounds{X: -2000, Y: -1200, Width: 800, Height: 600}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.bounds.visibleOn(area); got != tt.want {
				t.Errorf("%+v visibleOn(%+v) = %v, want %v", tt.bounds, area, got, tt.want)
			}
		})
	}
}
