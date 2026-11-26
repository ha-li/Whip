package stocks_test

import (
	"errors"
	"testing"

	"github.com/ha-li/Whip/internal/stocks"
)

func TestAPIKey(t *testing.T) {
	t.Parallel()

	settings := stocks.NewMemSettings(t.TempDir())
	s := stocks.New(settings)

	has, err := s.HasAPIKey()
	if err != nil || has {
		t.Fatalf("HasAPIKey() before saving = %v, %v; want false, nil", has, err)
	}
	if err := s.SetAPIKey("  "); err == nil {
		t.Error("SetAPIKey(blank) succeeded, want error")
	}
	if err := s.SetAPIKey(" KEY123\n"); err != nil {
		t.Fatalf("SetAPIKey() error = %v", err)
	}
	has, err = s.HasAPIKey()
	if err != nil || !has {
		t.Fatalf("HasAPIKey() after saving = %v, %v; want true, nil", has, err)
	}

	// The key lands in the settings, trimmed.
	c, err := settings.Load()
	if err != nil {
		t.Fatal(err)
	}
	if c.AlphaVantage.APIKey != "KEY123" {
		t.Errorf("config api_key = %q, want %q", c.AlphaVantage.APIKey, "KEY123")
	}
}

func TestChartRejectsBeforeFetching(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		key    bool
		symbol string
		r      stocks.Range
		want   error // nil: any error
	}{
		{name: "no key", symbol: "IBM", r: stocks.Range1M, want: stocks.ErrNoAPIKey},
		{name: "empty symbol", key: true, symbol: " ", r: stocks.Range1M},
		{name: "path in symbol", key: true, symbol: "../IBM", r: stocks.Range1M},
		{name: "unknown range", key: true, symbol: "IBM", r: "2W"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := stocks.New(stocks.NewMemSettings(t.TempDir()))
			if tt.key {
				if err := s.SetAPIKey("KEY"); err != nil {
					t.Fatal(err)
				}
			}
			_, err := s.Chart(t.Context(), tt.symbol, tt.r)
			if err == nil {
				t.Fatal("Chart() succeeded, want error")
			}
			if tt.want != nil && !errors.Is(err, tt.want) {
				t.Errorf("Chart() error = %v, want %v", err, tt.want)
			}
		})
	}
}
