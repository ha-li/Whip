package app

import "github.com/wailsapp/wails/v3/pkg/application"

// windowBounds is the main window's frame in points, with (0, 0) at the
// top-left of the primary display, as Wails reports it.
type windowBounds struct {
	X      int `yaml:"x"`
	Y      int `yaml:"y"`
	Width  int `yaml:"width"`
	Height int `yaml:"height"`
}

// The main window's default and smallest restored sizes.
const (
	defaultWidth  = 1450
	defaultHeight = 925
	minWidth      = 480
	minHeight     = 320
)

// usable reports whether b is a size worth restoring. Anything smaller is
// treated as a damaged file and ignored.
func (b *windowBounds) usable() bool {
	return b != nil && b.Width >= minWidth && b.Height >= minHeight
}

// minVisible is how much of a restored window, in points on each axis, must
// be on a display for it to count as visible: enough to grab its title bar.
const minVisible = 100

// visibleOn reports whether enough of b lies inside area (a display's work
// area) for the user to see and move the window.
func (b windowBounds) visibleOn(area application.Rect) bool {
	w := min(b.X+b.Width, area.X+area.Width) - max(b.X, area.X)
	h := min(b.Y+b.Height, area.Y+area.Height) - max(b.Y, area.Y)
	return w >= minVisible && h >= minVisible
}
