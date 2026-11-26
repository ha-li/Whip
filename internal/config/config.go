// Package config owns an app's per-user config directory and the settings
// files in it. Package app names the directory and creates the current user's Store
// with NewStore("Whip"), then passes it to the modules that need settings.
//
// Each module owns its own settings file: it defines the file's contents and
// opens the file from the Store with Open, getting back a *File to read and
// update. Neither this package nor this package's other users know what a
// module's file holds. Files that aren't settings files, such as the
// database, are claimed with Store.Reserve.
//
// # Adding a settings file
//
// To give a module, say editor, its own file editor.yml, first define the
// file's contents in the module as a struct with yaml tags. Its zero value is
// what Load returns before the file exists, so make that a sensible default:
//
//	package editor
//
//	type Settings struct {
//		Theme    string `yaml:"theme"`
//		FontSize int    `yaml:"font_size"`
//	}
//
// Then have the module take the *config.Store and open the file from it.
// Opening the same name again returns the same File, so Updates from
// different places stay serialized; opening it with a different type is an
// error:
//
//	func New(store *config.Store) (*Editor, error) {
//		f, err := config.Open[Settings](store, "editor.yml")
//		...
//	}
//
// Read it with Load and change it with Update, which rewrites the whole file.
// The file is created on the first Update, readable only by the user, so it
// may hold secrets:
//
//	s, err := f.Load()
//	err = f.Update(func(s *editor.Settings) { s.Theme = "dark" })
//
// Tests shouldn't touch the user's real files: build the Store on t.TempDir()
// with NewStoreWith and WithConfigDir, and pass it to the module.
package config

import (
	"errors"
	"fmt"
	"path/filepath"
	"sync"
)

// Store knows an app's config directory and hands out the files in it. Create
// one per directory (NewStore for the current user, NewStoreWith in tests) and
// share it: a second Store on the same directory would hand out
// separate Files whose Updates aren't serialized with the first's. It is safe
// for concurrent use.
type Store struct {
	configDir string // e.g. ~/Library/Application Support/Whip

	mu    sync.Mutex
	files map[string]any // file name -> *File[T] (see Open) or reserved (see Reserve)
}

// NewStore returns the Store for the current user, keeping app's files in a
// directory named app inside the user's config directory: for "Whip",
// ~/Library/Application Support/Whip.
func NewStore(app string) (*Store, error) {
	return NewStoreWith(WithDefaultConfigDir(app))
}

// NewStoreWith returns a Store whose directory is set by opts (see Option),
// e.g. a temporary directory in tests. A config directory is required.
// Settings files are created on first write.
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
	return &Store{configDir: d.config, files: map[string]any{}}, nil
}

// Open returns the settings file name (e.g. "editor.yml") in s's config
// directory, decoded as T. Opening the same name again returns the same File, so Updates to it stay
// serialized; opening it with a different T is an error. name must be a plain
// file name, not a path.
func Open[T any](s *Store, name string) (*File[T], error) {
	if !isPlainName(name) {
		return nil, fmt.Errorf("settings file name %q is not a plain file name", name)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if f, ok := s.files[name]; ok {
		tf, ok := f.(*File[T])
		if !ok {
			return nil, fmt.Errorf("settings file %s is already open as %T", name, f)
		}
		return tf, nil
	}
	f := &File[T]{path: filepath.Join(s.configDir, name)}
	s.files[name] = f
	return f, nil
}

// reserved marks a file name in the config directory claimed with Reserve.
type reserved struct{}

// Reserve claims name in s's config directory for a file that isn't a
// settings file, such as a database, and returns its path. The caller owns
// the file and creates it. A name can be claimed once, by either Reserve or
// Open: claiming it again is an error, so two owners can't share a file.
// name must be a plain file name, not a path.
func (s *Store) Reserve(name string) (string, error) {
	if !isPlainName(name) {
		return "", fmt.Errorf("file name %q is not a plain file name", name)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.files[name]; ok {
		return "", fmt.Errorf("file %s in the config directory is already claimed", name)
	}
	s.files[name] = reserved{}
	return filepath.Join(s.configDir, name), nil
}

// isPlainName reports whether name is a single path element: not empty, not
// "." or "..", and without separators.
func isPlainName(name string) bool {
	return name != "" && name != "." && name != ".." && filepath.Base(name) == name
}
