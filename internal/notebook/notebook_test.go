package notebook_test

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/ha-li/Whip/internal/db"
	"github.com/ha-li/Whip/internal/notebook"
)

// newNotebook returns a Notebook on a database in a temporary directory.
func newNotebook(t *testing.T) *notebook.Notebook {
	t.Helper()
	database, err := db.Open(filepath.Join(t.TempDir(), "data.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	n, err := notebook.New(t.Context(), database)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	return n
}

// texts returns the notes as "Title: text", sorted.
func texts(list []notebook.Note) []string {
	out := make([]string, len(list))
	for i, n := range list {
		out[i] = n.Title + ": " + n.Text
	}
	slices.Sort(out)
	return out
}

func TestNotebook(t *testing.T) {
	t.Parallel()

	n := newNotebook(t)
	ctx := t.Context()

	list, err := n.List(ctx)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if list == nil || len(list) != 0 {
		t.Fatalf("List() with no notes = %#v, want an empty, non-nil list", list)
	}

	first, err := n.Add(ctx, "  Groceries ", "  milk, eggs\n")
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	if first.Title != "Groceries" || first.Text != "milk, eggs" || first.ID == "" {
		t.Errorf("Add() = %+v, want trimmed title and text and an ID", first)
	}
	if !first.Created.Equal(first.Updated) {
		t.Errorf("new note Created %v != Updated %v", first.Created, first.Updated)
	}
	if _, err := n.Add(ctx, "Ideas", "a better clock"); err != nil {
		t.Fatal(err)
	}

	list, err = n.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := texts(list), []string{"Groceries: milk, eggs", "Ideas: a better clock"}; !slices.Equal(got, want) {
		t.Errorf("List() = %q, want %q", got, want)
	}

	updated, err := n.Update(ctx, first.ID, "Shopping", "bread")
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if updated.Title != "Shopping" || updated.Text != "bread" || !updated.Created.Equal(first.Created) || updated.Updated.Before(first.Updated) {
		t.Errorf("Update() = %+v, want new title and text, same Created, later or equal Updated", updated)
	}

	if err := n.Delete(ctx, first.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	list, _ = n.List(ctx)
	if got, want := texts(list), []string{"Ideas: a better clock"}; !slices.Equal(got, want) {
		t.Errorf("List() after Delete = %q, want %q", got, want)
	}
	if _, err := n.Update(ctx, first.ID, "Shopping", "too late"); err == nil {
		t.Error("Update() of a deleted note succeeded, want error")
	}
}

func TestNotebookReject(t *testing.T) {
	t.Parallel()

	n := newNotebook(t)
	ctx := t.Context()
	tests := []struct {
		name  string
		title string
		text  string
	}{
		{"blank text", "Title", " \n "},
		{"blank title", " ", "text"},
		{"title on two lines", "one\ntwo", "text"},
		{"title too long", strings.Repeat("x", 201), "text"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if _, err := n.Add(ctx, tt.title, tt.text); err == nil {
				t.Errorf("Add(%q, %q) succeeded, want error", tt.title, tt.text)
			}
		})
	}

	note, err := n.Add(ctx, "Title", "text")
	if err != nil {
		t.Fatal(err)
	}
	for _, tt := range tests {
		t.Run("update "+tt.name, func(t *testing.T) {
			t.Parallel()
			if _, err := n.Update(ctx, note.ID, tt.title, tt.text); err == nil {
				t.Errorf("Update(%q, %q) succeeded, want error", tt.title, tt.text)
			}
		})
	}
}
