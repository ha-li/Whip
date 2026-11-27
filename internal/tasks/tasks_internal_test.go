package tasks

import (
	"errors"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/ha-li/Whip/internal/db"
)

// newTestTasks returns a Tasks on a temporary database whose clock moves a
// minute forward on every reading.
func newTestTasks(t *testing.T) *Tasks {
	t.Helper()
	database, err := db.Open(filepath.Join(t.TempDir(), "data.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	clock := time.Date(2026, 11, 26, 9, 0, 0, 0, time.UTC)
	tasks, err := newTasks(t.Context(), database, func() time.Time {
		clock = clock.Add(time.Minute)
		return clock
	})
	if err != nil {
		t.Fatal(err)
	}
	return tasks
}

// textsOf returns a function that takes a list call's results and returns
// the tasks' texts, in order, failing t on error.
func textsOf(t *testing.T) func([]Task, error) []string {
	return func(tasks []Task, err error) []string {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
		out := []string{}
		for _, task := range tasks {
			out = append(out, task.Text)
		}
		return out
	}
}

func TestTasks(t *testing.T) {
	t.Parallel()

	tasks := newTestTasks(t)
	texts := textsOf(t)
	ctx := t.Context()

	if got := texts(tasks.List(ctx)); len(got) != 0 {
		t.Fatalf("List() with no tasks = %q, want none", got)
	}
	var added []Task
	for _, text := range []string{"  buy milk\n", "file taxes", "call mum"} {
		task, err := tasks.Add(ctx, text)
		if err != nil {
			t.Fatalf("Add(%q) error = %v", text, err)
		}
		added = append(added, task)
	}
	if added[0].Text != "buy milk" || added[0].ID == "" || added[0].Completed != nil {
		t.Errorf("Add() = %+v, want trimmed text, an ID and not completed", added[0])
	}
	// Open tasks are listed in the order they were added.
	if got, want := texts(tasks.List(ctx)), []string{"buy milk", "file taxes", "call mum"}; !slices.Equal(got, want) {
		t.Errorf("List() = %q, want %q", got, want)
	}

	// Completed out of order: the latest completion is listed first.
	for _, task := range []Task{added[2], added[0]} {
		done, err := tasks.Complete(ctx, task.ID)
		if err != nil {
			t.Fatalf("Complete() error = %v", err)
		}
		if done.Completed == nil {
			t.Errorf("Complete() = %+v, want a completion time", done)
		}
	}
	if got, want := texts(tasks.List(ctx)), []string{"file taxes"}; !slices.Equal(got, want) {
		t.Errorf("List() after Complete = %q, want %q", got, want)
	}
	if got, want := texts(tasks.ListCompleted(ctx)), []string{"buy milk", "call mum"}; !slices.Equal(got, want) {
		t.Errorf("ListCompleted() = %q, want %q", got, want)
	}

	// Completing again keeps the first completion time.
	first, _ := tasks.tasks.Get(ctx, added[0].ID)
	again, err := tasks.Complete(ctx, added[0].ID)
	if err != nil || !again.Completed.Equal(*first.Completed) {
		t.Errorf("second Complete() = %+v, %v; want completion time %v kept", again, err, *first.Completed)
	}

	// Reopening puts the task back among the open ones, in its original place.
	reopened, err := tasks.Reopen(ctx, added[0].ID)
	if err != nil || reopened.Completed != nil {
		t.Fatalf("Reopen() = %+v, %v; want an open task", reopened, err)
	}
	if got, want := texts(tasks.List(ctx)), []string{"buy milk", "file taxes"}; !slices.Equal(got, want) {
		t.Errorf("List() after Reopen = %q, want %q", got, want)
	}
	if got, err := tasks.Reopen(ctx, added[1].ID); err != nil || got.Completed != nil {
		t.Errorf("Reopen() of an open task = %+v, %v; want it unchanged", got, err)
	}
}

func TestDelete(t *testing.T) {
	t.Parallel()

	tasks := newTestTasks(t)
	texts := textsOf(t)
	ctx := t.Context()
	open, err := tasks.Add(ctx, "still to do")
	if err != nil {
		t.Fatal(err)
	}
	done, err := tasks.Add(ctx, "done")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tasks.Complete(ctx, done.ID); err != nil {
		t.Fatal(err)
	}

	if err := tasks.Delete(ctx, open.ID); err == nil {
		t.Error("Delete() of an open task succeeded, want error")
	}
	if err := tasks.Delete(ctx, done.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	// Gone from the database, not just from the lists.
	if _, err := tasks.tasks.Get(ctx, done.ID); !errors.Is(err, db.ErrNotFound) {
		t.Errorf("Get() after Delete error = %v, want ErrNotFound", err)
	}
	if got := texts(tasks.ListCompleted(ctx)); len(got) != 0 {
		t.Errorf("ListCompleted() after Delete = %q, want none", got)
	}
	if got, want := texts(tasks.List(ctx)), []string{"still to do"}; !slices.Equal(got, want) {
		t.Errorf("List() after Delete = %q, want %q", got, want)
	}
	if err := tasks.Delete(ctx, done.ID); err != nil {
		t.Errorf("Delete() of a deleted task error = %v, want nil", err)
	}
	if _, err := tasks.Complete(ctx, done.ID); err == nil {
		t.Error("Complete() of a deleted task succeeded, want error")
	}
}

func TestAddRejectsBlank(t *testing.T) {
	t.Parallel()

	if _, err := newTestTasks(t).Add(t.Context(), " \n\t "); err == nil {
		t.Error("Add() with blank text succeeded, want error")
	}
}
