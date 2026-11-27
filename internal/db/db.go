// Package db is the app's document database: one SQLite file holding JSON
// documents in named collections.
//
// Each module owns its collections, the way it owns its settings file (see
// package config): it opens them with OpenCollection, defines their document
// type, and no other module reads or writes them. A collection name can be
// opened once per DB, so two modules can't share one by accident.
//
// Each collection is a table with one row per document, the document stored
// as JSON text. The file can be inspected with the sqlite3 command-line tool.
package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	_ "modernc.org/sqlite" // registers the "sqlite" database/sql driver
)

// ErrNotFound is returned by Get when no document has the requested ID.
var ErrNotFound = errors.New("document not found")

// namePattern accepts collection names. They become SQL identifiers, so
// nothing that needs quoting is allowed.
var namePattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]{0,62}$`)

// DB is an open database. It is safe for concurrent use.
type DB struct {
	sql *sql.DB

	mu          sync.Mutex
	collections map[string]bool // opened collection names
}

// Open opens the database file at path, creating it (and its directory) if it
// doesn't exist. The file is readable only by the user. Close it when done.
func Open(path string) (*DB, error) {
	if strings.Contains(path, "?") {
		return nil, fmt.Errorf("database path %q contains a '?'", path) // the driver reads it as options
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, fmt.Errorf("create %s: %w", filepath.Dir(path), err)
	}
	// Create the file first so it gets 0600; SQLite gives its -wal and -shm
	// files the same permissions.
	f, err := os.OpenFile(path, os.O_RDONLY|os.O_CREATE, 0o600)
	if err != nil {
		return nil, fmt.Errorf("create %s: %w", path, err)
	}
	if err := f.Close(); err != nil {
		return nil, fmt.Errorf("create %s: %w", path, err)
	}

	// WAL lets reads run alongside a write; busy_timeout makes a write wait
	// for another instead of failing; _txlock=immediate takes the write lock
	// when a transaction starts, so it can't fail halfway on lock upgrade.
	dsn := path + "?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_txlock=immediate"
	conn, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	if err := conn.Ping(); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	return &DB{sql: conn, collections: map[string]bool{}}, nil
}

// Close closes the database. Collections opened from it stop working.
func (d *DB) Close() error {
	return d.sql.Close()
}

// Document is a value stored in a Collection. DocumentID is its ID, unique
// within the collection; the collection keys the document by it.
type Document interface {
	DocumentID() string
}

// Collection is a named set of documents of type T, stored as JSON.
// It is safe for concurrent use.
type Collection[T Document] struct {
	sql  *sql.DB
	name string
}

// OpenCollection opens the collection name in d, creating it if it doesn't
// exist. The name must be letters, digits and underscores, starting with a
// letter. Each name can be opened once per DB.
func OpenCollection[T Document](ctx context.Context, d *DB, name string) (*Collection[T], error) {
	if !namePattern.MatchString(name) || strings.HasPrefix(strings.ToLower(name), "sqlite_") {
		return nil, fmt.Errorf("collection name %q is not allowed", name)
	}

	d.mu.Lock()
	defer d.mu.Unlock()
	if d.collections[name] {
		return nil, fmt.Errorf("collection %s is already open", name)
	}

	stmt := fmt.Sprintf(
		`CREATE TABLE IF NOT EXISTS %q (id TEXT PRIMARY KEY NOT NULL, doc TEXT NOT NULL CHECK (json_valid(doc))) STRICT`,
		name)
	if _, err := d.sql.ExecContext(ctx, stmt); err != nil {
		return nil, fmt.Errorf("create collection %s: %w", name, err)
	}
	d.collections[name] = true
	return &Collection[T]{sql: d.sql, name: name}, nil
}

// Put saves doc under doc.DocumentID(), replacing any document with that ID.
func (c *Collection[T]) Put(ctx context.Context, doc T) error {
	id := doc.DocumentID()
	if id == "" {
		return fmt.Errorf("put in %s: empty document ID", c.name)
	}
	b, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("put %s in %s: %w", id, c.name, err)
	}
	q := fmt.Sprintf(`INSERT INTO %q (id, doc) VALUES (?, ?) ON CONFLICT (id) DO UPDATE SET doc = excluded.doc`, c.name)
	if _, err := c.sql.ExecContext(ctx, q, id, string(b)); err != nil {
		return fmt.Errorf("put %s in %s: %w", id, c.name, err)
	}
	return nil
}

// Get returns the document with the given ID, or an error wrapping
// ErrNotFound if there is none.
func (c *Collection[T]) Get(ctx context.Context, id string) (T, error) {
	var doc T
	var raw string
	q := fmt.Sprintf(`SELECT doc FROM %q WHERE id = ?`, c.name)
	err := c.sql.QueryRowContext(ctx, q, id).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return doc, fmt.Errorf("%s in %s: %w", id, c.name, ErrNotFound)
	}
	if err != nil {
		return doc, fmt.Errorf("get %s from %s: %w", id, c.name, err)
	}
	if err := json.Unmarshal([]byte(raw), &doc); err != nil {
		return doc, fmt.Errorf("decode %s from %s: %w", id, c.name, err)
	}
	return doc, nil
}

// Delete permanently removes the documents with the given IDs, all of them
// or, on error, none. IDs with no document are skipped.
func (c *Collection[T]) Delete(ctx context.Context, ids ...string) (err error) {
	tx, err := c.sql.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("delete from %s: %w", c.name, err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()
	q := fmt.Sprintf(`DELETE FROM %q WHERE id = ?`, c.name)
	for _, id := range ids {
		if _, err := tx.ExecContext(ctx, q, id); err != nil {
			return fmt.Errorf("delete %s from %s: %w", id, c.name, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("delete from %s: %w", c.name, err)
	}
	return nil
}

// All returns every document in the collection, in no particular order.
func (c *Collection[T]) All(ctx context.Context) ([]T, error) {
	rows, err := c.sql.QueryContext(ctx, fmt.Sprintf(`SELECT doc FROM %q`, c.name))
	if err != nil {
		return nil, fmt.Errorf("list %s: %w", c.name, err)
	}
	defer func() { _ = rows.Close() }()

	var docs []T
	for rows.Next() {
		var raw string
		if err := rows.Scan(&raw); err != nil {
			return nil, fmt.Errorf("list %s: %w", c.name, err)
		}
		var doc T
		if err := json.Unmarshal([]byte(raw), &doc); err != nil {
			return nil, fmt.Errorf("list %s: decode: %w", c.name, err)
		}
		docs = append(docs, doc)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list %s: %w", c.name, err)
	}
	return docs, nil
}
