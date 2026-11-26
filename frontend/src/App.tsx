import { useEffect, useState } from 'react'
import { Menu, type Tab } from '../bindings/github.com/ha-li/Whip/internal/app'
import { About } from '../bindings/github.com/ha-li/Whip/internal/app/version'
import { pages } from './pages'
import { iconFor } from './icons'

function App() {
  const [tabs, setTabs] = useState<Tab[]>([]);
  const [active, setActive] = useState<string | null>(null);
  const [version, setVersion] = useState<string>('');

  // The menu is defined in Go (internal/app/menu.go).
  useEffect(() => {
    Menu.Tabs()
      .then((list) => {
        const t = list ?? [];
        setTabs(t);
        setActive((current) => current ?? t[0]?.id ?? null);
      })
      .catch((err) => console.error('Failed to load menu', err));
    About.Version()
      .then(setVersion)
      .catch((err) => console.error('Failed to load version', err));
  }, []);

  const Page = active ? pages[active] : undefined;

  return (
    <div className="layout">
      <nav className="sidebar" aria-label="Main menu">
        {/* Empty strip that clears the macOS traffic lights and drags the window. */}
        <div className="titlebar" />
        <ul className="menu">
          {tabs.map((tab) => {
            const Icon = iconFor(tab.icon);
            const isActive = tab.id === active;
            return (
              <li key={tab.id}>
                <button
                  className={`menu-item${isActive ? ' is-active' : ''}`}
                  aria-current={isActive ? 'page' : undefined}
                  onClick={() => setActive(tab.id)}
                >
                  <Icon />
                  <span>{tab.label}</span>
                </button>
              </li>
            );
          })}
        </ul>
        {version && <div className="sidebar-version">v{version}</div>}
      </nav>

      <main className="content">
        <div className="titlebar" />
        <div className="page">
          {Page ? <Page /> : active && <p className="page-missing">No page for “{active}” yet.</p>}
        </div>
      </main>
    </div>
  )
}

export default App
