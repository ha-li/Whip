package app

import "testing"

func TestLastTab(t *testing.T) {
	t.Parallel()

	f, dir := openSettings(t)
	ui := &UI{settings: f}
	if got, err := ui.LastTab(); err != nil || got != "" {
		t.Fatalf("LastTab() before saving = %q, %v; want \"\", nil", got, err)
	}
	if err := ui.SetLastTab("portfolio"); err != nil {
		t.Fatalf("SetLastTab() error = %v", err)
	}
	if got, err := ui.LastTab(); err != nil || got != "portfolio" {
		t.Errorf("LastTab() = %q, %v; want notes, nil", got, err)
	}
	if got, want := readSettings(t, dir), "tab: portfolio\n"; got != want {
		t.Errorf("whip.yml = %q, want %q", got, want)
	}
}

func TestSidebarWidth(t *testing.T) {
	t.Parallel()

	f, dir := openSettings(t)
	ui := &UI{settings: f}
	if got, err := ui.SidebarWidth(); err != nil || got != 0 {
		t.Fatalf("SidebarWidth() before saving = %d, %v; want 0, nil", got, err)
	}
	if err := ui.SetLastTab("notes"); err != nil {
		t.Fatal(err)
	}
	if err := ui.SetSidebarWidth(260); err != nil {
		t.Fatalf("SetSidebarWidth() error = %v", err)
	}
	if got, err := ui.SidebarWidth(); err != nil || got != 260 {
		t.Errorf("SidebarWidth() = %d, %v; want 260, nil", got, err)
	}
	// Saving the width keeps the tab.
	if got, want := readSettings(t, dir), "tab: notes\nsidebar_width: 260\n"; got != want {
		t.Errorf("whip.yml = %q, want %q", got, want)
	}

	for _, tt := range []struct {
		name  string
		width int
	}{
		{"zero", 0},
		{"negative", -10},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if err := ui.SetSidebarWidth(tt.width); err == nil {
				t.Errorf("SetSidebarWidth(%d) succeeded, want error", tt.width)
			}
		})
	}
}
