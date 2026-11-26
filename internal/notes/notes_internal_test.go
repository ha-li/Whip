package notes

import (
	"path/filepath"
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
