package app_test

import (
	"testing"

	"github.com/ha-li/Whip/internal/app"
)

func TestTabs(t *testing.T) {
	t.Parallel()

	got := app.NewMenu().Tabs()
	if len(got) == 0 {
		t.Fatal("Tabs() returned no tabs")
	}
	if got[0].ID != "home" {
		t.Errorf("first tab = %q, want %q", got[0].ID, "home")
	}

	seen := make(map[string]bool)
	for _, tab := range got {
		t.Run(tab.ID, func(t *testing.T) {
			t.Parallel()
			if tab.ID == "" || tab.Label == "" || tab.Icon == "" {
				t.Errorf("tab %+v has an empty field", tab)
			}
		})
		if seen[tab.ID] {
			t.Errorf("duplicate tab ID %q", tab.ID)
		}
		seen[tab.ID] = true
	}
}

func TestTabsReturnsCopy(t *testing.T) {
	t.Parallel()

	got := app.NewMenu().Tabs()
	got[0].Label = "changed"
	if app.NewMenu().Tabs()[0].Label == "changed" {
		t.Error("Tabs() exposes the package-level slice; callers can modify the menu")
	}
}
