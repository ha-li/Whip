import { useState } from 'react'
import { pages, PageId } from './pages'

function App() {
  const [active, setActive] = useState<PageId>('home');
  const page = pages.find((p) => p.id === active) ?? pages[0];
  const Page = page.component;

  return (
    <div className="layout">
      <nav className="sidebar" aria-label="Main menu">
        {/* Empty strip that clears the macOS traffic lights and drags the window. */}
        <div className="titlebar" />
        <ul className="menu">
          {pages.map((p) => (
            <li key={p.id}>
              <button
                className={`menu-item${p.id === active ? ' is-active' : ''}`}
                aria-current={p.id === active ? 'page' : undefined}
                onClick={() => setActive(p.id)}
              >
                <p.icon />
                <span>{p.label}</span>
              </button>
            </li>
          ))}
        </ul>
      </nav>

      <main className="content">
        <div className="titlebar" />
        <div className="page">
          <Page />
        </div>
      </main>
    </div>
  )
}

export default App
