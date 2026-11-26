// Formatting for the Stocks page. Alpha Vantage doesn't say which currency a
// price is in, so prices are plain numbers with two decimals.

const price = new Intl.NumberFormat(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 });
const signedPrice = new Intl.NumberFormat(undefined, {
  minimumFractionDigits: 2,
  maximumFractionDigits: 2,
  signDisplay: 'exceptZero',
});
const signedPercent = new Intl.NumberFormat(undefined, {
  style: 'percent',
  minimumFractionDigits: 2,
  maximumFractionDigits: 2,
  signDisplay: 'exceptZero',
});

export const formatPrice = (n: number) => price.format(n);
export const formatChange = (n: number) => signedPrice.format(n);
// percent is in percent units (5 means 5%).
export const formatChangePercent = (percent: number) => signedPercent.format(percent / 100);

// parseDate reads "YYYY-MM-DD" as a local calendar date (no time-zone shift).
export function parseDate(s: string): Date {
  const [y, m, d] = s.split('-').map(Number);
  return new Date(y, m - 1, d);
}

const longDate = new Intl.DateTimeFormat(undefined, { year: 'numeric', month: 'short', day: 'numeric' });
const monthDay = new Intl.DateTimeFormat(undefined, { month: 'short', day: 'numeric' });
const monthYear = new Intl.DateTimeFormat(undefined, { month: 'short', year: 'numeric' });

export const formatLongDate = (s: string) => longDate.format(parseDate(s));
// formatAxisDate keeps axis labels short: day precision for short ranges, month for long ones.
export const formatAxisDate = (s: string, long: boolean) => (long ? monthYear : monthDay).format(parseDate(s));

// errorText turns a rejected service call into a message for the page.
export function errorText(err: unknown): string {
  return err instanceof Error ? err.message : String(err);
}
