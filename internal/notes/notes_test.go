package notes_test

import (
	"path/filepath"
	"slices"
	"testing"

	"github.com/ha-li/Whip/internal/db"
	"github.com/ha-li/Whip/internal/notes"
)

// newNotes returns a Notes on a database in a temporary directory.
func newNotes(t *testing.T) *notes.Notes {
	t.Helper()
	database, err := db.Open(filepath.Join(t.TempDir(), "data.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	n, err := notes.New(t.Context(), database)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	return n
}

// texts returns the notes' texts as "SYMBOL: text", sorted.
func texts(list []notes.Note) []string {
	out := make([]string, len(list))
	for i, n := range list {
		out[i] = n.Symbol + ": " + n.Text
	}
	slices.Sort(out)
	return out
}

func TestNotes(t *testing.T) {
	t.Parallel()

	n := newNotes(t)
	ctx := t.Context()

	list, err := n.List(ctx)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if list == nil || len(list) != 0 {
		t.Fatalf("List() with no notes = %#v, want an empty, non-nil list", list)
	}

	first, err := n.Add(ctx, " aapl ", "  bought at 182\n")
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	if first.Symbol != "AAPL" || first.Text != "bought at 182" || first.ID == "" {
		t.Errorf("Add() = %+v, want symbol AAPL, trimmed text and an ID", first)
	}
	if !first.Created.Equal(first.Updated) {
		t.Errorf("new note Created %v != Updated %v", first.Created, first.Updated)
	}
	if _, err := n.Add(ctx, "MSFT", "watch earnings"); err != nil {
		t.Fatal(err)
	}

	// Every ticker's notes. (Their order is tested with a fixed clock in
	// TestListNewestFirst.)
	list, err = n.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := texts(list), []string{"AAPL: bought at 182", "MSFT: watch earnings"}; !slices.Equal(got, want) {
		t.Errorf("List() = %q, want %q", got, want)
	}

	updated, err := n.Update(ctx, first.ID, "sold at 210")
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if updated.Text != "sold at 210" || updated.Symbol != "AAPL" || !updated.Created.Equal(first.Created) || updated.Updated.Before(first.Updated) {
		t.Errorf("Update() = %+v, want new text, same symbol and Created, later or equal Updated", updated)
	}

	if err := n.Delete(ctx, first.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	list, _ = n.List(ctx)
	if got, want := texts(list), []string{"MSFT: watch earnings"}; !slices.Equal(got, want) {
		t.Errorf("List() after Delete = %q, want %q", got, want)
	}
	if _, err := n.Update(ctx, first.ID, "too late"); err == nil {
		t.Error("Update() of a deleted note succeeded, want error")
	}
}

func TestNotesReject(t *testing.T) {
	t.Parallel()

	n := newNotes(t)
	ctx := t.Context()
	tests := []struct {
		name   string
		symbol string
		text   string
	}{
		{"blank text", "AAPL", " \n "},
		{"empty symbol", " ", "text"},
		{"path in symbol", "../AAPL", "text"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if _, err := n.Add(ctx, tt.symbol, tt.text); err == nil {
				t.Errorf("Add(%q, %q) succeeded, want error", tt.symbol, tt.text)
			}
		})
	}

	note, err := n.Add(ctx, "AAPL", "text")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := n.Update(ctx, note.ID, "   "); err == nil {
		t.Error("Update() with blank text succeeded, want error")
	}
}
