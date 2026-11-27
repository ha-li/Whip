package motd

import (
	"testing"
	"time"
)

func TestParse(t *testing.T) {
	t.Parallel()

	got, err := parse("First quote,\nover two lines.\n-- Someone\n%\nSecond.\r\n-- Someone Else\n")
	if err != nil {
		t.Fatalf("parse() error = %v", err)
	}
	want := []Quote{
		{Text: "First quote,\nover two lines.", Author: "Someone"},
		{Text: "Second.", Author: "Someone Else"},
	}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("parse() = %+v, want %+v", got, want)
	}
}

func TestParseRejects(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name, file string
	}{
		{"empty", ""},
		{"no author", "A quote.\n"},
		{"empty author", "A quote.\n-- \n"},
		{"no quote", "-- Someone\n"},
		{"missing separator", "One.\n-- A\nTwo.\n-- B\n"},
		{"empty entry", "One.\n-- A\n%\n%\nTwo.\n-- B\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got, err := parse(tt.file); err == nil {
				t.Errorf("parse(%q) = %+v, want error", tt.file, got)
			}
		})
	}
}

func TestTodayChangesDaily(t *testing.T) {
	t.Parallel()

	var day time.Time
	m, err := newMotd(quotesFile, func() time.Time { return day })
	if err != nil {
		t.Fatal(err)
	}

	// The same all day, from just after midnight to just before the next.
	start := time.Date(2026, 11, 26, 0, 0, 1, 0, time.Local)
	day = start
	morning := m.Today()
	day = start.Add(24*time.Hour - 2*time.Second)
	if got := m.Today(); got != morning {
		t.Errorf("Today() late in the day = %q, want the morning's %q", got.Text, morning.Text)
	}

	// Over a month, it picks many different quotes.
	seen := map[Quote]bool{}
	for i := range 30 {
		day = start.AddDate(0, 0, i)
		seen[m.Today()] = true
	}
	if len(seen) < 10 {
		t.Errorf("30 days gave %d different quotes, want at least 10", len(seen))
	}
}
