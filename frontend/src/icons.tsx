import type { ComponentType } from 'react'

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

function HomeIcon() {
  return (
    <svg {...iconProps}>
      <path d="M3 10.5 12 3l9 7.5" />
      <path d="M5 9v11a1 1 0 0 0 1 1h4v-6h4v6h4a1 1 0 0 0 1-1V9" />
    </svg>
  );
}

// Fallback for an icon name Go sends that isn't registered below.
function DotIcon() {
  return (
    <svg {...iconProps}>
      <circle cx="12" cy="12" r="3" />
    </svg>
  );
}

// Icon for each name used in internal/app/menu.go.
const icons: Record<string, ComponentType> = {
  home: HomeIcon,
};

export function iconFor(name: string): ComponentType {
  return icons[name] ?? DotIcon;
}
