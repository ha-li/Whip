package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// Option sets where a Store keeps its files. NewStoreWith needs a config
// directory: NewStore passes WithDefaultConfigDir; tests pass WithConfigDir
// with a temporary directory.
type Option func(*dirs) error

type dirs struct {
	config string
}

// WithConfigDir keeps the Store's files (settings files opened with Open, and
// files claimed with Reserve) in dir. dir is used as is.
func WithConfigDir(dir string) Option {
	return func(d *dirs) error {
		d.config = dir
		return nil
	}
}

// WithDefaultConfigDir keeps the Store's files in the directory named app
// inside the user's config directory (from os.UserConfigDir), e.g.
// ~/Library/Application Support/Whip.
func WithDefaultConfigDir(app string) Option {
	return func(d *dirs) error {
		if !isPlainName(app) {
			return fmt.Errorf("app directory name %q is not a plain name", app)
		}
		dir, err := os.UserConfigDir()
		if err != nil {
			return fmt.Errorf("find config directory: %w", err)
		}
		return WithConfigDir(filepath.Join(dir, app))(d)
	}
}
