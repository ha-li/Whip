package app

import "github.com/ha-li/Whip/internal/config"

// UI remembers which tab the frontend has open. It is bound to the frontend.
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
