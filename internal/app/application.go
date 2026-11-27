// Package app assembles the Whip application: the Wails app, its services,
// the main window and the menu bar icon.
package app

import (
	"io/fs"
	"log"

	"github.com/ha-li/Whip/internal/app/tray"
	"github.com/ha-li/Whip/internal/app/version"
	"github.com/ha-li/Whip/internal/app/whipapp"
	"github.com/ha-li/Whip/internal/palette"
	"github.com/ha-li/Whip/internal/user"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// appName is the app's name: the Wails app name, the window title, the menu
// bar icon's tooltip and menu, and the name of its directory in the user's
// config directory (~/Library/Application Support/Whip). It must match
// productName in build/config.yml.
const appName = "Whip"

// New creates the Wails application, its main window and the menu bar icon.
// assets holds the built frontend (frontend/dist), embedded by package main
// because go:embed can't reach files outside its own package directory.
// v is the app version, shown in the UI.
// Call Run on the result to start the app; the database closes when it quits.
func New(assets fs.FS, v version.Version) (*application.App, error) {
	w, err := whipapp.NewWhipApp(appName)
	if err != nil {
		return nil, err
	}
	saved, err := w.AppSetting().Load()
	if err != nil {
		// A damaged whip.yml shouldn't stop Whip: start with the defaults.
		log.Printf("load settings: %v", err)
	}
	windows := &windowTracker{settings: w.AppSetting()}

	// Created right after the app below; the callbacks that use it only run
	// once the app is running.
	var window *application.WebviewWindow
	show := func() { showWindow(window) }

	a := application.New(application.Options{
		Name:        appName,
		Description: "An agentic IDE",
		// Only one Whip runs at a time: launching it again (e.g. `open -n`, or
		// running the binary directly) brings the running app to the front and
		// the new copy exits.
		SingleInstance: &application.SingleInstanceOptions{
			UniqueID: "com.gecko.whip", // productIdentifier in build/config.yml
			OnSecondInstanceLaunch: func(application.SecondInstanceData) {
				show()
			},
		},
		// Services are Go structs whose exported methods the frontend can call.
		Services: []application.Service{
			application.NewService(user.New()),
			application.NewService(version.NewAbout(v)),
			application.NewService(w.StocksService()),
			application.NewService(w.NotebookService()),
			application.NewService(palette.New()),
			application.NewService(w.TaskService()),
			application.NewService(w.Quotes()),
			application.NewService(&UI{settings: w.AppSetting()}),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
		OnShutdown: func() {
			if err := windows.save(); err != nil {
				log.Printf("save window position: %v", err)
			}
			if err := w.Close(); err != nil {
				log.Printf("close database: %v", err)
			}
		},
	})

	window = newMainWindow(a, saved.Window)
	windows.track(window)

	// Show the Whip icon in the menu bar while the app is running.
	tray.New(a, appName, show)

	return a, nil
}

// showWindow brings window to the front: restored if minimised, shown if
// hidden, and focused.
func showWindow(window *application.WebviewWindow) {
	if window == nil {
		return
	}
	if window.IsMinimised() {
		window.UnMinimise()
	}
	window.Show()
	window.Focus()
}

// newMainWindow creates the main IDE window, at bounds if they're usable and
// centred at the default size if not.
func newMainWindow(a *application.App, bounds *whipapp.WindowBounds) *application.WebviewWindow {
	opts := application.WebviewWindowOptions{
		Title:  appName,
		Width:  whipapp.DefaultWidth,
		Height: whipapp.DefaultHeight,
		Mac: application.MacWindow{
			// Hidden title bar: the page draws under it; the top 50px still
			// drags the window (matches --titlebar-height in frontend/src/styles/base.css).
			InvisibleTitleBarHeight: 50,
			Backdrop:                application.MacBackdropNormal,
			TitleBar:                application.MacTitleBarHiddenInset,
		},
		BackgroundColour: application.NewRGB(250, 249, 246), // off-white, matches --paper-50 in frontend/src/styles/base.css
		URL:              "/",
	}
	if bounds.Usable() {
		opts.Width, opts.Height = bounds.Width, bounds.Height
		opts.InitialPosition = application.WindowXY
		opts.X, opts.Y = bounds.X, bounds.Y
	}
	window := a.Window.NewWithOptions(opts)
	if bounds.Usable() {
		keepOnScreen(window)
	}
	return window
}
