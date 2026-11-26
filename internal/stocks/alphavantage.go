package stocks

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"time"
)

// seriesKind is an Alpha Vantage time-series function. The free key's daily
// data is the last 100 trading days; weekly covers 25+ years.
type seriesKind int

const (
	seriesDaily seriesKind = iota
	seriesWeekly
)

// function is the API's function parameter.
func (k seriesKind) function() string {
	if k == seriesWeekly {
		return "TIME_SERIES_WEEKLY"
	}
	return "TIME_SERIES_DAILY"
}

// key is the response's time-series key.
func (k seriesKind) key() string {
	if k == seriesWeekly {
		return "Weekly Time Series"
	}
	return "Time Series (Daily)"
}

// ttl is how long a cached response stays fresh.
func (k seriesKind) ttl() time.Duration {
	if k == seriesWeekly {
		return 12 * time.Hour
	}
	return time.Hour
}

// client fetches Alpha Vantage time series, caching each response on disk.
type client struct {
	baseURL  string
	http     *http.Client
	cacheDir string
	now      func() time.Time
}

// series returns symbol's closing prices for kind, oldest first, from the
// cache if it is fresh enough and from the API otherwise.
func (c *client) series(ctx context.Context, kind seriesKind, symbol, apiKey string) ([]Point, error) {
	path := filepath.Join(c.cacheDir, kind.function()+"-"+symbol+".json")
	if info, err := os.Stat(path); err == nil && c.now().Sub(info.ModTime()) < kind.ttl() {
		if body, err := os.ReadFile(path); err == nil {
			if points, err := parseSeries(body, kind.key()); err == nil {
				return points, nil
			}
		}
	}

	body, err := c.fetch(ctx, kind.function(), symbol, apiKey)
	if err != nil {
		return nil, err
	}
	points, err := parseSeries(body, kind.key())
	if err != nil {
		return nil, fmt.Errorf("%s: %w", symbol, err)
	}
	// Only well-formed responses are cached; a failed write costs a request next time.
	if err := os.MkdirAll(c.cacheDir, 0o700); err == nil {
		_ = os.WriteFile(path, body, 0o600)
	}
	return points, nil
}

func (c *client) fetch(ctx context.Context, function, symbol, apiKey string) ([]byte, error) {
	// Parameters in the documented order (url.Values.Encode would sort them):
	// Alpha Vantage's demo key only answers URLs that match its examples exactly.
	q := "function=" + url.QueryEscape(function) + "&symbol=" + url.QueryEscape(symbol) + "&apikey=" + url.QueryEscape(apiKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"?"+q, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		// The URL holds the API key; report the failure without it.
		var uerr *url.Error
		if errors.As(err, &uerr) {
			err = uerr.Err
		}
		return nil, fmt.Errorf("reach Alpha Vantage: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected HTTP %d from Alpha Vantage", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
	if err != nil {
		return nil, fmt.Errorf("read Alpha Vantage response: %w", err)
	}
	return body, nil
}

// parseSeries reads a time-series response. Alpha Vantage reports problems
// (bad key, unknown symbol, rate limit) with HTTP 200 and a message under
// "Error Message", "Information" or "Note"; that message is returned as is.
func parseSeries(body []byte, key string) ([]Point, error) {
	var top map[string]json.RawMessage
	if err := json.Unmarshal(body, &top); err != nil {
		return nil, fmt.Errorf("unreadable Alpha Vantage response: %w", err)
	}
	for _, field := range []string{"Error Message", "Information", "Note"} {
		if raw, ok := top[field]; ok {
			var msg string
			if json.Unmarshal(raw, &msg) == nil && msg != "" {
				return nil, fmt.Errorf("from Alpha Vantage: %s", msg)
			}
		}
	}
	raw, ok := top[key]
	if !ok {
		return nil, fmt.Errorf("no %q in the Alpha Vantage response", key)
	}
	var days map[string]map[string]string
	if err := json.Unmarshal(raw, &days); err != nil {
		return nil, fmt.Errorf("unreadable %q: %w", key, err)
	}
	points := make([]Point, 0, len(days))
	for date, fields := range days {
		price, err := strconv.ParseFloat(fields["4. close"], 64)
		if err != nil {
			return nil, fmt.Errorf("close price for %s: %w", date, err)
		}
		points = append(points, Point{Date: date, Close: price})
	}
	if len(points) == 0 {
		return nil, errors.New("no price data")
	}
	// "YYYY-MM-DD" sorts chronologically as text.
	sort.Slice(points, func(i, j int) bool { return points[i].Date < points[j].Date })
	return points, nil
}

// latestQuote returns the last close in points (oldest first) and its change
// from the close before it.
func latestQuote(points []Point) (Quote, error) {
	if len(points) < 2 {
		return Quote{}, errors.New("not enough price data for a quote")
	}
	last, prev := points[len(points)-1], points[len(points)-2]
	q := Quote{Price: last.Close, Change: last.Close - prev.Close, AsOf: last.Date}
	if prev.Close != 0 {
		q.ChangePercent = q.Change / prev.Close * 100
	}
	return q, nil
}

// since returns the points (oldest first) within months of the latest one.
func since(points []Point, months int) []Point {
	if len(points) == 0 {
		return points
	}
	latest, err := time.Parse(time.DateOnly, points[len(points)-1].Date)
	if err != nil {
		return points
	}
	cutoff := latest.AddDate(0, -months, 0).Format(time.DateOnly)
	i := sort.Search(len(points), func(i int) bool { return points[i].Date >= cutoff })
	return points[i:]
}
