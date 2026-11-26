// Package stocks looks up stock quotes and price history from Alpha Vantage
// (https://www.alphavantage.co). Its API key is kept in Whip's config file
// (see package config). Responses are cached on disk, in Whip's cache
// directory, because the free key allows only 25 requests a day.
package stocks

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/ha-li/Whip/internal/config"
)

// Range is a span of price history to chart.
type Range string

// The ranges the chart offers.
const (
	Range1M Range = "1M"
	Range6M Range = "6M"
	Range1Y Range = "1Y"
	Range5Y Range = "5Y"
)

// Point is one closing price.
type Point struct {
	// Date is the trading day, "YYYY-MM-DD" (US/Eastern). For weekly data it is
	// the last trading day of the week.
	Date  string  `json:"date"`
	Close float64 `json:"close"`
}

// Quote is the latest daily close and its change from the day before.
type Quote struct {
	Price         float64 `json:"price"`
	Change        float64 `json:"change"`
	ChangePercent float64 `json:"changePercent"`
	// AsOf is the trading day of Price, "YYYY-MM-DD".
	AsOf string `json:"asOf"`
}

// Chart is a symbol's quote and its price history over a range.
type Chart struct {
	Symbol string  `json:"symbol"`
	Quote  Quote   `json:"quote"`
	Range  Range   `json:"range"`
	Points []Point `json:"points"`
}

// ErrNoAPIKey is returned by Chart until an API key has been saved.
var ErrNoAPIKey = errors.New("no Alpha Vantage API key has been saved")

// symbolPattern accepts ticker symbols such as "IBM", "BRK.B" or "TSCO.LON".
// It also keeps symbols safe to use in cache file names.
var symbolPattern = regexp.MustCompile(`^[A-Z0-9][A-Z0-9.\-]{0,14}$`)

// Stocks is bound to the frontend.
type Stocks struct {
	settings Settings
	client   *client
}

// Settings is what Stocks needs from Whip's settings: the saved config (for
// the API key) and the cache directory. *config.Store implements it.
type Settings interface {
	Load() (config.Config, error)
	Update(change func(*config.Config)) error
	CacheDir() string
}

// New returns a Stocks that reads and saves its API key through settings and
// caches responses under settings.CacheDir().
func New(settings Settings) *Stocks {
	return newStocks(
		"https://www.alphavantage.co/query",
		settings,
		&http.Client{Timeout: 15 * time.Second},
		time.Now,
	)
}

func newStocks(baseURL string, settings Settings, httpClient *http.Client, now func() time.Time) *Stocks {
	return &Stocks{
		settings: settings,
		client: &client{
			baseURL:  baseURL,
			http:     httpClient,
			cacheDir: filepath.Join(settings.CacheDir(), "alphavantage"),
			now:      now,
		},
	}
}

// HasAPIKey reports whether an API key has been saved.
func (s *Stocks) HasAPIKey() (bool, error) {
	key, err := s.apiKey()
	return key != "", err
}

// SetAPIKey saves the Alpha Vantage API key in the config file.
func (s *Stocks) SetAPIKey(key string) error {
	key = strings.TrimSpace(key)
	if key == "" {
		return errors.New("the API key is empty")
	}
	if err := s.settings.Update(func(c *config.Config) { c.AlphaVantage.APIKey = key }); err != nil {
		return fmt.Errorf("save API key: %w", err)
	}
	return nil
}

// Chart returns symbol's latest quote and its closing prices over r, oldest
// first. 1M uses daily closes; longer ranges use weekly closes, because the
// free key only returns the last 100 trading days of daily data.
func (s *Stocks) Chart(ctx context.Context, symbol string, r Range) (Chart, error) {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	if !symbolPattern.MatchString(symbol) {
		return Chart{}, fmt.Errorf("%q is not a ticker symbol", symbol)
	}
	span, weekly, err := r.span()
	if err != nil {
		return Chart{}, err
	}
	key, err := s.apiKey()
	if err != nil {
		return Chart{}, err
	}
	if key == "" {
		return Chart{}, ErrNoAPIKey
	}

	daily, err := s.client.series(ctx, seriesDaily, symbol, key)
	if err != nil {
		return Chart{}, err
	}
	quote, err := latestQuote(daily)
	if err != nil {
		return Chart{}, fmt.Errorf("%s: %w", symbol, err)
	}

	history := daily
	if weekly {
		if history, err = s.client.series(ctx, seriesWeekly, symbol, key); err != nil {
			return Chart{}, err
		}
	}
	return Chart{Symbol: symbol, Quote: quote, Range: r, Points: since(history, span)}, nil
}

// apiKey returns the saved key, or "" if none has been saved.
func (s *Stocks) apiKey() (string, error) {
	c, err := s.settings.Load()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(c.AlphaVantage.APIKey), nil
}

// span returns how many months back r reaches and whether it uses weekly data.
func (r Range) span() (months int, weekly bool, err error) {
	switch r {
	case Range1M:
		return 1, false, nil
	case Range6M:
		return 6, true, nil
	case Range1Y:
		return 12, true, nil
	case Range5Y:
		return 60, true, nil
	default:
		return 0, false, fmt.Errorf("unknown range %q", r)
	}
}
