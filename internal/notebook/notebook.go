// Package notebook keeps the user's general notes: each has a title, text
// and tags, and each tag can have a colour from package palette. It owns the
// notebook and notebook_tags collections in the app's database (see package
// db).
package notebook

import (
	"cmp"
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/ha-li/Whip/internal/db"
	"github.com/ha-li/Whip/internal/palette"
)

// The collections Notebook owns in the app's database: the notes, and the
// colours given to tags.
const (
	collection     = "notebook"
	tagsCollection = "notebook_tags"
)

// Limits on a note's title and tags.
const (
	// maxTitle is the longest title, in characters, a note can have.
	maxTitle = 200
	// maxTags is the most tags a note can have.
	maxTags = 10
	// maxTag is the longest tag, in characters.
	maxTag = 32
)

// Note is a dated note with a title.
type Note struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Text  string `json:"text"`
	// Tags label the note, e.g. "work". They are lower case, sorted and
	// distinct; never nil in notes Notebook returns.
	Tags []string `json:"tags"`
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

// tagColor is the colour given to a tag, stored in notebook_tags.
type tagColor struct {
	Tag string `json:"tag"`
	// Color is the name of a colour in package palette.
	Color string `json:"color"`
}

// DocumentID implements db.Document.
func (t tagColor) DocumentID() string { return t.Tag }

// Notebook keeps the user's notes. It is bound to the frontend.
type Notebook struct {
	notes *db.Collection[Note]
	tags  *db.Collection[tagColor]
	now   func() time.Time
}

// New returns a Notebook that keeps its notes in database.
func New(ctx context.Context, database *db.DB) (*Notebook, error) {
	return newNotebook(ctx, database, time.Now)
}

func newNotebook(ctx context.Context, database *db.DB, now func() time.Time) (*Notebook, error) {
	notes, err := db.OpenCollection[Note](ctx, database, collection)
	if err != nil {
		return nil, err
	}
	tags, err := db.OpenCollection[tagColor](ctx, database, tagsCollection)
	if err != nil {
		return nil, err
	}
	return &Notebook{notes: notes, tags: tags, now: now}, nil
}

// List returns all notes that haven't been deleted, newest first.
func (n *Notebook) List(ctx context.Context) ([]Note, error) {
	return n.list(ctx, false, func(note Note) time.Time { return note.Created })
}

// ListDeleted returns the deleted notes, most recently deleted first.
func (n *Notebook) ListDeleted(ctx context.Context) ([]Note, error) {
	return n.list(ctx, true, func(note Note) time.Time { return *note.Deleted })
}

