package whipapp

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ha-li/Whip/internal/config"
)

func TestNewWhipApp(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	store, err := config.NewStoreWith(config.WithConfigDir(dir))
	if err != nil {
		t.Fatal(err)
	}
	w, err := newWhipApp(store)
	if err != nil {
		t.Fatalf("newWhipApp() error = %v", err)
	}
	t.Cleanup(func() {
		if err := w.Close(); err != nil {
			t.Errorf("Close() error = %v", err)
		}
	})

	parts := []struct {
		name string
		nil  bool
	}{
		{"Store", w.Store() == nil},
		{"DB", w.DB() == nil},
		{"StocksService", w.StocksService() == nil},
		{"NotebookService", w.NotebookService() == nil},
		{"TaskService", w.TaskService() == nil},
		{"Quotes", w.Quotes() == nil},
		{"AppSetting", w.AppSetting() == nil},
	}
	for _, p := range parts {
		t.Run(p.name, func(t *testing.T) {
			t.Parallel()
			if p.nil {
				t.Errorf("%s() = nil, want it created", p.name)
			}
		})
	}
	if _, err := os.Stat(filepath.Join(dir, databaseFile)); err != nil {
		t.Errorf("database file: %v, want it created in the config directory", err)
	}

	// The services work on the database.
	if _, err := w.NotebookService().List(t.Context()); err != nil {
		t.Errorf("NotebookService().List() error = %v", err)
	}
	if err := w.AppSetting().Update(func(s *Settings) { s.Tab = "notes" }); err != nil {
		t.Errorf("AppSetting().Update() error = %v", err)
	}
}
