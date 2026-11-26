// Package tray adds Whip's icon to the macOS menu bar (the system tray on
// Windows and Linux) while the app is running.
package tray

import (
	_ "embed"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// icon is a 44×44 (22pt @2x) black-on-transparent gecko. On macOS it's used as
// a template image, so the system tints it to match light/dark menu bars.
//
//go:embed icon.png
var icon []byte

// New adds the menu bar icon. Clicking it opens a menu to bring the main
// window to the front or quit the app.
func New(app *application.App, window *application.WebviewWindow) *application.SystemTray {
	menu := application.NewMenu()
	menu.Add("Show Whip").OnClick(func(*application.Context) {
		if window.IsMinimised() {
			window.UnMinimise()
		}
		window.Show()
		window.Focus()
	})
	menu.AddSeparator()
	menu.Add("Quit Whip").OnClick(func(*application.Context) {
		app.Quit()
	})

	t := app.SystemTray.New()
	t.SetTemplateIcon(icon)
	t.SetTooltip("Whip")
	t.SetMenu(menu)
	// With no window attached, a left click opens the menu too.
	return t
}
