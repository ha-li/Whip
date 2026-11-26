package config_test

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/ha-li/Whip/internal/config"
)

func TestNewStoreWithRequiresBothDirs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		opts []config.Option
	}{
		{"neither", nil},
		{"config only", []config.Option{config.WithConfigDir(t.TempDir())}},
		{"cache only", []config.Option{config.WithCacheDir(t.TempDir())}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if _, err := config.NewStoreWith(tt.opts...); err == nil {
				t.Error("NewStoreWith() succeeded without both directories, want error")
			}
		})
	}
}

func TestOptionsOverrideDirs(t *testing.T) {
	t.Parallel()

	configDir, cacheDir := t.TempDir(), t.TempDir()
	s := newTestStore(t, configDir, cacheDir)
	if s.CacheDir() != cacheDir {
		t.Errorf("CacheDir() = %q, want %q (used as is)", s.CacheDir(), cacheDir)
	}
	if err := s.Update(func(c *config.Config) { c.AlphaVantage.APIKey = "K" }); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(configDir, "config.yml")); err != nil {
		t.Errorf("config.yml not directly in the injected config dir: %v", err)
	}
}

// newTestStore returns a Store whose files live in the given directories.
func newTestStore(t *testing.T, configDir, cacheDir string) *config.Store {
	t.Helper()
	s, err := config.NewStoreWith(config.WithConfigDir(configDir), config.WithCacheDir(cacheDir))
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestLoadMissingFile(t *testing.T) {
	t.Parallel()

	c, err := newTestStore(t, t.TempDir(), t.TempDir()).Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if c != (config.Config{}) {
		t.Errorf("Load() = %+v, want zero Config", c)
	}
}

func TestUpdateWritesYAML(t *testing.T) {
	t.Parallel()

	dir := filepath.Join(t.TempDir(), "Whip") // doesn't exist yet
	s := newTestStore(t, dir, t.TempDir())
	if err := s.Update(func(c *config.Config) { c.AlphaVantage.APIKey = "KEY123" }); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	path := filepath.Join(dir, "config.yml")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want := "alphavantage:\n    api_key: KEY123\n"
	if string(b) != want {
		t.Errorf("config.yml =\n%s\nwant\n%s", b, want)
	}
	if !strings.HasPrefix(string(b), "alphavantage:") {
		t.Errorf("alphavantage is not the first key:\n%s", b)
	}

	// It holds secrets: readable only by the user.
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("config.yml mode = %o, want 600", perm)
	}

	// No temporary files are left behind.
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Errorf("files in %s = %d, want just config.yml", dir, len(entries))
	}

	c, err := s.Load()
	if err != nil {
		t.Fatal(err)
	}
	if c.AlphaVantage.APIKey != "KEY123" {
		t.Errorf("Load() api_key = %q, want KEY123", c.AlphaVantage.APIKey)
	}
}

func TestLoadRejectsMalformedFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "config.yml"), []byte("alphavantage: [unclosed\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := newTestStore(t, dir, t.TempDir()).Load(); err == nil {
		t.Error("Load() of malformed YAML succeeded, want error")
	}
}

func TestUpdateIsSerialized(t *testing.T) {
	t.Parallel()

	s := newTestStore(t, t.TempDir(), t.TempDir())
	var wg sync.WaitGroup
	for range 20 {
		wg.Go(func() {
			if err := s.Update(func(c *config.Config) { c.AlphaVantage.APIKey += "x" }); err != nil {
				t.Error(err)
			}
		})
	}
	wg.Wait()

	c, err := s.Load()
	if err != nil {
		t.Fatal(err)
	}
	if got := len(c.AlphaVantage.APIKey); got != 20 {
		t.Errorf("after 20 concurrent appends, key length = %d, want 20 (an update was lost)", got)
	}
}
