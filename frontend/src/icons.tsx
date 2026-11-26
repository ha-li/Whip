import './icons.css'

// Menu icons: 24×24 line icons that inherit the current text colour.

const iconProps = {
  viewBox: '0 0 24 24',
  fill: 'none',
  stroke: 'currentColor',
  strokeWidth: 1.8,
  strokeLinecap: 'round' as const,
  strokeLinejoin: 'round' as const,
  'aria-hidden': true,
  className: 'icon',
};

export function StocksIcon() {
  return (
    <svg {...iconProps}>
      <path d="M3 3v17a1 1 0 0 0 1 1h17" />
      <path d="m7 15 4-4 3 3 6-7" />
    </svg>
  );
}

export function HomeIcon() {
  return (
    <svg {...iconProps}>
      <path d="M3 10.5 12 3l9 7.5" />
      <path d="M5 9v11a1 1 0 0 0 1 1h4v-6h4v6h4a1 1 0 0 0 1-1V9" />
    </svg>
  );
}
