package whipapp_test

import (
	"testing"

	"github.com/ha-li/Whip/internal/app/whipapp"
	"github.com/wailsapp/wails/v3/pkg/application"
)

func TestWindowBoundsUsable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		bounds *whipapp.WindowBounds
		want   bool
	}{
		{"none saved", nil, false},
		{"normal", &whipapp.WindowBounds{Width: 1450, Height: 925}, true},
		{"smallest", &whipapp.WindowBounds{Width: whipapp.MinWidth, Height: whipapp.MinHeight}, true},
		{"too narrow", &whipapp.WindowBounds{Width: whipapp.MinWidth - 1, Height: 925}, false},
		{"too short", &whipapp.WindowBounds{Width: 1450, Height: whipapp.MinHeight - 1}, false},
		{"zero", &whipapp.WindowBounds{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.bounds.Usable(); got != tt.want {
				t.Errorf("%+v Usable() = %v, want %v", tt.bounds, got, tt.want)
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
		bounds whipapp.WindowBounds
		want   bool
	}{
		{"inside", whipapp.WindowBounds{X: 100, Y: 100, Width: 800, Height: 600}, true},
		{"overhangs the right edge", whipapp.WindowBounds{X: 1000, Y: 100, Width: 800, Height: 600}, true},
		{"just enough on the left", whipapp.WindowBounds{X: -700, Y: 100, Width: 800, Height: 600}, true},
		{"too little on the left", whipapp.WindowBounds{X: -701, Y: 100, Width: 800, Height: 600}, false},
		{"off the right edge", whipapp.WindowBounds{X: 1500, Y: 100, Width: 800, Height: 600}, false},
		{"below the display", whipapp.WindowBounds{X: 100, Y: 900, Width: 800, Height: 600}, false},
		{"on a display that's gone", whipapp.WindowBounds{X: -2000, Y: -1200, Width: 800, Height: 600}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.bounds.VisibleOn(area); got != tt.want {
				t.Errorf("%+v VisibleOn(%+v) = %v, want %v", tt.bounds, area, got, tt.want)
			}
		})
	}
}
