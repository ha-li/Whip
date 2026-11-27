import { useEffect, useRef, useState, type CSSProperties, type KeyboardEvent, type PointerEvent } from 'react'
import { UI } from '../bindings/github.com/ha-li/Whip/internal/app'
import { About } from '../bindings/github.com/ha-li/Whip/internal/app/version'
import { tabs } from './tabs'
import './App.css'

// The side menu's width limits and default, in CSS pixels. The default
// matches --sidebar-width in styles/base.css.
const minSidebar = 160;
const maxSidebar = 400;
const defaultSidebar = 220;

function clampSidebar(width: number): number {
  return Math.round(Math.min(maxSidebar, Math.max(minSidebar, width)));
}

function saveSidebarWidth(width: number) {
  UI.SetSidebarWidth(width).catch((err) => console.error('Failed to save the menu width', err));
}

function App() {
  // null until the last open tab has been read, so the first tab doesn't flash.
  const [activeId, setActiveId] = useState<string | null>(null);
  const [version, setVersion] = useState<string>('');
  // null until the saved width has been read: the CSS default applies.
  const [sidebarWidth, setSidebarWidth] = useState<number | null>(null);
  const [resizing, setResizing] = useState(false);
  // Where a drag started: the pointer's x and the menu's width then.
  const dragStart = useRef<{ x: number, width: number } | null>(null);

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
    UI.SidebarWidth()
      .then((w) => {
        if (w > 0) setSidebarWidth(clampSidebar(w));
      })
      .catch((err) => console.error('Failed to read the menu width', err));
  }, []);

  // Dragging the handle on the menu's right edge resizes it; the width is
  // saved when the drag ends.
  const startResize = (e: PointerEvent<HTMLDivElement>) => {
    if (e.button !== 0) return;
    e.preventDefault();
    e.currentTarget.setPointerCapture(e.pointerId);
    dragStart.current = { x: e.clientX, width: sidebarWidth ?? defaultSidebar };
    setResizing(true);
  };
  const moveResize = (e: PointerEvent<HTMLDivElement>) => {
    const start = dragStart.current;
    if (!start) return;
    setSidebarWidth(clampSidebar(start.width + e.clientX - start.x));
  };
  const endResize = () => {
    if (!dragStart.current) return;
    dragStart.current = null;
    setResizing(false);
    if (sidebarWidth !== null) saveSidebarWidth(sidebarWidth);
  };
  // Double-clicking the handle puts the menu back to its default width.
  const resetWidth = () => {
    setSidebarWidth(defaultSidebar);
    saveSidebarWidth(defaultSidebar);
  };
  // ← and → resize from the keyboard, Home and End jump to the limits.
  const resizeKey = (e: KeyboardEvent<HTMLDivElement>) => {
    const current = sidebarWidth ?? defaultSidebar;
    const step = e.shiftKey ? 40 : 10;
    const next = {
      ArrowLeft: current - step,
      ArrowRight: current + step,
      Home: minSidebar,
      End: maxSidebar,
    }[e.key];
    if (next === undefined) return;
    e.preventDefault();
    const width = clampSidebar(next);
    setSidebarWidth(width);
    saveSidebarWidth(width);
  };

  useEffect(() => {
    About.Version()
      .then(setVersion)
      .catch((err) => console.error('Failed to load version', err));
  }, []);

  const active = activeId === null ? null : (tabs.find((tab) => tab.id === activeId) ?? tabs[0]);

  return (
    <div
      className={`layout${resizing ? ' is-resizing' : ''}`}
      style={sidebarWidth === null ? undefined : ({ '--sidebar-width': `${sidebarWidth}px` } as CSSProperties)}
    >
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
        <div
          className="sidebar-resizer"
          role="separator"
          aria-orientation="vertical"
          aria-label="Resize menu"
          aria-valuemin={minSidebar}
          aria-valuemax={maxSidebar}
          aria-valuenow={sidebarWidth ?? defaultSidebar}
          tabIndex={0}
          title="Drag to resize · double-click to reset"
          onPointerDown={startResize}
          onPointerMove={moveResize}
          onPointerUp={endResize}
          onPointerCancel={endResize}
          onDoubleClick={resetWidth}
          onKeyDown={resizeKey}
        />
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
