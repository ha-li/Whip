package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ha-li/Whip/internal/app/whipapp"
	"github.com/ha-li/Whip/internal/config"
)

// openSettings opens whip.yml in a temporary directory and returns it with
// the directory.
func openSettings(t *testing.T) (*config.File[whipapp.Settings], string) {
	t.Helper()
	dir := t.TempDir()
	store, err := config.NewStoreWith(config.WithConfigDir(dir))
	if err != nil {
		t.Fatal(err)
	}
	f, err := config.Open[whipapp.Settings](store, whipapp.SettingsFile)
	if err != nil {
		t.Fatal(err)
	}
	return f, dir
}

func readSettings(t *testing.T, dir string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(dir, whipapp.SettingsFile))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestWindowTrackerSave(t *testing.T) {
	t.Parallel()

	t.Run("nothing to save", func(t *testing.T) {
		t.Parallel()
		f, dir := openSettings(t)
		if err := (&windowTracker{settings: f}).save(); err != nil {
			t.Fatalf("save() error = %v", err)
		}
		if _, err := os.Stat(filepath.Join(dir, whipapp.SettingsFile)); !os.IsNotExist(err) {
			t.Errorf("whip.yml exists (stat error = %v), want none when the window never moved", err)
		}
	})

	t.Run("keeps the tab", func(t *testing.T) {
		t.Parallel()
		f, dir := openSettings(t)
		if err := (&UI{settings: f}).SetLastTab("portfolio"); err != nil {
			t.Fatal(err)
		}
		tracker := &windowTracker{settings: f, bounds: &whipapp.WindowBounds{X: 10, Y: 20, Width: 800, Height: 600}}
		if err := tracker.save(); err != nil {
			t.Fatalf("save() error = %v", err)
		}
		// "y" is quoted because YAML 1.1 reads a bare y as true.
		want := "window:\n    x: 10\n    \"y\": 20\n    width: 800\n    height: 600\ntab: portfolio\n"
		if got := readSettings(t, dir); got != want {
			t.Errorf("whip.yml =\n%s\nwant\n%s", got, want)
		}
		s, err := f.Load()
		if err != nil || s.Window == nil || *s.Window != *tracker.bounds {
			t.Errorf("Load() = %+v, %v; want window %+v", s, err, *tracker.bounds)
		}
	})
}
