// Package config owns Whip's per-user directories and its settings file,
// ~/Library/Application Support/Whip/config.yml.
//
// The file can hold secrets (API keys), so it is written readable only by the
// current user. Whip rewrites the whole file on every change; comments and keys
// it doesn't know about are not kept.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"go.yaml.in/yaml/v3"
)

// Config is the contents of config.yml.
type Config struct {
	AlphaVantage AlphaVantage `yaml:"alphavantage"`
}

// AlphaVantage holds the settings for the Alpha Vantage market data API.
type AlphaVantage struct {
	APIKey string `yaml:"api_key"`
}

// appDir names Whip's directory inside the user's config and cache directories.
const appDir = "Whip"

// Store reads and writes config.yml and knows Whip's cache directory.
// It is safe for concurrent use.
type Store struct {
	path     string // ~/Library/Application Support/Whip/config.yml
	cacheDir string // ~/Library/Caches/Whip
	mu       sync.Mutex
}

// NewStore returns the Store for the current user: config.yml in
// ~/Library/Application Support/Whip and the cache in ~/Library/Caches/Whip.
func NewStore() (*Store, error) {
	return NewStoreWith(WithDefaultConfigDir(), WithDefaultCacheDir())
}

// NewStoreWith returns a Store whose directories are set by opts (see Option),
// e.g. temporary directories in tests. Both a config directory and a cache
// directory are required. config.yml is created on first write.
func NewStoreWith(opts ...Option) (*Store, error) {
	var d dirs
	for _, opt := range opts {
		if err := opt(&d); err != nil {
			return nil, err
		}
	}
	if d.config == "" {
		return nil, errors.New("config directory not set (use NewStore, or NewStoreWith and WithConfigDir)")
	}
	if d.cache == "" {
		return nil, errors.New("cache directory not set (use NewStore, or NewStoreWith and WithCacheDir)")
	}
	return &Store{path: filepath.Join(d.config, "config.yml"), cacheDir: d.cache}, nil
}

// CacheDir returns Whip's cache directory, ~/Library/Caches/Whip. Its contents
// can be deleted at any time; callers create it when they first write.
func (s *Store) CacheDir() string {
	return s.cacheDir
}

// Load returns the saved settings, or the zero Config if the file doesn't exist.
func (s *Store) Load() (Config, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.load()
}

// Update applies change to the saved settings and writes them back.
func (s *Store) Update(change func(*Config)) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	c, err := s.load()
	if err != nil {
		return err
	}
	change(&c)
	return s.save(c)
}

func (s *Store) load() (Config, error) {
	var c Config
	b, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return c, nil
	}
	if err != nil {
		return c, fmt.Errorf("read %s: %w", s.path, err)
	}
	if err := yaml.Unmarshal(b, &c); err != nil {
		return c, fmt.Errorf("parse %s: %w", s.path, err)
	}
	return c, nil
}

// save writes c to a temporary file and renames it into place, so a crash
// never leaves a half-written config.
func (s *Store) save(c Config) error {
	b, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("encode config: %w", err)
	}
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create %s: %w", dir, err)
	}
	tmp, err := os.CreateTemp(dir, ".config-*.yml") // created with mode 0600
	if err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	defer func() { _ = os.Remove(tmp.Name()) }() // no-op after a successful rename
	if _, err := tmp.Write(b); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write config: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	if err := os.Rename(tmp.Name(), s.path); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}
