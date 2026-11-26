import { useEffect, useState } from 'react'
import { UI } from '../bindings/github.com/ha-li/Whip/internal/app'
import { About } from '../bindings/github.com/ha-li/Whip/internal/app/version'
import { tabs } from './tabs'
import './App.css'

function App() {
  // null until the last open tab has been read, so the first tab doesn't flash.
  const [activeId, setActiveId] = useState<string | null>(null);
  const [version, setVersion] = useState<string>('');

  // Reopen the tab that was open last time; an unknown id (e.g. a removed
  // tab) falls back to the first tab below.
  useEffect(() => {
    UI.LastTab()
      .then((id) => setActiveId(id))
      .catch((err) => {
        console.error('Failed to read the last open tab', err);
        setActiveId(tabs[0].id);
      });
  }, []);

  const openTab = (id: string) => {
    setActiveId(id);
    UI.SetLastTab(id).catch((err) => console.error('Failed to save the open tab', err));
  };

  useEffect(() => {
    About.Version()
      .then(setVersion)
      .catch((err) => console.error('Failed to load version', err));
  }, []);

  const active = activeId === null ? null : (tabs.find((tab) => tab.id === activeId) ?? tabs[0]);

  return (
    <div className="layout">
      <nav className="sidebar" aria-label="Main menu">
        {/* Empty strip that clears the macOS traffic lights and drags the window. */}
        <div className="titlebar" />
        <ul className="menu">
          {tabs.map(({ id, label, Icon }) => {
            const isActive = id === active?.id;
            return (
              <li key={id}>
                <button
                  className={`menu-item${isActive ? ' is-active' : ''}`}
                  aria-current={isActive ? 'page' : undefined}
                  onClick={() => openTab(id)}
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
          {active && <active.Page />}
        </div>
      </main>
    </div>
  )
}

export default App
