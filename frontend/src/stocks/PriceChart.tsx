import { useEffect, useMemo, useRef, useState, type KeyboardEvent, type PointerEvent } from 'react'
import type { Point } from '../../bindings/github.com/ha-li/Whip/internal/stocks'
import { formatAxisDate, formatLongDate, formatPrice } from './format'
import './PriceChart.css'

type Props = {
  points: Point[]
  // label names the series for screen readers, e.g. "IBM closing price".
  label: string
  // longRange uses month-and-year axis labels instead of month-and-day.
  longRange: boolean
}

const HEIGHT = 280;
const MARGIN = { top: 12, right: 72, bottom: 28, left: 56 };

// niceTicks returns about `count` evenly spaced round numbers covering [min, max].
function niceTicks(min: number, max: number, count = 4): number[] {
  if (min === max) {
    const pad = Math.abs(min) * 0.05 || 1;
    min -= pad;
    max += pad;
  }
  const raw = (max - min) / count;
  const magnitude = 10 ** Math.floor(Math.log10(raw));
  const step = [1, 2, 2.5, 5, 10].map((m) => m * magnitude).find((s) => s >= raw) ?? raw;
  const ticks: number[] = [];
  for (let v = Math.floor(min / step) * step; v <= Math.ceil(max / step) * step + step / 2; v += step) {
    ticks.push(Number(v.toFixed(10)));
  }
  return ticks;
}

// PriceChart draws closing prices as a single line, with a hover/keyboard
// crosshair that snaps to the nearest trading day and a data table below.
export default function PriceChart({ points, label, longRange }: Props) {
  const boxRef = useRef<HTMLDivElement>(null);
  const [width, setWidth] = useState(0);
  const [active, setActive] = useState<number | null>(null);

  useEffect(() => {
    const box = boxRef.current;
    if (!box) return;
    const observer = new ResizeObserver(([entry]) => setWidth(entry.contentRect.width));
    observer.observe(box);
    return () => observer.disconnect();
  }, []);

  // A new series (symbol or range) clears the crosshair.
  useEffect(() => setActive(null), [points]);

  const geometry = useMemo(() => {
    if (points.length === 0 || width === 0) return null;
    const closes = points.map((p) => p.close);
    const ticks = niceTicks(Math.min(...closes), Math.max(...closes));
    const yMin = ticks[0];
    const yMax = ticks[ticks.length - 1];
    const plotW = Math.max(width - MARGIN.left - MARGIN.right, 1);
    const plotH = HEIGHT - MARGIN.top - MARGIN.bottom;
    const x = (i: number) => MARGIN.left + (points.length === 1 ? plotW / 2 : (i / (points.length - 1)) * plotW);
    const y = (v: number) => MARGIN.top + (1 - (v - yMin) / (yMax - yMin)) * plotH;
    const path = points.map((p, i) => `${i === 0 ? 'M' : 'L'}${x(i).toFixed(1)},${y(p.close).toFixed(1)}`).join('');
    // Axis dates at the start, middle and end, spaced by trading day.
    const dateIdx = [...new Set([0, Math.floor((points.length - 1) / 2), points.length - 1])];
    return { ticks, x, y, path, plotW, dateIdx };
  }, [points, width]);

  const indexAt = (clientX: number) => {
    const box = boxRef.current;
    if (!box || !geometry || points.length < 2) return 0;
    const rel = clientX - box.getBoundingClientRect().left - MARGIN.left;
    const i = Math.round((rel / geometry.plotW) * (points.length - 1));
    return Math.min(Math.max(i, 0), points.length - 1);
  };

  const onPointerMove = (e: PointerEvent) => setActive(indexAt(e.clientX));

  const onKeyDown = (e: KeyboardEvent) => {
    const last = points.length - 1;
    const step = { ArrowLeft: -1, ArrowRight: 1 }[e.key];
    if (step !== undefined) {
      e.preventDefault();
      setActive((i) => Math.min(Math.max((i ?? last) + step, 0), last));
    } else if (e.key === 'Home' || e.key === 'End') {
      e.preventDefault();
      setActive(e.key === 'Home' ? 0 : last);
    } else if (e.key === 'Escape') {
      setActive(null);
    }
  };

  const last = points[points.length - 1];
  const hovered = active !== null ? points[active] : undefined;

  return (
    <figure className="price-chart">
      <div
        ref={boxRef}
        className="price-chart-plot"
        tabIndex={0}
        role="img"
        aria-label={`${label}, ${points.length} points from ${formatLongDate(points[0].date)} to ${formatLongDate(last.date)}. Use the left and right arrow keys to read values.`}
        onPointerMove={onPointerMove}
        onPointerLeave={() => setActive(null)}
        onKeyDown={onKeyDown}
        onBlur={() => setActive(null)}
      >
        {geometry && (
          <svg width={width} height={HEIGHT} aria-hidden="true">
            {geometry.ticks.map((t) => (
              <g key={t}>
                <line className="price-chart-grid" x1={MARGIN.left} x2={MARGIN.left + geometry.plotW} y1={geometry.y(t)} y2={geometry.y(t)} />
                <text className="price-chart-axis" x={MARGIN.left - 8} y={geometry.y(t)} textAnchor="end" dominantBaseline="middle">
                  {formatPrice(t)}
                </text>
              </g>
            ))}
            {geometry.dateIdx.map((i, n) => (
              <text
                key={i}
                className="price-chart-axis"
                x={geometry.x(i)}
                y={HEIGHT - 8}
                textAnchor={n === 0 ? 'start' : n === geometry.dateIdx.length - 1 ? 'end' : 'middle'}
              >
                {formatAxisDate(points[i].date, longRange)}
              </text>
            ))}
            <path className="price-chart-line" d={geometry.path} />
            {/* Direct label: the latest close at the end of the line. */}
            <text className="price-chart-end" x={geometry.x(points.length - 1) + 8} y={geometry.y(last.close)} dominantBaseline="middle">
              {formatPrice(last.close)}
            </text>
            {hovered && active !== null && (
              <g>
                <line className="price-chart-crosshair" x1={geometry.x(active)} x2={geometry.x(active)} y1={MARGIN.top} y2={HEIGHT - MARGIN.bottom} />
                <circle className="price-chart-dot" cx={geometry.x(active)} cy={geometry.y(hovered.close)} r={4} />
              </g>
            )}
          </svg>
        )}
        {geometry && hovered && active !== null && (
          <div
            className="price-chart-tooltip"
            style={{
              left: Math.min(Math.max(geometry.x(active), 70), width - 70),
              top: MARGIN.top,
            }}
          >
            <div className="price-chart-tooltip-date">{formatLongDate(hovered.date)}</div>
            <div className="price-chart-tooltip-value">
              <span className="price-chart-key" aria-hidden="true" />
              {formatPrice(hovered.close)}
            </div>
          </div>
        )}
      </div>

      <details className="price-chart-table">
        <summary>Show data table</summary>
        <table>
          <thead>
            <tr>
              <th scope="col">Date</th>
              <th scope="col">Close</th>
            </tr>
          </thead>
          <tbody>
            {[...points].reverse().map((p) => (
              <tr key={p.date}>
                <td>{formatLongDate(p.date)}</td>
                <td>{formatPrice(p.close)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </details>
    </figure>
  )
}
