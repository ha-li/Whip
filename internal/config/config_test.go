package config_test

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/ha-li/Whip/internal/config"
)

func TestNewStoreWithRequiresConfigDir(t *testing.T) {
	t.Parallel()

	if _, err := config.NewStoreWith(); err == nil {
		t.Error("NewStoreWith() succeeded without a config directory, want error")
	}
}

func TestWithConfigDir(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	s := newTestStore(t, dir)
	if err := openTestConfig(t, s).Update(func(c *testConfig) { c.Service.APIKey = "K" }); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "config.yml")); err != nil {
		t.Errorf("config.yml not directly in the injected config dir: %v", err)
	}
}

// testConfig stands in for an owner's settings type, kept in config.yml.
type testConfig struct {
	Service testService `yaml:"service"`
}

type testService struct {
	APIKey string `yaml:"api_key"`
}

// openTestConfig opens config.yml in s as a testConfig.
func openTestConfig(t *testing.T, s *config.Store) *config.File[testConfig] {
	t.Helper()
	f, err := config.Open[testConfig](s, "config.yml")
	if err != nil {
		t.Fatal(err)
	}
	return f
}

// newTestStore returns a Store whose files live in dir.
func newTestStore(t *testing.T, dir string) *config.Store {
	t.Helper()
	s, err := config.NewStoreWith(config.WithConfigDir(dir))
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestLoadMissingFile(t *testing.T) {
	t.Parallel()

	c, err := openTestConfig(t, newTestStore(t, t.TempDir())).Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if c != (testConfig{}) {
		t.Errorf("Load() = %+v, want zero testConfig", c)
	}
}

func TestUpdateWritesYAML(t *testing.T) {
	t.Parallel()

	dir := filepath.Join(t.TempDir(), "Whip") // doesn't exist yet
	s := newTestStore(t, dir)
	if err := openTestConfig(t, s).Update(func(c *testConfig) { c.Service.APIKey = "KEY123" }); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	path := filepath.Join(dir, "config.yml")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want := "service:\n    api_key: KEY123\n"
	if string(b) != want {
		t.Errorf("config.yml =\n%s\nwant\n%s", b, want)
	}
	if !strings.HasPrefix(string(b), "service:") {
		t.Errorf("service is not the first key:\n%s", b)
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

	c, err := openTestConfig(t, s).Load()
	if err != nil {
		t.Fatal(err)
	}
	if c.Service.APIKey != "KEY123" {
		t.Errorf("Load() api_key = %q, want KEY123", c.Service.APIKey)
	}
}

func TestLoadRejectsMalformedFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "config.yml"), []byte("service: [unclosed\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := openTestConfig(t, newTestStore(t, dir)).Load(); err == nil {
		t.Error("Load() of malformed YAML succeeded, want error")
	}
}

func TestUpdateIsSerialized(t *testing.T) {
	t.Parallel()

	s := newTestStore(t, t.TempDir())
	var wg sync.WaitGroup
	for range 20 {
		wg.Go(func() {
			if err := openTestConfig(t, s).Update(func(c *testConfig) { c.Service.APIKey += "x" }); err != nil {
				t.Error(err)
			}
		})
	}
	wg.Wait()

	c, err := openTestConfig(t, s).Load()
	if err != nil {
		t.Fatal(err)
	}
	if got := len(c.Service.APIKey); got != 20 {
		t.Errorf("after 20 concurrent appends, key length = %d, want 20 (an update was lost)", got)
	}
}

// editorSettings stands in for another module's settings type.
type editorSettings struct {
	Theme string `yaml:"theme"`
}

func TestOpenSeparateFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	s := newTestStore(t, dir)
	f, err := config.Open[editorSettings](s, "editor.yml")
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	if err := f.Update(func(e *editorSettings) { e.Theme = "dark" }); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	b, err := os.ReadFile(filepath.Join(dir, "editor.yml"))
	if err != nil {
		t.Fatal(err)
	}
	if want := "theme: dark\n"; string(b) != want {
		t.Errorf("editor.yml = %q, want %q", b, want)
	}
	if _, err := os.Stat(filepath.Join(dir, "config.yml")); !os.IsNotExist(err) {
		t.Errorf("config.yml exists after writing editor.yml only (stat error = %v)", err)
	}

	e, err := f.Load()
	if err != nil {
		t.Fatal(err)
	}
	if e.Theme != "dark" {
		t.Errorf("Load() theme = %q, want dark", e.Theme)
	}
}

func TestOpenReturnsSameFile(t *testing.T) {
	t.Parallel()

	s := newTestStore(t, t.TempDir())
	a, err := config.Open[editorSettings](s, "editor.yml")
	if err != nil {
		t.Fatal(err)
	}
	b, err := config.Open[editorSettings](s, "editor.yml")
	if err != nil {
		t.Fatal(err)
	}
	if a != b {
		t.Error("Open() twice returned different Files, want the same one")
	}
}

func TestOpenRejects(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		file string
	}{
		{"empty", ""},
		{"dot", "."},
		{"dot dot", ".."},
		{"path", "sub/editor.yml"},
		{"escape", "../editor.yml"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := newTestStore(t, t.TempDir())
			if _, err := config.Open[editorSettings](s, tt.file); err == nil {
				t.Errorf("Open(%q) succeeded, want error", tt.file)
			}
		})
	}
}

func TestOpenRejectsOtherType(t *testing.T) {
	t.Parallel()

	s := newTestStore(t, t.TempDir())
	openTestConfig(t, s)
	if _, err := config.Open[editorSettings](s, "config.yml"); err == nil {
		t.Error("Open() of config.yml as a second type succeeded, want error")
	}
}

func TestNewStoreRejectsAppName(t *testing.T) {
	t.Parallel()

	for _, app := range []string{"", ".", "..", "a/b", "../Whip"} {
		t.Run(app, func(t *testing.T) {
			t.Parallel()
			if _, err := config.NewStore(app); err == nil {
				t.Errorf("NewStore(%q) succeeded, want error", app)
			}
		})
	}
}

func TestReserve(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	s := newTestStore(t, dir)
	path, err := s.Reserve("data.db")
	if err != nil {
		t.Fatalf("Reserve() error = %v", err)
	}
	if want := filepath.Join(dir, "data.db"); path != want {
		t.Errorf("Reserve() = %q, want %q", path, want)
	}

	// A claimed name can't be claimed again, by either Reserve or Open.
	if _, err := s.Reserve("data.db"); err == nil {
		t.Error("Reserve() of a reserved name succeeded, want error")
	}
	if _, err := config.Open[editorSettings](s, "data.db"); err == nil {
		t.Error("Open() of a reserved name succeeded, want error")
	}
	openTestConfig(t, s)
	if _, err := s.Reserve("config.yml"); err == nil {
		t.Error("Reserve() of an open settings file succeeded, want error")
	}
	if _, err := s.Reserve("../data.db"); err == nil {
		t.Error("Reserve() of a path succeeded, want error")
	}
}
