// Package whipapp holds the parts of the running Whip application that keep
// state: the config store, the database, the services over them and the
// app's own settings file, whip.yml. Package app builds the Wails app on
// top of them.
package whipapp

import (
	"context"
	"errors"

	"github.com/ha-li/Whip/internal/config"
	"github.com/ha-li/Whip/internal/db"
	"github.com/ha-li/Whip/internal/motd"
	"github.com/ha-li/Whip/internal/notebook"
	"github.com/ha-li/Whip/internal/stocks"
	"github.com/ha-li/Whip/internal/tasks"
)

// databaseFile is the app's document database (see package db) in its config
// directory, e.g. ~/Library/Application Support/Whip.
const databaseFile = "data.db"

// SettingsFile is the app's own settings file in its config directory. It
// remembers the window, the open tab and the side menu's width between
// launches; modules keep their settings in their own files.
const SettingsFile = "whip.yml"

// WhipApp owns the parts of Whip that keep state, and creates them.
type WhipApp struct {
	// store is the app's config directory (see package config).
	store *config.Store
	// db is the app's document database; Close closes it.
	db *db.DB

	// The services bound to the frontend that keep the user's data.
	stocksService   *stocks.Notes
	notebookService *notebook.Notebook
	taskService     *tasks.Tasks
	quotes          *motd.Motd

	// appSetting is the app's own settings file, whip.yml.
	appSetting *config.File[Settings]
}

// NewWhipApp creates the app's parts in its config directory, named appName
// (e.g. ~/Library/Application Support/<appName>): it opens the database and
// whip.yml and creates the services. Call Close when the app quits.
func NewWhipApp(appName string) (*WhipApp, error) {
	store, err := config.NewStore(appName)
	if err != nil {
		return nil, err
	}
	return newWhipApp(store)
}

// newWhipApp creates the app's parts in store. If any part can't be created,
// it closes the database again.
func newWhipApp(store *config.Store) (*WhipApp, error) {
	dbPath, err := store.Reserve(databaseFile)
	if err != nil {
		return nil, err
	}
	database, err := db.Open(dbPath)
	if err != nil {
		return nil, err
	}
	w := &WhipApp{store: store, db: database}
	if err := w.init(context.Background()); err != nil {
		return nil, errors.Join(err, database.Close())
	}
	return w, nil
}

// init creates the services over w's database, and opens whip.yml.
func (w *WhipApp) init(ctx context.Context) error {
	var err error
	if w.stocksService, err = stocks.New(ctx, w.db); err != nil {
		return err
	}
	if w.notebookService, err = notebook.New(ctx, w.db); err != nil {
		return err
	}
	if w.taskService, err = tasks.New(ctx, w.db); err != nil {
		return err
	}
	if w.quotes, err = motd.New(); err != nil {
		return err
	}
	if w.appSetting, err = config.Open[Settings](w.store, SettingsFile); err != nil {
		return err
	}
	return nil
}

// Close closes the database. The WhipApp can't be used afterwards.
func (w *WhipApp) Close() error {
	return w.db.Close()
}

// Store returns the app's config directory.
func (w *WhipApp) Store() *config.Store { return w.store }

// DB returns the app's document database.
func (w *WhipApp) DB() *db.DB { return w.db }

// StocksService returns the service that keeps the user's stock notes.
func (w *WhipApp) StocksService() *stocks.Notes { return w.stocksService }

// NotebookService returns the service that keeps the user's notes.
func (w *WhipApp) NotebookService() *notebook.Notebook { return w.notebookService }

// TaskService returns the service that keeps the user's tasks.
func (w *WhipApp) TaskService() *tasks.Tasks { return w.taskService }

// Quotes returns the service that hands out the quote of the day.
func (w *WhipApp) Quotes() *motd.Motd { return w.quotes }

// AppSetting returns the app's own settings file, whip.yml.
func (w *WhipApp) AppSetting() *config.File[Settings] { return w.appSetting }
