package db_test

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/ha-li/Whip/internal/db"
)

// item is a test document.
type item struct {
	ID    string `json:"id"`
	Color string `json:"color"`
	Size  int    `json:"size"`
}

func (i item) DocumentID() string { return i.ID }

// openDB opens a database in a new temporary directory.
func openDB(t *testing.T) *db.DB {
	t.Helper()
	path := filepath.Join(t.TempDir(), "data.db")
	d, err := db.Open(path)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() { _ = d.Close() })
	return d
}

func openItems(t *testing.T, d *db.DB, name string) *db.Collection[item] {
	t.Helper()
	c, err := db.OpenCollection[item](t.Context(), d, name)
	if err != nil {
		t.Fatalf("OpenCollection() error = %v", err)
	}
	return c
}

func TestPutGetDelete(t *testing.T) {
	t.Parallel()

	d := openDB(t)
	c := openItems(t, d, "items")
	ctx := t.Context()

	if _, err := c.Get(ctx, "a"); !errors.Is(err, db.ErrNotFound) {
		t.Fatalf("Get() of a missing ID error = %v, want ErrNotFound", err)
	}
	want := item{ID: "a", Color: "red", Size: 3}
	if err := c.Put(ctx, want); err != nil {
		t.Fatalf("Put() error = %v", err)
	}
	got, err := c.Get(ctx, "a")
	if err != nil || got != want {
		t.Fatalf("Get() = %+v, %v; want %+v, nil", got, err, want)
	}

	// Put with the same ID replaces the document.
	want.Color = "blue"
	if err := c.Put(ctx, want); err != nil {
		t.Fatal(err)
	}
	if got, _ := c.Get(ctx, "a"); got != want {
		t.Errorf("Get() after replacing = %+v, want %+v", got, want)
	}

	if err := c.Delete(ctx, "a"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if _, err := c.Get(ctx, "a"); !errors.Is(err, db.ErrNotFound) {
		t.Errorf("Get() after Delete error = %v, want ErrNotFound", err)
	}
	if err := c.Delete(ctx, "a"); err != nil {
		t.Errorf("Delete() of a missing ID error = %v, want nil", err)
	}
	if err := c.Put(ctx, item{}); err == nil {
		t.Error("Put() with an empty ID succeeded, want error")
	}
}

func TestAll(t *testing.T) {
	t.Parallel()

	d := openDB(t)
	c := openItems(t, d, "items")
	others := openItems(t, d, "others")
	ctx := t.Context()

	if docs, err := c.All(ctx); err != nil || len(docs) != 0 {
		t.Fatalf("All() of an empty collection = %v, %v; want none", docs, err)
	}
	for _, it := range []item{
		{ID: "a", Color: "red", Size: 1},
		{ID: "b", Color: "blue", Size: 2},
	} {
		if err := c.Put(ctx, it); err != nil {
			t.Fatal(err)
		}
	}
	if err := others.Put(ctx, item{ID: "z"}); err != nil {
		t.Fatal(err)
	}

	docs, err := c.All(ctx)
	if err != nil {
		t.Fatalf("All() error = %v", err)
	}
	var ids []string
	for _, doc := range docs {
		ids = append(ids, doc.ID)
	}
	slices.Sort(ids)
	if want := []string{"a", "b"}; !slices.Equal(ids, want) {
		t.Errorf("All() IDs = %v, want %v (only this collection's)", ids, want)
	}
}

func TestOpenCollectionRejects(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		coll string
	}{
		{"empty name", ""},
		{"quote in name", `items"; DROP TABLE x; --`},
		{"space in name", "my items"},
		{"leading digit", "1items"},
		{"sqlite prefix", "sqlite_items"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			d := openDB(t)
			if _, err := db.OpenCollection[item](t.Context(), d, tt.coll); err == nil {
				t.Errorf("OpenCollection(%q) succeeded, want error", tt.coll)
			}
		})
	}
}

func TestOpenCollectionOnce(t *testing.T) {
	t.Parallel()

	d := openDB(t)
	openItems(t, d, "items")
	if _, err := db.OpenCollection[item](t.Context(), d, "items"); err == nil {
		t.Error("opening a collection twice succeeded, want error")
	}
	openItems(t, d, "others") // a different name is fine
}

func TestDataPersists(t *testing.T) {
	t.Parallel()

	// A space in the path, like ~/Library/Application Support.
	path := filepath.Join(t.TempDir(), "Application Support", "data.db")
	d, err := db.Open(path)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	if err := openItems(t, d, "items").Put(t.Context(), item{ID: "a", Color: "red"}); err != nil {
		t.Fatal(err)
	}
	if err := d.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("database file mode = %o, want 600", perm)
	}

	d, err = db.Open(path)
	if err != nil {
		t.Fatalf("reopen error = %v", err)
	}
	t.Cleanup(func() { _ = d.Close() })
	got, err := openItems(t, d, "items").Get(t.Context(), "a")
	if err != nil || got.Color != "red" {
		t.Errorf("Get() after reopening = %+v, %v; want color red", got, err)
	}
}

func TestOpenRejectsQuestionMark(t *testing.T) {
	t.Parallel()

	if _, err := db.Open(filepath.Join(t.TempDir(), "data.db?mode=ro")); err == nil {
		t.Error("Open() of a path with '?' succeeded, want error")
	}
}
