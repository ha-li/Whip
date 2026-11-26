// Package tray adds Whip's icon to the macOS menu bar while the app is running.
package tray

import (
	_ "embed"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// icon is a 44×44 (22pt @2x) black-on-transparent gecko. On macOS it's
// used as a template image, so the system tints it to match light/dark menu bars.
//
//go:embed icon.png
var icon []byte

// New adds the menu bar icon, with name as its tooltip. Clicking it opens a
// menu to bring the app to the front (by calling show) or quit it.
func New(a *application.App, name string, show func()) *application.SystemTray {
	menu := application.NewMenu()
	menu.Add("Show " + name).OnClick(func(*application.Context) {
		show()
	})
	menu.AddSeparator()
	menu.Add("Quit " + name).OnClick(func(*application.Context) {
		a.Quit()
	})

	t := a.SystemTray.New()
	t.SetTemplateIcon(icon)
	t.SetTooltip(name)
	t.SetMenu(menu)
	// With no window attached, a left click opens the menu too.
	return t
}
