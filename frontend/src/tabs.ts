import type { ComponentType } from 'react'
import { HomeIcon, NotesIcon } from './icons'
import Home from './pages/Home'
import Notes from './pages/Notes'

// Tab is one entry in the left-hand menu.
export type Tab = {
  // id identifies the tab; it must be unique.
  id: string
  // label is the text shown in the menu.
  label: string
  // Icon is the menu icon (see icons.tsx).
  Icon: ComponentType
  // Page is the component shown while the tab is selected.
  Page: ComponentType
}

// The left-hand menu, in display order. The first tab is selected on launch.
// To add a tab, add its page under pages/, its icon in icons.tsx, and an entry here.
export const tabs: readonly [Tab, ...Tab[]] = [
  { id: 'home', label: 'Home', Icon: HomeIcon, Page: Home },
  { id: 'notes', label: 'Notes', Icon: NotesIcon, Page: Notes },
]
