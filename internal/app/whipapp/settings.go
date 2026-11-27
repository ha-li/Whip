package whipapp

// Settings is the contents of whip.yml, the app's own settings file.
type Settings struct {
	// Window is where the main window was when Whip last quit; nil until then.
	Window *WindowBounds `yaml:"window,omitempty"`
	// Tab is the id of the tab that was last open (see frontend/src/tabs.ts).
	Tab string `yaml:"tab,omitempty"`
	// SidebarWidth is the width of the side menu in CSS pixels; 0 until the
	// user resizes it.
	SidebarWidth int `yaml:"sidebar_width,omitempty"`
}
