package motd_test

import (
	"testing"

	"github.com/ha-li/Whip/internal/motd"
)

// TestBuiltInQuotes checks that quotes.txt parses: a broken entry would stop
// the app from starting.
func TestBuiltInQuotes(t *testing.T) {
	t.Parallel()

	m, err := motd.New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if q := m.Today(); q.Text == "" || q.Author == "" {
		t.Errorf("Today() = %+v, want a quote and an author", q)
	}
}
