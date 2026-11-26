package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// Option sets where a Store keeps its files. NewStoreWith needs both a config
// directory and a cache directory: NewStore passes WithDefaultConfigDir and
// WithDefaultCacheDir; tests pass WithConfigDir and WithCacheDir with
// temporary directories.
type Option func(*dirs) error

type dirs struct {
	config, cache string
}

// WithConfigDir puts config.yml in dir. dir is used as is.
func WithConfigDir(dir string) Option {
	return func(d *dirs) error {
		d.config = dir
		return nil
	}
}

// WithCacheDir uses dir as the cache directory. dir is used as is.
func WithCacheDir(dir string) Option {
	return func(d *dirs) error {
		d.cache = dir
		return nil
	}
}

// WithDefaultConfigDir puts config.yml in the user's config directory,
// ~/Library/Application Support/Whip (from os.UserConfigDir).
func WithDefaultConfigDir() Option {
	return func(d *dirs) error {
		dir, err := os.UserConfigDir()
		if err != nil {
			return fmt.Errorf("find config directory: %w", err)
		}
		return WithConfigDir(filepath.Join(dir, appDir))(d)
	}
}

// WithDefaultCacheDir uses the user's cache directory, ~/Library/Caches/Whip
// (from os.UserCacheDir).
func WithDefaultCacheDir() Option {
	return func(d *dirs) error {
		dir, err := os.UserCacheDir()
		if err != nil {
			return fmt.Errorf("find cache directory: %w", err)
		}
		return WithCacheDir(filepath.Join(dir, appDir))(d)
	}
}
