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