// list returns the notes that are deleted (or not, as deleted says), latest
// first by the time when returns.
func (n *Notebook) list(ctx context.Context, deleted bool, when func(Note) time.Time) ([]Note, error) {
	all, err := n.notes.All(ctx)
	if err != nil {
		return nil, err
	}
	notes := slices.DeleteFunc(all, func(note Note) bool { return (note.Deleted != nil) != deleted })
	for i := range notes {
		notes[i] = withTags(notes[i])
	}
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

// Add saves a new note with the given tags and returns it.
func (n *Notebook) Add(ctx context.Context, title, text string, tags []string) (Note, error) {
	title, text, err := clean(title, text)
	if err != nil {
		return Note{}, err
	}
	tags, err = cleanTags(tags)
	if err != nil {
		return Note{}, err
	}
	now := n.now().UTC()
	note := Note{ID: rand.Text(), Title: title, Text: text, Tags: tags, Created: now, Updated: now}
	if err := n.notes.Put(ctx, note); err != nil {
		return Note{}, fmt.Errorf("save note: %w", err)
	}
	return note, nil
}

// Update replaces the title, text and tags of the note with the given ID and
// returns it.
func (n *Notebook) Update(ctx context.Context, id, title, text string, tags []string) (Note, error) {
	title, text, err := clean(title, text)
	if err != nil {
		return Note{}, err
	}
	tags, err = cleanTags(tags)
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
	note.Title = title
	note.Text = text
	note.Tags = tags
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
func (n *Notebook) Delete(ctx context.Context, id string) error {
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
func (n *Notebook) Restore(ctx context.Context, id string) (Note, error) {
	note, err := n.notes.Get(ctx, id)
	if errors.Is(err, db.ErrNotFound) {
		return Note{}, errors.New("the note no longer exists")
	}
	if err != nil {
		return Note{}, err
	}
	if note.Deleted == nil {
		return withTags(note), nil
	}
	note.Deleted = nil
	if err := n.notes.Put(ctx, note); err != nil {
		return Note{}, fmt.Errorf("restore note: %w", err)
	}
	return withTags(note), nil
}

// EmptyTrash permanently removes every deleted note from the database and
// returns how many it removed. Notes that aren't deleted are kept.
func (n *Notebook) EmptyTrash(ctx context.Context) (int, error) {
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

// clean returns title and text with surrounding whitespace removed, or an
// error if either is empty or the title is on several lines or too long.
func clean(title, text string) (string, string, error) {
	title = strings.TrimSpace(title)
	text = strings.TrimSpace(text)
	switch {
	case title == "":
		return "", "", errors.New("the note needs a title")
	case strings.ContainsAny(title, "\r\n"):
		return "", "", errors.New("the title must be on one line")
	case utf8.RuneCountInString(title) > maxTitle:
		return "", "", fmt.Errorf("the title is longer than %d characters", maxTitle)
	case text == "":
		return "", "", errors.New("the note is empty")
	}
	return title, text, nil
}

// TagColors returns the colour given to each tag that has one, as tag →
// palette colour name. Tags without a colour aren't in the map.
func (n *Notebook) TagColors(ctx context.Context) (map[string]string, error) {
	all, err := n.tags.All(ctx)
	if err != nil {
		return nil, err
	}
	colors := make(map[string]string, len(all))
	for _, t := range all {
		colors[t.Tag] = t.Color
	}
	return colors, nil
}

// SetTagColor gives tag the palette colour with the given name, or takes its
// colour away if color is "". The tag needn't be on any note yet. It returns
// the tag as stored (see Note.Tags).
func (n *Notebook) SetTagColor(ctx context.Context, tag, color string) (string, error) {
	tags, err := cleanTags([]string{tag})
	if err != nil {
		return "", err
	}
	if len(tags) == 0 {
		return "", errors.New("the tag is empty")
	}
	tag = tags[0]
	if color == "" {
		if err := n.tags.Delete(ctx, tag); err != nil {
			return "", fmt.Errorf("clear tag colour: %w", err)
		}
		return tag, nil
	}
	if _, ok := palette.Lookup(color); !ok {
		return "", fmt.Errorf("%q is not a palette colour", color)
	}
	if err := n.tags.Put(ctx, tagColor{Tag: tag, Color: color}); err != nil {
		return "", fmt.Errorf("save tag colour: %w", err)
	}
	return tag, nil
}

// cleanTags returns tags trimmed, lower-cased, with runs of spaces made one,
// without blanks or repeats, and sorted; or an error if a tag has a comma or
// is too long, or there are too many. It never returns nil tags.
func cleanTags(tags []string) ([]string, error) {
	out := []string{}
	for _, tag := range tags {
		tag = strings.ToLower(strings.Join(strings.Fields(tag), " "))
		switch {
		case tag == "":
			continue
		case strings.Contains(tag, ","):
			return nil, fmt.Errorf("the tag %q has a comma", tag)
		case utf8.RuneCountInString(tag) > maxTag:
			return nil, fmt.Errorf("the tag %q is longer than %d characters", tag, maxTag)
		}
		out = append(out, tag)
	}
	slices.Sort(out)
	out = slices.Compact(out)
	if len(out) > maxTags {
		return nil, fmt.Errorf("a note can have at most %d tags", maxTags)
	}
	return out, nil
}

// withTags returns note with empty, non-nil tags if it has none, e.g. one
// saved before notes had tags, so the frontend gets a list rather than null.
func withTags(note Note) Note {
	if note.Tags == nil {
		note.Tags = []string{}
	}
	return note
}
