// Package portfolio keeps the user's notes about stocks: several notes per
// ticker symbol. It owns the stock_notes collection in the app's database
// (see package db).
package portfolio

import (
	"cmp"
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/ha-li/Whip/internal/db"
)

// collection is the collection Notes owns in the app's database. Its name
// predates this package and is kept so existing notes stay readable.
const collection = "stock_portfolio"

// Note is a dated note about a stock.
type Note struct {
	ID string `json:"id"`
	// Symbol is the ticker the note is about, e.g. "AAPL".
	Symbol string `json:"symbol"`
	Text   string `json:"text"`
	// Created and Updated are when the note was added and last edited.
	Created time.Time `json:"created"`
	Updated time.Time `json:"updated"`
	// Deleted is when the note was deleted, or nil if it wasn't. Deleted
	// notes stay in the database but List leaves them out and they can't be
	// edited.
	Deleted *time.Time `json:"deleted,omitempty"`
}

// DocumentID implements db.Document.
func (n Note) DocumentID() string { return n.ID }

// Notes keeps the user's notes. It is bound to the frontend.
type Notes struct {
	notes *db.Collection[Note]
	now   func() time.Time
}

// New returns a Notes that keeps its notes in database.
func New(ctx context.Context, database *db.DB) (*Notes, error) {
	return newNotes(ctx, database, time.Now)
}

func newNotes(ctx context.Context, database *db.DB, now func() time.Time) (*Notes, error) {
	c, err := db.OpenCollection[Note](ctx, database, collection)
	if err != nil {
		return nil, err
	}
	return &Notes{notes: c, now: now}, nil
}

// List returns all notes that haven't been deleted, newest first.
func (n *Notes) List(ctx context.Context) ([]Note, error) {
	return n.list(ctx, false, func(note Note) time.Time { return note.Created })
}

// ListDeleted returns the deleted notes, most recently deleted first.
func (n *Notes) ListDeleted(ctx context.Context) ([]Note, error) {
	return n.list(ctx, true, func(note Note) time.Time { return *note.Deleted })
}

// list returns the notes that are deleted (or not, as deleted says), latest
// first by the time when returns.
func (n *Notes) list(ctx context.Context, deleted bool, when func(Note) time.Time) ([]Note, error) {
	all, err := n.notes.All(ctx)
	if err != nil {
		return nil, err
	}
	notes := slices.DeleteFunc(all, func(note Note) bool { return (note.Deleted != nil) != deleted })
	slices.SortFunc(notes, func(a, b Note) int {
		if c := when(b).Compare(when(a)); c != 0 {
			return c
		}
		return cmp.Compare(b.ID, a.ID)
	})
	if notes == nil {
		notes = []Note{} // an empty list, not null, for the frontend
	}
	return notes, nil
}

// Add saves a new note about symbol and returns it.
func (n *Notes) Add(ctx context.Context, symbol, text string) (Note, error) {
	symbol, err := normalizeSymbol(symbol)
	if err != nil {
		return Note{}, err
	}
	text, err = noteText(text)
	if err != nil {
		return Note{}, err
	}
	now := n.now().UTC()
	note := Note{ID: rand.Text(), Symbol: symbol, Text: text, Created: now, Updated: now}
	if err := n.notes.Put(ctx, note); err != nil {
		return Note{}, fmt.Errorf("save note: %w", err)
	}
	return note, nil
}

// Update replaces the text of the note with the given ID and returns it.
func (n *Notes) Update(ctx context.Context, id, text string) (Note, error) {
	text, err := noteText(text)
	if err != nil {
		return Note{}, err
	}
	note, err := n.notes.Get(ctx, id)
	if errors.Is(err, db.ErrNotFound) || (err == nil && note.Deleted != nil) {
		return Note{}, errors.New("the note no longer exists")
	}
	if err != nil {
		return Note{}, err
	}
	note.Text = text
	note.Updated = n.now().UTC()
	if err := n.notes.Put(ctx, note); err != nil {
		return Note{}, fmt.Errorf("save note: %w", err)
	}
	return note, nil
}

// Delete marks the note with the given ID as deleted. The note stays in the
// database with the time it was deleted, but List no longer returns it.
// Deleting a note that's already deleted, or doesn't exist, is not an error
// and changes nothing.
func (n *Notes) Delete(ctx context.Context, id string) error {
	note, err := n.notes.Get(ctx, id)
	if errors.Is(err, db.ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if note.Deleted != nil {
		return nil
	}
	now := n.now().UTC()
	note.Deleted = &now
	if err := n.notes.Put(ctx, note); err != nil {
		return fmt.Errorf("delete note: %w", err)
	}
	return nil
}

// Restore undeletes the note with the given ID and returns it: List returns
// it again, in its original place, and it can be edited. Restoring a note
// that isn't deleted returns it unchanged.
func (n *Notes) Restore(ctx context.Context, id string) (Note, error) {
	note, err := n.notes.Get(ctx, id)
	if errors.Is(err, db.ErrNotFound) {
		return Note{}, errors.New("the note no longer exists")
	}
	if err != nil {
		return Note{}, err
	}
	if note.Deleted == nil {
		return note, nil
	}
	note.Deleted = nil
	if err := n.notes.Put(ctx, note); err != nil {
		return Note{}, fmt.Errorf("restore note: %w", err)
	}
	return note, nil
}

// EmptyTrash permanently removes every deleted note from the database and
// returns how many it removed. Notes that aren't deleted are kept.
func (n *Notes) EmptyTrash(ctx context.Context) (int, error) {
	trash, err := n.ListDeleted(ctx)
	if err != nil {
		return 0, err
	}
	ids := make([]string, len(trash))
	for i, note := range trash {
		ids[i] = note.ID
	}
	if err := n.notes.Delete(ctx, ids...); err != nil {
		return 0, fmt.Errorf("empty trash: %w", err)
	}
	return len(ids), nil
}

// symbolPattern accepts ticker symbols such as "IBM", "BRK.B" or "TSCO.LON".
var symbolPattern = regexp.MustCompile(`^[A-Z0-9][A-Z0-9.\-]{0,14}$`)

// normalizeSymbol returns symbol trimmed and upper-cased, or an error if it
// isn't a ticker symbol.
func normalizeSymbol(symbol string) (string, error) {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	if !symbolPattern.MatchString(symbol) {
		return "", fmt.Errorf("%q is not a ticker symbol", symbol)
	}
	return symbol, nil
}

// noteText returns text with surrounding whitespace removed, or an error if
// nothing is left.
func noteText(text string) (string, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return "", errors.New("the note is empty")
	}
	return text, nil
}
