import type { ComponentType } from 'react'
import Home from './Home'

// Page component for each tab ID. The tab list itself comes from Go
// (internal/app/menu.go); add a page here when you add a tab there.
export const pages: Record<string, ComponentType> = {
  home: Home,
};
