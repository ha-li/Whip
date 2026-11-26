package stocks

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ha-li/Whip/internal/config"
)

// MemSettings is an in-memory Settings for tests (exported so the stocks_test
// package can use it too). Its cache directory is a real directory, cacheDir.
type MemSettings struct {
	mu       sync.Mutex
	cfg      config.Config
	cacheDir string
}

// NewMemSettings returns empty settings whose cache directory is cacheDir.
func NewMemSettings(cacheDir string) *MemSettings {
	return &MemSettings{cacheDir: cacheDir}
}

func (m *MemSettings) Load() (config.Config, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.cfg, nil
}

func (m *MemSettings) Update(change func(*config.Config)) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	change(&m.cfg)
	return nil
}

func (m *MemSettings) CacheDir() string { return m.cacheDir }

// seriesJSON builds an Alpha Vantage time-series response from date -> close.
func seriesJSON(t *testing.T, key string, closes map[string]float64) []byte {
	t.Helper()
	days := make(map[string]map[string]string, len(closes))
	for date, c := range closes {
		days[date] = map[string]string{"1. open": "1.0", "4. close": fmt.Sprintf("%.4f", c)}
	}
	b, err := json.Marshal(map[string]any{"Meta Data": map[string]string{}, key: days})
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// fakeAPI serves daily and weekly series and counts the requests it gets.
func fakeAPI(t *testing.T) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	daily := seriesJSON(t, "Time Series (Daily)", map[string]float64{
		"2026-09-30": 90, "2026-10-24": 100, "2026-10-27": 110, "2026-11-24": 120, "2026-11-25": 126,
	})
	weekly := seriesJSON(t, "Weekly Time Series", map[string]float64{
		"2020-01-03": 50, "2025-11-21": 80, "2026-05-29": 95, "2026-11-25": 126,
	})
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		if r.URL.Query().Get("apikey") != "KEY" {
			_, _ = w.Write([]byte(`{"Error Message": "the parameter apikey is invalid or missing."}`))
			return
		}
		switch r.URL.Query().Get("function") {
		case "TIME_SERIES_DAILY":
			_, _ = w.Write(daily)
		case "TIME_SERIES_WEEKLY":
			_, _ = w.Write(weekly)
		default:
			http.Error(w, "unexpected function", http.StatusBadRequest)
		}
	}))
	t.Cleanup(srv.Close)
	return srv, &hits
}

func newTestStocks(t *testing.T, baseURL string, now func() time.Time) *Stocks {
	t.Helper()
	dir := t.TempDir()
	s := newStocks(baseURL, NewMemSettings(dir), http.DefaultClient, now)
	if err := s.SetAPIKey("KEY"); err != nil {
		t.Fatal(err)
	}
	return s
}

func TestChart(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		r     Range
		dates []string
	}{
		{"1M uses daily closes", Range1M, []string{"2026-10-27", "2026-11-24", "2026-11-25"}},
		{"6M uses weekly closes", Range6M, []string{"2026-05-29", "2026-11-25"}},
		{"5Y uses weekly closes", Range5Y, []string{"2025-11-21", "2026-05-29", "2026-11-25"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			srv, _ := fakeAPI(t)
			s := newTestStocks(t, srv.URL, time.Now)

			got, err := s.Chart(t.Context(), " ibm ", tt.r)
			if err != nil {
				t.Fatalf("Chart() error = %v", err)
			}
			if got.Symbol != "IBM" || got.Range != tt.r {
				t.Errorf("Chart() symbol, range = %q, %q; want IBM, %q", got.Symbol, got.Range, tt.r)
			}
			want := Quote{Price: 126, Change: 6, ChangePercent: 5, AsOf: "2026-11-25"}
			if got.Quote != want {
				t.Errorf("Quote = %+v, want %+v", got.Quote, want)
			}
			var dates []string
			for _, p := range got.Points {
				dates = append(dates, p.Date)
			}
			if strings.Join(dates, ",") != strings.Join(tt.dates, ",") {
				t.Errorf("Points dates = %v, want %v", dates, tt.dates)
			}
		})
	}
}

func TestChartCaches(t *testing.T) {
	t.Parallel()

	srv, hits := fakeAPI(t)
	now := time.Now()
	clock := func() time.Time { return now }
	s := newTestStocks(t, srv.URL, clock)

	for range 2 {
		if _, err := s.Chart(t.Context(), "IBM", Range1M); err != nil {
			t.Fatal(err)
		}
	}
	if n := hits.Load(); n != 1 {
		t.Errorf("requests after two lookups = %d, want 1 (second from cache)", n)
	}

	now = now.Add(2 * time.Hour) // past the daily TTL
	if _, err := s.Chart(t.Context(), "IBM", Range1M); err != nil {
		t.Fatal(err)
	}
	if n := hits.Load(); n != 2 {
		t.Errorf("requests after the cache expired = %d, want 2", n)
	}
}

func TestChartReportsProviderMessage(t *testing.T) {
	t.Parallel()

	tests := []struct{ name, body string }{
		{"error message", `{"Error Message": "Invalid API call."}`},
		{"information", `{"Information": "Our standard API rate limit is 25 requests per day."}`},
		{"note", `{"Note": "Thank you for using Alpha Vantage!"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(tt.body))
			}))
			t.Cleanup(srv.Close)
			s := newTestStocks(t, srv.URL, time.Now)

			_, err := s.Chart(t.Context(), "IBM", Range1M)
			var msg map[string]string
			if jerr := json.Unmarshal([]byte(tt.body), &msg); jerr != nil {
				t.Fatal(jerr)
			}
			for _, m := range msg {
				if err == nil || !strings.Contains(err.Error(), m) {
					t.Errorf("Chart() error = %v, want it to contain %q", err, m)
				}
			}
		})
	}
}

func TestChartErrorHidesAPIKey(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.NotFoundHandler())
	url := srv.URL
	srv.Close() // connection refused

	dir := t.TempDir()
	s := newStocks(url, NewMemSettings(dir), http.DefaultClient, time.Now)
	if err := s.SetAPIKey("SECRET-KEY"); err != nil {
		t.Fatal(err)
	}
	_, err := s.Chart(t.Context(), "IBM", Range1M)
	if err == nil {
		t.Fatal("Chart() succeeded against a closed server")
	}
	if strings.Contains(err.Error(), "SECRET-KEY") {
		t.Errorf("error leaks the API key: %v", err)
	}
}

func TestParseSeriesRejectsBadData(t *testing.T) {
	t.Parallel()

	tests := []struct{ name, body string }{
		{"not json", `<html>`},
		{"missing series", `{"Meta Data": {}}`},
		{"empty series", `{"Time Series (Daily)": {}}`},
		{"bad close", `{"Time Series (Daily)": {"2026-11-25": {"4. close": "n/a"}}}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if points, err := parseSeries([]byte(tt.body), "Time Series (Daily)"); err == nil {
				t.Errorf("parseSeries() = %v, want error", points)
			}
		})
	}
}
