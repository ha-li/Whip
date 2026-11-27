// Package motd picks a quote of the day from a list built into the app
// (quotes.txt). The quote changes at local midnight; within a day it stays
// the same.
//
// To add a quote, append it to quotes.txt: the quote, a line starting with
// "-- " naming who said it, and a "%" line between quotes.
package motd

import (
	_ "embed"
	"errors"
	"fmt"
	"hash/fnv"
	"strings"
	"time"
)

//go:embed quotes.txt
var quotesFile string

// Quote is a quote and who said it.
type Quote struct {
	Text   string `json:"text"`
	Author string `json:"author"`
}

// Motd hands out the quote of the day. It is bound to the frontend.
type Motd struct {
	quotes []Quote
	now    func() time.Time
}

// New returns a Motd over the built-in quotes.
func New() (*Motd, error) {
	return newMotd(quotesFile, time.Now)
}

func newMotd(file string, now func() time.Time) (*Motd, error) {
	quotes, err := parse(file)
	if err != nil {
		return nil, err
	}
	return &Motd{quotes: quotes, now: now}, nil
}

// Today returns today's quote. Every call on the same local date returns the
// same quote; the next day's is picked at random from the list, so it can
// occasionally repeat.
func (m *Motd) Today() Quote {
	// Hashing the date scatters consecutive days across the list.
	h := fnv.New32a()
	_, _ = h.Write([]byte(m.now().Format(time.DateOnly))) // a hash.Hash never fails to write
	return m.quotes[h.Sum32()%uint32(len(m.quotes))]
}

// parse reads quotes in the format of quotes.txt (see the package comment).
func parse(file string) ([]Quote, error) {
	// Entries are separated by lines holding only "%".
	var entries []string
	var entry strings.Builder
	for line := range strings.Lines(strings.ReplaceAll(file, "\r\n", "\n")) {
		if strings.TrimSpace(line) == "%" {
			entries = append(entries, entry.String())
			entry.Reset()
			continue
		}
		entry.WriteString(line)
	}
	entries = append(entries, entry.String())

	var quotes []Quote
	for i, entry := range entries {
		if i == len(entries)-1 && strings.TrimSpace(entry) == "" && i > 0 {
			break // a trailing "%" line
		}
		text, author, ok := strings.Cut(strings.TrimSpace(entry), "\n-- ")
		text, author = strings.TrimSpace(text), strings.TrimSpace(author)
		if !ok || text == "" || author == "" || strings.Contains(author, "\n") {
			return nil, fmt.Errorf("quote %d: want the quote, then a line \"-- author\"", i+1)
		}
		quotes = append(quotes, Quote{Text: text, Author: author})
	}
	if len(quotes) == 0 {
		return nil, errors.New("no quotes")
	}
	return quotes, nil
}
