package notebook_test

import (
	"maps"
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

	first, err := n.Add(ctx, "  Groceries ", "  milk, eggs\n", nil)
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	if first.Title != "Groceries" || first.Text != "milk, eggs" || first.ID == "" {
		t.Errorf("Add() = %+v, want trimmed title and text and an ID", first)
	}
	if !first.Created.Equal(first.Updated) {
		t.Errorf("new note Created %v != Updated %v", first.Created, first.Updated)
	}
	if _, err := n.Add(ctx, "Ideas", "a better clock", nil); err != nil {
		t.Fatal(err)
	}

	list, err = n.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := texts(list), []string{"Groceries: milk, eggs", "Ideas: a better clock"}; !slices.Equal(got, want) {
		t.Errorf("List() = %q, want %q", got, want)
	}

	updated, err := n.Update(ctx, first.ID, "Shopping", "bread", nil)
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
	if _, err := n.Update(ctx, first.ID, "Shopping", "too late", nil); err == nil {
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
			if _, err := n.Add(ctx, tt.title, tt.text, nil); err == nil {
				t.Errorf("Add(%q, %q) succeeded, want error", tt.title, tt.text)
			}
		})
	}

	note, err := n.Add(ctx, "Title", "text", nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, tt := range tests {
		t.Run("update "+tt.name, func(t *testing.T) {
			t.Parallel()
			if _, err := n.Update(ctx, note.ID, tt.title, tt.text, nil); err == nil {
				t.Errorf("Update(%q, %q) succeeded, want error", tt.title, tt.text)
			}
		})
	}
}

func TestNotebookTags(t *testing.T) {
	t.Parallel()

	n := newNotebook(t)
	ctx := t.Context()

	untagged, err := n.Add(ctx, "Plain", "text", nil)
	if err != nil {
		t.Fatal(err)
	}
	if untagged.Tags == nil || len(untagged.Tags) != 0 {
		t.Errorf("Add() with no tags: Tags = %#v, want an empty, non-nil list", untagged.Tags)
	}

	note, err := n.Add(ctx, "Tagged", "text", []string{" Work ", "ideas", "work", "", "  side   project "})
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	if want := []string{"ideas", "side project", "work"}; !slices.Equal(note.Tags, want) {
		t.Errorf("Add() Tags = %q, want %q", note.Tags, want)
	}

	updated, err := n.Update(ctx, note.ID, "Tagged", "text", []string{"Home"})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if want := []string{"home"}; !slices.Equal(updated.Tags, want) {
		t.Errorf("Update() Tags = %q, want %q", updated.Tags, want)
	}
	list, err := n.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, got := range list {
		if got.ID == note.ID && !slices.Equal(got.Tags, []string{"home"}) {
			t.Errorf("List() Tags = %q, want [home]", got.Tags)
		}
	}

	tooMany := make([]string, 11)
	for i := range tooMany {
		tooMany[i] = strings.Repeat("t", i+1)
	}
	tests := []struct {
		name string
		tags []string
	}{
		{"comma", []string{"a,b"}},
		{"too long", []string{strings.Repeat("x", 33)}},
		{"too many", tooMany},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if _, err := n.Add(ctx, "Title", "text", tt.tags); err == nil {
				t.Errorf("Add() with tags %q succeeded, want error", tt.tags)
			}
			if _, err := n.Update(ctx, note.ID, "Title", "text", tt.tags); err == nil {
				t.Errorf("Update() with tags %q succeeded, want error", tt.tags)
			}
		})
	}
}

func TestTagColors(t *testing.T) {
	t.Parallel()

	n := newNotebook(t)
	ctx := t.Context()

	colors, err := n.TagColors(ctx)
	if err != nil || len(colors) != 0 {
		t.Fatalf("TagColors() with none set = %v, %v; want empty, nil", colors, err)
	}
	if tag, err := n.SetTagColor(ctx, " Work ", "blue"); err != nil || tag != "work" {
		t.Fatalf("SetTagColor() = %q, %v; want work, nil", tag, err)
	}
	if _, err := n.SetTagColor(ctx, "ideas", "green"); err != nil {
		t.Fatal(err)
	}
	if _, err := n.SetTagColor(ctx, "work", "red"); err != nil {
		t.Fatal(err)
	}
	colors, err = n.TagColors(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if want := map[string]string{"work": "red", "ideas": "green"}; !maps.Equal(colors, want) {
		t.Errorf("TagColors() = %v, want %v", colors, want)
	}

	// An empty colour takes the colour away.
	if _, err := n.SetTagColor(ctx, "work", ""); err != nil {
		t.Fatalf("SetTagColor() clearing error = %v", err)
	}
	colors, _ = n.TagColors(ctx)
	if want := map[string]string{"ideas": "green"}; !maps.Equal(colors, want) {
		t.Errorf("TagColors() after clearing = %v, want %v", colors, want)
	}

	tests := []struct {
		name  string
		tag   string
		color string
	}{
		{"unknown colour", "work", "chartreuse"},
		{"empty tag", "  ", "blue"},
		{"comma in tag", "a,b", "blue"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if _, err := n.SetTagColor(ctx, tt.tag, tt.color); err == nil {
				t.Errorf("SetTagColor(%q, %q) succeeded, want error", tt.tag, tt.color)
			}
		})
	}
}
