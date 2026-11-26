// Package app assembles the Whip application: the Wails app, its services,
// the main window and the menu bar icon.
package app

import (
	"io/fs"

	"github.com/ha-li/Whip/internal/app/tray"
	"github.com/ha-li/Whip/internal/app/version"
	"github.com/ha-li/Whip/internal/config"
	"github.com/ha-li/Whip/internal/stocks"
	"github.com/ha-li/Whip/internal/user"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// New creates the Wails application, its main window and the menu bar icon.
// assets holds the built frontend (frontend/dist), embedded by package main
// because go:embed can't reach files outside its own package directory.
// v is the app version, shown in the UI.
// Call Run on the result to start the app.
func New(assets fs.FS, v version.Version) (*application.App, error) {
	cfg, err := config.NewStore()
	if err != nil {
		return nil, err
	}

	a := application.New(application.Options{
		Name:        "Whip",
		Description: "An agentic IDE",
		// Services are Go structs whose exported methods the frontend can call.
		Services: []application.Service{
			application.NewService(user.New()),
			application.NewService(version.NewAbout(v)),
			application.NewService(stocks.New(cfg)),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	window := newMainWindow(a)

	// Show the Whip icon in the menu bar while the app is running.
	tray.New(a, window)

	return a, nil
}

// newMainWindow creates the main IDE window.
func newMainWindow(a *application.App) *application.WebviewWindow {
	return a.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:  "Whip",
		Width:  1450,
		Height: 925,
		Mac: application.MacWindow{
			// Hidden title bar: the page draws under it; the top 50px still
			// drags the window (matches --titlebar-height in frontend/src/styles/base.css).
			InvisibleTitleBarHeight: 50,
			Backdrop:                application.MacBackdropNormal,
			TitleBar:                application.MacTitleBarHiddenInset,
		},
		BackgroundColour: application.NewRGB(250, 249, 246), // off-white, matches --paper-50 in frontend/src/styles/base.css
		URL:              "/",
	})
}
