// Package tasks keeps the user's to-do list. It owns the tasks collection in
// the app's database (see package db).
package tasks

import (
	"cmp"
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/ha-li/Whip/internal/db"
)

// collection is the collection Tasks owns in the app's database.
const collection = "tasks"

// Task is one thing to do.
type Task struct {
	ID   string `json:"id"`
	Text string `json:"text"`
	// Created is when the task was added.
	Created time.Time `json:"created"`
	// Completed is when the task was marked done, or nil while it's open.
	Completed *time.Time `json:"completed,omitempty"`
}

// DocumentID implements db.Document.
func (t Task) DocumentID() string { return t.ID }

// Tasks keeps the user's tasks. It is bound to the frontend.
type Tasks struct {
	tasks *db.Collection[Task]
	now   func() time.Time
}

// errNoTask is returned for an ID with no task.
var errNoTask = errors.New("the task no longer exists")

// New returns a Tasks that keeps its tasks in database.
func New(ctx context.Context, database *db.DB) (*Tasks, error) {
	return newTasks(ctx, database, time.Now)
}

func newTasks(ctx context.Context, database *db.DB, now func() time.Time) (*Tasks, error) {
	c, err := db.OpenCollection[Task](ctx, database, collection)
	if err != nil {
		return nil, err
	}
	return &Tasks{tasks: c, now: now}, nil
}

// List returns the open tasks in the order they were added.
func (t *Tasks) List(ctx context.Context) ([]Task, error) {
	tasks, err := t.list(ctx, false)
	if err != nil {
		return nil, err
	}
	slices.SortFunc(tasks, func(a, b Task) int {
		if c := a.Created.Compare(b.Created); c != 0 {
			return c
		}
		return cmp.Compare(a.ID, b.ID)
	})
	return tasks, nil
}

// ListCompleted returns the completed tasks, most recently completed first.
func (t *Tasks) ListCompleted(ctx context.Context) ([]Task, error) {
	tasks, err := t.list(ctx, true)
	if err != nil {
		return nil, err
	}
	slices.SortFunc(tasks, func(a, b Task) int {
		if c := b.Completed.Compare(*a.Completed); c != 0 {
			return c
		}
		return cmp.Compare(b.ID, a.ID)
	})
	return tasks, nil
}

// list returns the tasks that are completed (or open, as completed says), in
// no particular order.
func (t *Tasks) list(ctx context.Context, completed bool) ([]Task, error) {
	all, err := t.tasks.All(ctx)
	if err != nil {
		return nil, err
	}
	tasks := slices.DeleteFunc(all, func(task Task) bool { return (task.Completed != nil) != completed })
	if tasks == nil {
		tasks = []Task{} // an empty list, not null, for the frontend
	}
	return tasks, nil
}

// Add saves a new open task and returns it.
func (t *Tasks) Add(ctx context.Context, text string) (Task, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return Task{}, errors.New("the task is empty")
	}
	task := Task{ID: rand.Text(), Text: text, Created: t.now().UTC()}
	if err := t.tasks.Put(ctx, task); err != nil {
		return Task{}, fmt.Errorf("save task: %w", err)
	}
	return task, nil
}

// Complete marks the task with the given ID as done and returns it.
// Completing a task that's already done returns it unchanged.
func (t *Tasks) Complete(ctx context.Context, id string) (Task, error) {
	task, err := t.get(ctx, id)
	if err != nil || task.Completed != nil {
		return task, err
	}
	now := t.now().UTC()
	task.Completed = &now
	if err := t.tasks.Put(ctx, task); err != nil {
		return Task{}, fmt.Errorf("complete task: %w", err)
	}
	return task, nil
}

// Reopen marks the completed task with the given ID as open again and
// returns it. Reopening an open task returns it unchanged.
func (t *Tasks) Reopen(ctx context.Context, id string) (Task, error) {
	task, err := t.get(ctx, id)
	if err != nil || task.Completed == nil {
		return task, err
	}
	task.Completed = nil
	if err := t.tasks.Put(ctx, task); err != nil {
		return Task{}, fmt.Errorf("reopen task: %w", err)
	}
	return task, nil
}

// Delete permanently removes the completed task with the given ID from the
// database. Open tasks can't be deleted: complete them first. Deleting a task
// that's already gone is not an error.
func (t *Tasks) Delete(ctx context.Context, id string) error {
	task, err := t.get(ctx, id)
	if errors.Is(err, errNoTask) {
		return nil
	}
	if err != nil {
		return err
	}
	if task.Completed == nil {
		return errors.New("only completed tasks can be deleted")
	}
	if err := t.tasks.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete task: %w", err)
	}
	return nil
}

// get returns the task with the given ID, or errNoTask.
func (t *Tasks) get(ctx context.Context, id string) (Task, error) {
	task, err := t.tasks.Get(ctx, id)
	if errors.Is(err, db.ErrNotFound) {
		return Task{}, errNoTask
	}
	return task, err
}
