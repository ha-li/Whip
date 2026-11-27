package app

import (
	"fmt"

	"github.com/ha-li/Whip/internal/config"
)

// UI remembers which tab the frontend has open and how wide its side menu
// is. It is bound to the frontend.
type UI struct {
	settings *config.File[settings]
}

// LastTab returns the id of the tab that was open when Whip last quit, or ""
// if none was saved. The frontend falls back to its first tab for an id it
// doesn't know.
func (u *UI) LastTab() (string, error) {
	s, err := u.settings.Load()
	return s.Tab, err
}

// SetLastTab saves id as the open tab.
func (u *UI) SetLastTab(id string) error {
	return u.settings.Update(func(s *settings) { s.Tab = id })
}

// SidebarWidth returns the side menu's width in CSS pixels as last saved, or
// 0 if none was saved. The frontend keeps the width within its own limits.
func (u *UI) SidebarWidth() (int, error) {
	s, err := u.settings.Load()
	return s.SidebarWidth, err
}

// SetSidebarWidth saves width, in CSS pixels, as the side menu's width.
func (u *UI) SetSidebarWidth(width int) error {
	if width <= 0 {
		return fmt.Errorf("sidebar width %d is not positive", width)
	}
	return u.settings.Update(func(s *settings) { s.SidebarWidth = width })
}
