package app

// Tab is one entry in the left-hand menu.
type Tab struct {
	// ID identifies the tab; the frontend maps it to the page component to show.
	ID string `json:"id"`
	// Label is the text shown in the menu.
	Label string `json:"label"`
	// Icon names the menu icon; the frontend maps it to an SVG.
	Icon string `json:"icon"`
}

// tabs is the left-hand menu, in display order. The first tab is selected on
// launch. To add a tab: append it here, then register its page (by ID) in
// frontend/src/pages/index.ts and its icon (by name) in frontend/src/icons.tsx.
var tabs = []Tab{
	{ID: "home", Label: "Home", Icon: "home"},
}

// Menu is bound to the frontend and serves the left-hand menu.
type Menu struct{}

// NewMenu returns a Menu.
func NewMenu() *Menu {
	return &Menu{}
}

// Tabs returns the tabs to show in the left-hand menu, in display order.
func (m *Menu) Tabs() []Tab {
	out := make([]Tab, len(tabs))
	copy(out, tabs)
	return out
}
