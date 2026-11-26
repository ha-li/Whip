package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"go.yaml.in/yaml/v3"
)

// File is a YAML settings file whose contents decode into T. It is safe for
// concurrent use. Get one with Open, which hands out a single File per file
// name so that concurrent Updates to the same file are serialized.
//
// Settings files can hold secrets (API keys), so a File is written readable
// only by the current user. The whole file is rewritten on every change;
// comments and keys that T doesn't know about are not kept.
type File[T any] struct {
	path string
	mu   sync.Mutex
}

// Load returns the saved settings, or the zero T if the file doesn't exist.
func (f *File[T]) Load() (T, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.load()
}

// Update applies change to the saved settings and writes them back.
func (f *File[T]) Update(change func(*T)) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	v, err := f.load()
	if err != nil {
		return err
	}
	change(&v)
	return f.save(v)
}

func (f *File[T]) load() (T, error) {
	var v T
	b, err := os.ReadFile(f.path)
	if errors.Is(err, os.ErrNotExist) {
		return v, nil
	}
	if err != nil {
		return v, fmt.Errorf("read %s: %w", f.path, err)
	}
	if err := yaml.Unmarshal(b, &v); err != nil {
		return v, fmt.Errorf("parse %s: %w", f.path, err)
	}
	return v, nil
}

// save writes v to a temporary file and renames it into place, so a crash
// never leaves a half-written file.
func (f *File[T]) save(v T) error {
	b, err := yaml.Marshal(v)
	if err != nil {
		return fmt.Errorf("encode %s: %w", f.path, err)
	}
	dir := filepath.Dir(f.path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create %s: %w", dir, err)
	}
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(f.path)+"-*") // created with mode 0600
	if err != nil {
		return fmt.Errorf("write %s: %w", f.path, err)
	}
	defer func() { _ = os.Remove(tmp.Name()) }() // no-op after a successful rename
	if _, err := tmp.Write(b); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write %s: %w", f.path, err)
	}
	// Flush to disk before the rename: otherwise a crash can leave the renamed
	// file empty. On macOS, Sync uses F_FULLFSYNC.
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write %s: %w", f.path, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("write %s: %w", f.path, err)
	}
	if err := os.Rename(tmp.Name(), f.path); err != nil {
		return fmt.Errorf("write %s: %w", f.path, err)
	}
	return nil
}
