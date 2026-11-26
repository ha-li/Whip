import { useEffect, useState, type FormEvent } from 'react'
import { Browser } from '@wailsio/runtime'
import { Stocks as StocksAPI, Range, type Chart } from '../../bindings/github.com/ha-li/Whip/internal/stocks'
import PriceChart from '../stocks/PriceChart'
import { errorText, formatChange, formatChangePercent, formatLongDate, formatPrice } from '../stocks/format'
import './Stocks.css'

const RANGES = [Range.Range1M, Range.Range6M, Range.Range1Y, Range.Range5Y];
const KEY_URL = 'https://www.alphavantage.co/support/#api-key';

function Stocks() {
  // null while checking whether a key has been saved.
  const [hasKey, setHasKey] = useState<boolean | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    StocksAPI.HasAPIKey()
      .then(setHasKey)
      .catch((err) => setError(errorText(err)));
  }, []);

  return (
    <section className="stocks">
      <h1 className="page-title">Stocks</h1>
      {error && <p className="stocks-error" role="alert">{error}</p>}
      {hasKey === false && <KeyForm onSaved={() => setHasKey(true)} />}
      {hasKey && <Lookup onChangeKey={() => setHasKey(false)} />}
    </section>
  )
}

// KeyForm asks for the Alpha Vantage API key on first use.
function KeyForm({ onSaved }: { onSaved: () => void }) {
  const [key, setKey] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);

  const submit = (e: FormEvent) => {
    e.preventDefault();
    setSaving(true);
    setError(null);
    StocksAPI.SetAPIKey(key)
      .then(onSaved)
      .catch((err) => setError(errorText(err)))
      .finally(() => setSaving(false));
  };

  return (
    <form className="stocks-key" onSubmit={submit}>
      <p className="page-subtitle">
        Stock data comes from Alpha Vantage. Paste your API key to start; a free key allows 25
        requests a day, and Whip caches results to make them last.{' '}
        <button type="button" className="link-button" onClick={() => Browser.OpenURL(KEY_URL)}>
          Get a free key
        </button>
      </p>
      <div className="stocks-row">
        <input
          type="password"
          className="text-input"
          aria-label="Alpha Vantage API key"
          placeholder="API key"
          autoComplete="off"
          value={key}
          onChange={(e) => setKey(e.target.value)}
        />
        <button type="submit" className="button" disabled={saving || key.trim() === ''}>
          Save key
        </button>
      </div>
      {error && <p className="stocks-error" role="alert">{error}</p>}
    </form>
  )
}

// Lookup searches a ticker and shows its quote and price chart.
function Lookup({ onChangeKey }: { onChangeKey: () => void }) {
  const [input, setInput] = useState('');
  const [symbol, setSymbol] = useState('');
  // Bumped on every submit, so looking up the same symbol again retries.
  const [attempt, setAttempt] = useState(0);
  const [range, setRange] = useState<Range>(Range.Range1M);
  const [chart, setChart] = useState<Chart | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!symbol) return;
    let cancelled = false;
    setLoading(true);
    setError(null);
    const call = StocksAPI.Chart(symbol, range);
    call
      .then((c) => {
        if (!cancelled) setChart(c);
      })
      .catch((err) => {
        if (!cancelled) setError(errorText(err));
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    // A newer lookup replaces this one; cancelling also stops the Go request.
    return () => {
      cancelled = true;
      call.cancel();
    };
  }, [symbol, range, attempt]);

  const submit = (e: FormEvent) => {
    e.preventDefault();
    setSymbol(input.trim().toUpperCase());
    setAttempt((n) => n + 1);
  };

  const points = chart?.points ?? [];
  const up = (chart?.quote.change ?? 0) >= 0;

  return (
    <div className="stocks-lookup">
      <form className="stocks-row" onSubmit={submit} role="search">
        <input
          className="text-input stocks-symbol"
          aria-label="Ticker symbol"
          placeholder="Ticker, e.g. AAPL"
          autoCapitalize="characters"
          spellCheck={false}
          value={input}
          onChange={(e) => setInput(e.target.value)}
        />
        <button type="submit" className="button" disabled={input.trim() === ''}>
          Look up
        </button>
        <button type="button" className="link-button stocks-change-key" onClick={onChangeKey}>
          Change API key
        </button>
      </form>

      {error && <p className="stocks-error" role="alert">{error}</p>}

      {chart && (
        <div className={`stocks-result${loading ? ' is-loading' : ''}`} aria-busy={loading}>
          <div className="stocks-quote">
            <h2 className="stocks-quote-symbol">{chart.symbol}</h2>
            <div className="stocks-quote-price">{formatPrice(chart.quote.price)}</div>
            <div className="stocks-quote-change">
              <span aria-hidden="true">{up ? '▲' : '▼'}</span>{' '}
              {formatChange(chart.quote.change)} ({formatChangePercent(chart.quote.changePercent)})
              <span className="visually-hidden">{up ? ' up' : ' down'} from the previous close</span>
            </div>
            <p className="stocks-quote-asof">Daily close on {formatLongDate(chart.quote.asOf)} · Alpha Vantage</p>
          </div>

          <div className="segmented" role="group" aria-label="Chart range">
            {RANGES.map((r) => (
              <button key={r} type="button" className="segmented-item" aria-pressed={r === range} onClick={() => setRange(r)}>
                {r}
              </button>
            ))}
          </div>

          {points.length > 0 ? (
            <PriceChart
              points={points}
              label={`${chart.symbol} closing price${range === Range.Range1M ? '' : ', weekly'}`}
              longRange={range === Range.Range1Y || range === Range.Range5Y}
            />
          ) : (
            <p className="page-subtitle">No prices in this range.</p>
          )}
        </div>
      )}
      {!chart && loading && <p className="page-subtitle">Loading {symbol}…</p>}
    </div>
  )
}

export default Stocks
