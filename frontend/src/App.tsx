import { useEffect, useState } from 'react'
import { About } from '../bindings/github.com/ha-li/Whip/internal/app/version'
import { tabs } from './tabs'

function App() {
  const [activeId, setActiveId] = useState(tabs[0].id);
  const [version, setVersion] = useState<string>('');

  useEffect(() => {
    About.Version()
      .then(setVersion)
      .catch((err) => console.error('Failed to load version', err));
  }, []);

  const { Page } = tabs.find((tab) => tab.id === activeId) ?? tabs[0];

  return (
    <div className="layout">
      <nav className="sidebar" aria-label="Main menu">
        {/* Empty strip that clears the macOS traffic lights and drags the window. */}
        <div className="titlebar" />
        <ul className="menu">
          {tabs.map(({ id, label, Icon }) => {
            const isActive = id === activeId;
            return (
              <li key={id}>
                <button
                  className={`menu-item${isActive ? ' is-active' : ''}`}
                  aria-current={isActive ? 'page' : undefined}
                  onClick={() => setActiveId(id)}
                >
                  <Icon />
                  <span>{label}</span>
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
          <Page />
        </div>
      </main>
    </div>
  )
}

export default App
