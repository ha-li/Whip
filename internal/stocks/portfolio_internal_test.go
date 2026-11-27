package stocks

import (
	"errors"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/ha-li/Whip/internal/db"
)

func TestListNewestFirst(t *testing.T) {
	t.Parallel()

	database, err := db.Open(filepath.Join(t.TempDir(), "data.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	// Each call to the clock is a minute later than the one before.
	clock := time.Date(2026, 11, 26, 9, 0, 0, 0, time.UTC)
	now := func() time.Time {
		clock = clock.Add(time.Minute)
		return clock
	}
	n, err := newNotes(t.Context(), database, now)
	if err != nil {
		t.Fatal(err)
	}

	ctx := t.Context()
	for _, add := range []struct{ symbol, text string }{
		{"AAPL", "first"},
		{"MSFT", "second"},
		{"AAPL", "third"},
	} {
		if _, err := n.Add(ctx, add.symbol, add.text); err != nil {
			t.Fatal(err)
		}
	}
	// Editing a note doesn't move it: the list is ordered by when notes were
	// added, whatever their ticker.
	list, err := n.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := n.Update(ctx, list[2].ID, "first, edited"); err != nil {
		t.Fatal(err)
	}

	list, err = n.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, note := range list {
		got = append(got, note.Text)
	}
	want := []string{"third", "second", "first, edited"}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] || got[2] != want[2] {
		t.Errorf("List() = %q, want %q", got, want)
	}
}

func TestDeleteKeepsNote(t *testing.T) {
	t.Parallel()

	database, err := db.Open(filepath.Join(t.TempDir(), "data.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	clock := time.Date(2026, 11, 26, 9, 0, 0, 0, time.UTC)
	now := func() time.Time {
		clock = clock.Add(time.Minute)
		return clock
	}
	n, err := newNotes(t.Context(), database, now)
	if err != nil {
		t.Fatal(err)
	}
	ctx := t.Context()

	note, err := n.Add(ctx, "AAPL", "bought at 182")
	if err != nil {
		t.Fatal(err)
	}
	if err := n.Delete(ctx, note.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	stored, err := n.notes.Get(ctx, note.ID)
	if err != nil {
		t.Fatalf("Get() after Delete error = %v, want the note still stored", err)
	}
	want := time.Date(2026, 11, 26, 9, 2, 0, 0, time.UTC) // the clock's second reading
	if stored.Deleted == nil || !stored.Deleted.Equal(want) || stored.Text != "bought at 182" {
		t.Fatalf("stored note = %+v, want the same text and Deleted %v", stored, want)
	}

	// Deleting again changes nothing: the note keeps its first deletion time.
	if err := n.Delete(ctx, note.ID); err != nil {
		t.Errorf("second Delete() error = %v, want nil", err)
	}
	if again, _ := n.notes.Get(ctx, note.ID); again.Deleted == nil || !again.Deleted.Equal(want) {
		t.Errorf("Deleted after a second Delete = %v, want %v", again.Deleted, want)
	}
	if err := n.Delete(ctx, "no-such-note"); err != nil {
		t.Errorf("Delete() of a missing ID error = %v, want nil", err)
	}
}

func TestListDeletedAndRestore(t *testing.T) {
	t.Parallel()

	database, err := db.Open(filepath.Join(t.TempDir(), "data.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	// Each call to the clock is a minute later than the one before.
	clock := time.Date(2026, 11, 26, 9, 0, 0, 0, time.UTC)
	now := func() time.Time {
		clock = clock.Add(time.Minute)
		return clock
	}
	n, err := newNotes(t.Context(), database, now)
	if err != nil {
		t.Fatal(err)
	}
	ctx := t.Context()

	add := func(text string) Note {
		t.Helper()
		note, err := n.Add(ctx, "AAPL", text)
		if err != nil {
			t.Fatal(err)
		}
		return note
	}
	textsOf := func(list []Note, err error) []string {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
		out := []string{}
		for _, note := range list {
			out = append(out, note.Text)
		}
		return out
	}
	first, second, third := add("first"), add("second"), add("third")

	if got := textsOf(n.ListDeleted(ctx)); len(got) != 0 {
		t.Fatalf("ListDeleted() with nothing deleted = %q, want none", got)
	}
	// Deleted out of creation order: the archive lists the latest deletion first.
	for _, note := range []Note{second, first} {
		if err := n.Delete(ctx, note.ID); err != nil {
			t.Fatal(err)
		}
	}
	if got, want := textsOf(n.ListDeleted(ctx)), []string{"first", "second"}; !slices.Equal(got, want) {
		t.Errorf("ListDeleted() = %q, want %q", got, want)
	}
	if got, want := textsOf(n.List(ctx)), []string{"third"}; !slices.Equal(got, want) {
		t.Errorf("List() = %q, want %q", got, want)
	}

	restored, err := n.Restore(ctx, second.ID)
	if err != nil {
		t.Fatalf("Restore() error = %v", err)
	}
	if restored.Deleted != nil || restored.Text != "second" || !restored.Updated.Equal(second.Updated) {
		t.Errorf("Restore() = %+v, want the note undeleted and otherwise unchanged", restored)
	}
	// Back in its original place, by creation time.
	if got, want := textsOf(n.List(ctx)), []string{"third", "second"}; !slices.Equal(got, want) {
		t.Errorf("List() after Restore = %q, want %q", got, want)
	}
	if got, want := textsOf(n.ListDeleted(ctx)), []string{"first"}; !slices.Equal(got, want) {
		t.Errorf("ListDeleted() after Restore = %q, want %q", got, want)
	}
	if _, err := n.Update(ctx, second.ID, "second, edited"); err != nil {
		t.Errorf("Update() of a restored note error = %v, want nil", err)
	}

	if got, err := n.Restore(ctx, third.ID); err != nil || got.Text != "third" || got.Deleted != nil {
		t.Errorf("Restore() of a note that isn't deleted = %+v, %v; want it unchanged", got, err)
	}
	if _, err := n.Restore(ctx, "no-such-note"); err == nil {
		t.Error("Restore() of a missing ID succeeded, want error")
	}
}

func TestEmptyTrash(t *testing.T) {
	t.Parallel()

	database, err := db.Open(filepath.Join(t.TempDir(), "data.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	n, err := newNotes(t.Context(), database, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	ctx := t.Context()

	if got, err := n.EmptyTrash(ctx); err != nil || got != 0 {
		t.Fatalf("EmptyTrash() of an empty trash = %d, %v; want 0, nil", got, err)
	}
	var notes []Note
	for _, text := range []string{"keep", "trash 1", "trash 2"} {
		note, err := n.Add(ctx, "AAPL", text)
		if err != nil {
			t.Fatal(err)
		}
		notes = append(notes, note)
	}
	for _, note := range notes[1:] {
		if err := n.Delete(ctx, note.ID); err != nil {
			t.Fatal(err)
		}
	}

	if got, err := n.EmptyTrash(ctx); err != nil || got != 2 {
		t.Fatalf("EmptyTrash() = %d, %v; want 2, nil", got, err)
	}
	// Gone from the database, not just from the lists.
	for _, note := range notes[1:] {
		if _, err := n.notes.Get(ctx, note.ID); !errors.Is(err, db.ErrNotFound) {
			t.Errorf("Get(%q) after EmptyTrash error = %v, want ErrNotFound", note.Text, err)
		}
	}
	if trash, err := n.ListDeleted(ctx); err != nil || len(trash) != 0 {
		t.Errorf("ListDeleted() after EmptyTrash = %v, %v; want none", trash, err)
	}
	if list, err := n.List(ctx); err != nil || len(list) != 1 || list[0].Text != "keep" {
		t.Errorf("List() after EmptyTrash = %v, %v; want just the kept note", list, err)
	}
	if _, err := n.Restore(ctx, notes[1].ID); err == nil {
		t.Error("Restore() of an emptied note succeeded, want error")
	}
}
