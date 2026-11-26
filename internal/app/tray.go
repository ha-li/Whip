package app

import (
	_ "embed"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// trayIcon is a 44×44 (22pt @2x) black-on-transparent gecko. On macOS it's
// used as a template image, so the system tints it to match light/dark menu bars.
//
//go:embed tray-icon.png
var trayIcon []byte

// newTray adds Whip's icon to the macOS menu bar (the system tray on Windows
// and Linux). Clicking it opens a menu to bring the main window to the front
// or quit the app.
func newTray(a *application.App, window *application.WebviewWindow) *application.SystemTray {
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
		a.Quit()
	})

	t := a.SystemTray.New()
	t.SetTemplateIcon(trayIcon)
	t.SetTooltip("Whip")
	t.SetMenu(menu)
	// With no window attached, a left click opens the menu too.
	return t
}
