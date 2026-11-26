import type { ComponentType } from 'react'
import Home from './Home'
import { HomeIcon } from '../icons'

// The left-hand menu, in display order. To add a page: create a component in
// this folder, add an icon in ../icons.tsx, and append an entry here.
export const pages = [
  { id: 'home', label: 'Home', icon: HomeIcon, component: Home },
] as const satisfies readonly {
  id: string
  label: string
  icon: ComponentType
  component: ComponentType
}[];

export type PageId = (typeof pages)[number]['id'];
