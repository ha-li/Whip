package app

import (
	"sync"

	"github.com/ha-li/Whip/internal/app/whipapp"
	"github.com/ha-li/Whip/internal/config"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

// windowTracker follows the main window's frame and saves it on quit, so the
// next launch reopens the window where it was.
type windowTracker struct {
	settings *config.File[whipapp.Settings]

	mu     sync.Mutex
	bounds *whipapp.WindowBounds // nil until the window moves or resizes
}

// track updates the frame whenever window moves or resizes. Frames while the
// window is minimised, maximised or full screen aren't kept, so Whip reopens
// at its last normal size.
func (t *windowTracker) track(window *application.WebviewWindow) {
	update := func(*application.WindowEvent) {
		if window.IsMinimised() || window.IsMaximised() || window.IsFullscreen() {
			return
		}
		r := window.Bounds()
		t.mu.Lock()
		t.bounds = &whipapp.WindowBounds{X: r.X, Y: r.Y, Width: r.Width, Height: r.Height}
		t.mu.Unlock()
	}
	window.OnWindowEvent(events.Common.WindowDidMove, update)
	window.OnWindowEvent(events.Common.WindowDidResize, update)
}

// save writes the last frame seen to whip.yml. It does nothing if the window
// never moved or resized.
func (t *windowTracker) save() error {
	t.mu.Lock()
	b := t.bounds
	t.mu.Unlock()
	if b == nil {
		return nil
	}
	return t.settings.Update(func(s *whipapp.Settings) { s.Window = b })
}

// keepOnScreen centres window if too little of it is on a display, e.g. when
// it was restored onto a display that has since been disconnected. It checks
// once, the first time the frontend loads: by then the window is on screen
// and Wails can say which display it's on.
func keepOnScreen(window *application.WebviewWindow) {
	var once sync.Once
	window.OnWindowEvent(events.Common.WindowRuntimeReady, func(*application.WindowEvent) {
		once.Do(func() {
			// For a window off every display, this is the main display.
			screen, err := window.GetScreen()
			r := window.Bounds()
			b := whipapp.WindowBounds{X: r.X, Y: r.Y, Width: r.Width, Height: r.Height}
			if err != nil || screen == nil || !b.VisibleOn(screen.WorkArea) {
				window.Center()
			}
		})
	})
}
