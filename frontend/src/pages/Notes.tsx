import { createContext, useCallback, useContext, useEffect, useLayoutEffect, useMemo, useRef, useState, type CSSProperties, type FormEvent, type KeyboardEvent, type MouseEvent, type ReactNode } from 'react'
import { Notebook, type Note } from '../../bindings/github.com/ha-li/Whip/internal/notebook'
import { Palette, type Color } from '../../bindings/github.com/ha-li/Whip/internal/palette'
import { errorText, formatDateTime } from '../format'
import { TrashIcon } from '../icons'
// The layout and list styles are shared with the Portfolio page.
import './Portfolio.css'
import './Notes.css'

// Editing is the note open in the editor and what the editor held before,
// restored when the edit ends so an unsent new note isn't lost.
type Editing = {
  note: Note
  before: { title: string, draft: string, tags: string[] }
}

// normalizeTag returns tag as the backend stores it: trimmed, lower case,
// with runs of spaces made one.
function normalizeTag(tag: string): string {
  return tag.trim().toLowerCase().split(/\s+/).filter(Boolean).join(' ');
}

// addTags returns tags with each new one added, normalized, once, sorted.
function addTags(tags: string[], ...more: string[]): string[] {
  const added = more.map(normalizeTag).filter((t) => t !== '');
  return [...new Set([...tags, ...added])].sort();
}

// newestFirst orders notes the way List does: latest created first.
function newestFirst(a: Note, b: Note): number {
  if (a.created !== b.created) return Date.parse(b.created) - Date.parse(a.created);
  return a.id < b.id ? 1 : a.id > b.id ? -1 : 0;
}

// tagsOf returns the note's tags; notes from before tags have none.
function tagsOf(note: Note): string[] {
  return note.tags ?? [];
}

// matches reports whether the note's title, text or tags contain every word
// of query, ignoring case.
function matches(note: Note, query: string): boolean {
  const hay = `${note.title}\n${note.text}\n${tagsOf(note).join('\n')}`.toLowerCase();
  return query.toLowerCase().split(/\s+/).filter(Boolean).every((w) => hay.includes(w));
}

// TagStyle gives the pill style for a tag: its palette colour, or undefined
// for a tag without one.
const TagStyle = createContext<(tag: string) => CSSProperties | undefined>(() => undefined);

type View = 'notes' | 'trash'

// Notes lists the user's titled, tagged notes, newest first, optionally
// narrowed by a tag and a search. Every tag is listed beside the editor,
// where it can be given a colour. One editor at the top adds notes and, after
// Edit on a note, changes that note's title, text and tags. The trash view lists deleted notes,
// restores them, and empties the trash for good.
function Notes() {
  const [view, setView] = useState<View>('notes');
  // null while loading.
  const [notes, setNotes] = useState<Note[] | null>(null);
  // Deleted notes, most recently deleted first; null while loading.
  const [trash, setTrash] = useState<Note[] | null>(null);
  const [query, setQuery] = useState('');
  // The tag to show notes for; '' shows all.
  const [tagFilter, setTagFilter] = useState('');
  const [title, setTitle] = useState('');
  const [draft, setDraft] = useState('');
  // The editor's tags, and the one being typed but not yet added.
  const [tags, setTags] = useState<string[]>([]);
  const [tagDraft, setTagDraft] = useState('');
  // The palette, and each coloured tag's colour name.
  const [colors, setColors] = useState<Color[]>([]);
  const [tagColors, setTagColors] = useState<Record<string, string>>({});
  const [editing, setEditing] = useState<Editing | null>(null);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const editorRef = useRef<HTMLFormElement>(null);
  const textRef = useRef<HTMLTextAreaElement>(null);

  useEffect(() => {
    let cancelled = false;
    const call = Notebook.List();
    call
      .then((list) => {
        if (!cancelled) setNotes(list ?? []);
      })
      .catch((err) => {
        if (!cancelled) {
          setNotes([]);
          setError(errorText(err));
        }
      });
    return () => {
      cancelled = true;
      call.cancel();
    };
  }, []);

  useEffect(() => {
    Palette.Colors()
      .then((list) => setColors(list ?? []))
      .catch((err) => console.error('Failed to load the colour palette', err));
    Notebook.TagColors()
      .then((map) => {
        const out: Record<string, string> = {};
        for (const [tag, color] of Object.entries(map ?? {})) if (color) out[tag] = color;
        setTagColors(out);
      })
      .catch((err) => setError(errorText(err)));
  }, []);

  // Reload the trash each time it's opened: notes deleted since are in it.
  useEffect(() => {
    if (view !== 'trash') return;
    let cancelled = false;
    setTrash(null);
    const call = Notebook.ListDeleted();
    call
      .then((list) => {
        if (!cancelled) setTrash(list ?? []);
      })
      .catch((err) => {
        if (!cancelled) {
          setTrash([]);
          setError(errorText(err));
        }
      });
    return () => {
      cancelled = true;
      call.cancel();
    };
  }, [view]);

  // Bring the editor into view with the cursor at the end of the note's text.
  useEffect(() => {
    if (!editing) return;
    editorRef.current?.scrollIntoView({ block: 'nearest' });
    const text = textRef.current;
    if (text) {
      text.focus();
      text.setSelectionRange(text.value.length, text.value.length);
    }
  }, [editing]);

  const current = view === 'notes' ? notes : trash;
  // The tags on notes in this view, with how many notes have each.
  const tagCounts = useMemo(() => {
    const counts = new Map<string, number>();
    for (const note of current ?? []) {
      for (const tag of tagsOf(note)) counts.set(tag, (counts.get(tag) ?? 0) + 1);
    }
    return [...counts].sort(([a], [b]) => (a < b ? -1 : a > b ? 1 : 0));
  }, [current]);
  const allTags = useMemo(() => tagCounts.map(([tag]) => tag), [tagCounts]);
  // Every tag on any note, suggested while typing a tag.
  const knownTags = useMemo(() => [...new Set([...(notes ?? []), ...(trash ?? [])].flatMap(tagsOf))].sort(), [notes, trash]);
  // A tag no note in this view has falls back to showing all.
  const shownTag = allTags.includes(tagFilter) ? tagFilter : '';
  const visible = (current ?? []).filter(
    (n) => (shownTag === '' || tagsOf(n).includes(shownTag)) && (query.trim() === '' || matches(n, query)),
  );

  const switchView = (v: View) => {
    setError(null);
    setView(v);
  };

  const tagStyle = useMemo(() => {
    const byName = new Map(colors.map((c) => [c.name, c]));
    return (tag: string): CSSProperties | undefined => {
      const c = byName.get(tagColors[tag] ?? '');
      return c && { background: c.background, color: c.text, borderColor: 'transparent' };
    };
  }, [colors, tagColors]);

  const setTagColor = (tag: string, color: string) => {
    setError(null);
    Notebook.SetTagColor(tag, color)
      .then((stored) =>
        setTagColors((cur) => {
          const next = { ...cur };
          if (color === '') delete next[stored];
          else next[stored] = color;
          return next;
        }),
      )
      .catch((err) => setError(errorText(err)));
  };

  // Clicking a note's tag filters by it; clicking the tag filtered by shows all.
  const toggleTagFilter = (tag: string) => setTagFilter((cur) => (cur === tag ? '' : tag));

  const startEdit = (note: Note) => {
    setError(null);
    // Switching from one note to another keeps the original unsent draft.
    setEditing((cur) => ({ note, before: cur?.before ?? { title, draft, tags: addTags(tags, tagDraft) } }));
    setTitle(note.title);
    setDraft(note.text);
    setTags(tagsOf(note));
    setTagDraft('');
  };

  const endEdit = () => {
    if (!editing) return;
    setTitle(editing.before.title);
    setDraft(editing.before.draft);
    setTags(editing.before.tags);
    setTagDraft('');
    setEditing(null);
  };

  const submit = (e: FormEvent) => {
    e.preventDefault();
    if (title.trim() === '' || draft.trim() === '' || saving) return;
    setSaving(true);
    setError(null);
    // A tag still being typed is saved too.
    const noteTags = addTags(tags, tagDraft);
    setTags(noteTags);
    setTagDraft('');
    if (editing) {
      Notebook.Update(editing.note.id, title, draft, noteTags)
        .then((note) => {
          setNotes((list) => (list ?? []).map((n) => (n.id === note.id ? note : n)));
          endEdit();
        })
        .catch((err) => setError(errorText(err)))
        .finally(() => setSaving(false));
      return;
    }
    Notebook.Add(title, draft, noteTags)
      .then((note) => {
        setNotes((list) => [note, ...(list ?? [])]);
        setTitle('');
        setDraft('');
        setTags([]);
        // Keep the new note in view.
        if (!matches(note, query)) setQuery('');
        if (shownTag !== '' && !tagsOf(note).includes(shownTag)) setTagFilter('');
      })
      .catch((err) => setError(errorText(err)))
      .finally(() => setSaving(false));
  };

  // ⌘↩ (or Ctrl+Enter) saves; Esc stops editing.
  const onKeyDown = (e: KeyboardEvent<HTMLInputElement | HTMLTextAreaElement>) => {
    if (e.key === 'Enter' && (e.metaKey || e.ctrlKey)) {
      e.preventDefault();
      e.currentTarget.form?.requestSubmit();
    } else if (e.key === 'Escape' && editing) {
      e.preventDefault();
      endEdit();
    }
  };

  // Enter in the title moves on to the text rather than submitting.
  const onTitleKeyDown = (e: KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter' && !e.metaKey && !e.ctrlKey) {
      e.preventDefault();
      textRef.current?.focus();
      return;
    }
    onKeyDown(e);
  };

  const remove = (id: string) => {
    setNotes((list) => (list ?? []).filter((n) => n.id !== id));
    if (editing?.note.id === id) endEdit();
  };

  // restored moves a restored note from the trash back into the notes.
  const restored = (note: Note) => {
    setTrash((list) => (list ?? []).filter((n) => n.id !== note.id));
    setNotes((list) => [...(list ?? []).filter((n) => n.id !== note.id), note].sort(newestFirst));
  };

  const filtered = query.trim() !== '' || shownTag !== '';

  return (
    <TagStyle.Provider value={tagStyle}>
      <section className="notes notebook">
        <div className="notes-header">
          <h1 className="page-title">Notes</h1>
          <div className="notes-controls">
            <div className="segmented" role="group" aria-label="Show">
              <button type="button" className="segmented-item" aria-pressed={view === 'notes'} onClick={() => switchView('notes')}>
                Notes
              </button>
              <button
                type="button"
                className="segmented-item notes-trash-toggle"
                aria-pressed={view === 'trash'}
                aria-label="Trash"
                title="Trash"
                onClick={() => switchView('trash')}
              >
                <TrashIcon />
              </button>
            </div>
            {(current ?? []).length > 0 && (
              <input
                type="search"
                className="text-input notes-filter"
                aria-label="Search notes"
                placeholder="Search"
                value={query}
                onChange={(e) => setQuery(e.target.value)}
              />
            )}
          </div>
        </div>

        <div className={`notebook-top${view === 'notes' ? ' has-editor' : ''}`}>
          {view === 'notes' && (
            <form ref={editorRef} className={`notes-new${editing ? ' is-editing' : ''}`} onSubmit={submit}>
              {editing && (
                <p className="notes-editing">
                  Editing a note from {formatDateTime(editing.note.created)}
                </p>
              )}
              <input
                className="text-input notes-title"
                aria-label="Title"
                placeholder="Title"
                maxLength={200}
                value={title}
                onChange={(e) => setTitle(e.target.value)}
                onKeyDown={onTitleKeyDown}
              />
              <textarea
                ref={textRef}
                className="text-input notes-textarea"
                aria-label={editing ? `Edit note “${editing.note.title}”` : 'New note'}
                placeholder="Add a note…"
                value={draft}
                onChange={(e) => setDraft(e.target.value)}
                onKeyDown={onKeyDown}
              />
              <TagInput
                tags={tags}
                draft={tagDraft}
                known={knownTags}
                onTags={setTags}
                onDraft={setTagDraft}
                onKeyDown={onKeyDown}
              />
              <div className="notes-actions">
                <span className="notes-hint">{editing ? '⌘↩ to save · Esc to cancel' : '⌘↩ to add'}</span>
                {editing && (
                  <button type="button" className="link-button note-action" onClick={endEdit}>
                    Cancel
                  </button>
                )}
                <button type="submit" className="button" disabled={saving || title.trim() === '' || draft.trim() === ''}>
                  {editing ? 'Save' : 'Add note'}
                </button>
              </div>
            </form>
          )}
          <TagPanel
            tags={tagCounts}
            activeTag={shownTag}
            colors={colors}
            colorOf={(tag) => tagColors[tag] ?? ''}
            onTag={toggleTagFilter}
            onColor={setTagColor}
          />
        </div>

        {error && <p className="notes-error" role="alert">{error}</p>}

        {current === null && <p className="page-subtitle">Loading notes…</p>}
        {current !== null && visible.length === 0 && !error && (
          <p className="page-subtitle">
            {filtered ? 'No notes match.' : view === 'notes' ? 'No notes yet.' : 'The trash is empty.'}
          </p>
        )}
        {view === 'trash' && trash !== null && trash.length > 0 && (
          <EmptyTrash count={trash.length} filtered={filtered} onEmptied={() => setTrash([])} onError={setError} />
        )}
        {view === 'trash' && visible.length > 0 && (
          <ul className="notes-list">
            {visible.map((note) => (
              <TrashItem
                key={note.id}
                note={note}
                activeTag={shownTag}
                onTag={toggleTagFilter}
                onRestored={restored}
                onError={setError}
              />
            ))}
          </ul>
        )}
        {view === 'notes' && visible.length > 0 && (
          <ul className="notes-list">
            {visible.map((note) => (
              <NoteItem
                key={note.id}
                note={note}
                isEditing={editing?.note.id === note.id}
                activeTag={shownTag}
                onTag={toggleTagFilter}
                onEdit={startEdit}
                onDeleted={remove}
                onError={setError}
              />
            ))}
          </ul>
        )}
      </section>
    </TagStyle.Provider>
  )
}

type TagInputProps = {
  tags: string[]
  // draft is the tag being typed, not yet added.
  draft: string
  // known are existing tags, suggested while typing.
  known: string[]
  onTags: (tags: string[]) => void
  onDraft: (draft: string) => void
  // onKeyDown handles the editor's shortcuts (⌘↩, Esc).
  onKeyDown: (e: KeyboardEvent<HTMLInputElement>) => void
}

// TagInput edits the editor's tags as pills. Enter, a comma or leaving the
// field adds the typed tag; Backspace in the empty field removes the last.
function TagInput({ tags, draft, known, onTags, onDraft, onKeyDown }: TagInputProps) {
  const inputRef = useRef<HTMLInputElement>(null);
  const commit = () => {
    if (draft.trim() === '') return;
    onTags(addTags(tags, draft));
    onDraft('');
  };

  const keyDown = (e: KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter' && !e.metaKey && !e.ctrlKey) {
      e.preventDefault();
      commit();
    } else if (e.key === 'Backspace' && draft === '' && tags.length > 0) {
      e.preventDefault();
      onTags(tags.slice(0, -1));
    } else {
      onKeyDown(e);
    }
  };

  // A pasted or typed comma splits the text into tags.
  const change = (value: string) => {
    if (!value.includes(',')) {
      onDraft(value);
      return;
    }
    const parts = value.split(',');
    onTags(addTags(tags, ...parts.slice(0, -1)));
    onDraft(parts[parts.length - 1]);
  };

  const styleOf = useContext(TagStyle);
  const suggestions = known.filter((t) => !tags.includes(t));
  return (
    // Clicking the field outside a pill types in it.
    <div
      className="text-input notes-tags"
      onMouseDown={(e) => {
        if (e.target === e.currentTarget) {
          e.preventDefault();
          inputRef.current?.focus();
        }
      }}
    >
      {tags.map((tag) => (
        <span key={tag} className="tag-pill" style={styleOf(tag)}>
          {tag}
          <button
            type="button"
            className="tag-remove"
            aria-label={`Remove tag ${tag}`}
            onClick={() => onTags(tags.filter((t) => t !== tag))}
          >
            ×
          </button>
        </span>
      ))}
      <input
        ref={inputRef}
        className="notes-tags-input"
        aria-label="Add a tag"
        placeholder={tags.length === 0 ? 'Tags, e.g. work, ideas' : 'Add a tag'}
        list="notes-known-tags"
        maxLength={32}
        spellCheck={false}
        value={draft}
        onChange={(e) => change(e.target.value)}
        onKeyDown={keyDown}
        onBlur={commit}
      />
      <datalist id="notes-known-tags">
        {suggestions.map((t) => (
          <option key={t} value={t} />
        ))}
      </datalist>
    </div>
  )
}

type TagPillsProps = {
  tags: string[]
  // activeTag is the tag the list is filtered by, or ''.
  activeTag: string
  onTag: (tag: string) => void
}

// TagPills shows a note's tags; clicking one filters the list by it.
function TagPills({ tags, activeTag, onTag }: TagPillsProps) {
  const styleOf = useContext(TagStyle);
  if (tags.length === 0) return null;
  return (
    <ul className="note-tags" aria-label="Tags">
      {tags.map((tag) => (
        <li key={tag}>
          <button
            type="button"
            className="tag-pill"
            style={styleOf(tag)}
            aria-pressed={tag === activeTag}
            title={tag === activeTag ? 'Show all notes' : `Show only notes tagged ${tag}`}
            onClick={() => onTag(tag)}
          >
            {tag}
          </button>
        </li>
      ))}
    </ul>
  )
}

type TagPanelProps = {
  // tags are the view's tags, sorted, with how many notes have each.
  tags: [string, number][]
  // activeTag is the tag the list is filtered by, or ''.
  activeTag: string
  colors: Color[]
  // colorOf returns a tag's colour name, or '' for none.
  colorOf: (tag: string) => string
  onTag: (tag: string) => void
  onColor: (tag: string, color: string) => void
}

// TagPanel lists every tag as a pill beside the editor. Clicking a pill
// filters the notes by it; the dot beside it picks the pill's colour.
function TagPanel({ tags, activeTag, colors, colorOf, onTag, onColor }: TagPanelProps) {
  const styleOf = useContext(TagStyle);
  // The tag whose colour menu is open, and where to put the menu.
  const [picking, setPicking] = useState<{ tag: string, x: number, y: number } | null>(null);
  const closeMenu = useCallback(() => setPicking(null), []);

  const openMenu = (tag: string, e: MouseEvent<HTMLButtonElement>) => {
    if (picking?.tag === tag) {
      setPicking(null);
      return;
    }
    const r = e.currentTarget.getBoundingClientRect();
    setPicking({ tag, x: r.left, y: r.bottom + 4 });
  };

  return (
    <aside className="tag-panel" aria-label="Tags">
      <div className="tag-panel-header">
        <h2 className="tag-panel-title">Tags</h2>
        {activeTag !== '' && (
          <button type="button" className="link-button note-action" onClick={() => onTag(activeTag)}>
            Show all
          </button>
        )}
      </div>
      {tags.length === 0 ? (
        <p className="tag-panel-empty">No tags yet.</p>
      ) : (
        <ul className="tag-panel-list">
          {tags.map(([tag, count]) => (
            <li key={tag} className="tag-panel-item">
              <button
                type="button"
                className="tag-pill"
                style={styleOf(tag)}
                aria-pressed={tag === activeTag}
                title={tag === activeTag ? 'Show all notes' : `Show only notes tagged ${tag}`}
                onClick={() => onTag(tag)}
              >
                {tag}
                <span className="tag-count">{count}</span>
              </button>
              <button
                type="button"
                className="tag-color-button"
                style={styleOf(tag)}
                aria-label={`Colour for ${tag}`}
                aria-haspopup="true"
                aria-expanded={picking?.tag === tag}
                title="Change colour"
                onClick={(e) => openMenu(tag, e)}
              />
            </li>
          ))}
        </ul>
      )}
      {picking && (
        <ColorMenu
          tag={picking.tag}
          x={picking.x}
          y={picking.y}
          colors={colors}
          current={colorOf(picking.tag)}
          onPick={(color) => {
            onColor(picking.tag, color);
            setPicking(null);
          }}
          onClose={closeMenu}
        />
      )}
    </aside>
  )
}

type ColorMenuProps = {
  tag: string
  // x and y place the menu's top left corner, in viewport pixels.
  x: number
  y: number
  colors: Color[]
  // current is the tag's colour name, or '' for none.
  current: string
  onPick: (color: string) => void
  onClose: () => void
}

// ColorMenu offers the palette's colours, and none, for a tag. Esc or a
// click outside closes it.
function ColorMenu({ tag, x, y, colors, current, onPick, onClose }: ColorMenuProps) {
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    ref.current?.querySelector<HTMLButtonElement>('[aria-checked="true"]')?.focus();
    const onDown = (e: globalThis.MouseEvent) => {
      const target = e.target as Element;
      // The dot that opened the menu toggles it itself.
      if (!ref.current?.contains(target) && !target.closest('.tag-color-button')) onClose();
    };
    const onKey = (e: globalThis.KeyboardEvent) => {
      if (e.key === 'Escape') onClose();
    };
    document.addEventListener('mousedown', onDown);
    document.addEventListener('keydown', onKey);
    return () => {
      document.removeEventListener('mousedown', onDown);
      document.removeEventListener('keydown', onKey);
    };
  }, [onClose]);

  // Keep the menu inside the window.
  const left = Math.max(8, Math.min(x, window.innerWidth - 200));
  return (
    <div ref={ref} className="color-menu" role="radiogroup" aria-label={`Colour for ${tag}`} style={{ left, top: y }}>
      <button
        type="button"
        role="radio"
        aria-checked={current === ''}
        className="color-swatch is-none"
        title="No colour"
        aria-label="No colour"
        onClick={() => onPick('')}
      />
      {colors.map((c) => (
        <button
          key={c.name}
          type="button"
          role="radio"
          aria-checked={current === c.name}
          className="color-swatch"
          style={{ background: c.background, color: c.text }}
          title={c.label}
          aria-label={c.label}
          onClick={() => onPick(c.name)}
        />
      ))}
    </div>
  )
}

// useExpandable tracks whether a note's text is longer than its collapsed
// height (10 lines) and whether the user has expanded it.
function useExpandable(text: string) {
  const ref = useRef<HTMLParagraphElement>(null);
  const [expanded, setExpanded] = useState(false);
  const [overflows, setOverflows] = useState(false);

  // Measure while collapsed, again whenever the text or the width changes.
  useLayoutEffect(() => {
    const el = ref.current;
    if (!el || expanded) return;
    const check = () => setOverflows(el.scrollHeight > el.clientHeight + 1);
    check();
    const observer = new ResizeObserver(check);
    observer.observe(el);
    return () => observer.disconnect();
  }, [text, expanded]);

  const collapse = () => {
    setExpanded(false);
    if (ref.current) ref.current.scrollTop = 0;
  };
  return { ref, expanded, overflows, expand: () => setExpanded(true), collapse };
}

type NoteBodyProps = {
  note: Note
  className: string
  activeTag: string
  onTag: (tag: string) => void
  // onOpen, if given, is called when the note is clicked: it opens the note
  // in the editor. Without it, clicking a longer note expands it.
  onOpen?: () => void
  // meta and buttons go below the title and the text.
  meta: ReactNode
  buttons: ReactNode
}

// NoteBody shows a note's title and its first 10 lines. "Show more" expands
// a longer note to 50 lines, scrolling the rest, and "Show less" collapses
// it again. Clicking the note calls onOpen, or expands it if there's none.
function NoteBody({ note, className, activeTag, onTag, onOpen, meta, buttons }: NoteBodyProps) {
  const { ref, expanded, overflows, expand, collapse } = useExpandable(note.text);

  const onClick = (e: MouseEvent<HTMLLIElement>) => {
    // Leave clicks on buttons (and tag pills) alone, and do nothing when
    // ending a text selection.
    if ((e.target as Element).closest('button')) return;
    if (window.getSelection()?.toString()) return;
    if (onOpen) onOpen();
    else if (overflows && !expanded) expand();
  };

  const classes = [
    className,
    onOpen ? 'is-openable' : '',
    !onOpen && overflows ? 'is-expandable' : '',
    expanded ? 'is-expanded' : '',
  ].filter(Boolean).join(' ');
  return (
    <li className={classes} onClick={onClick}>
      <div className="note-head">
        <h2 className="note-title">
          {/* The title is a button too, so the keyboard can open the note. */}
          {onOpen ? (
            <button type="button" className="note-title-button" onClick={onOpen}>
              {note.title}
            </button>
          ) : overflows ? (
            <button
              type="button"
              className="note-title-button"
              aria-expanded={expanded}
              onClick={expanded ? collapse : expand}
            >
              {note.title}
            </button>
          ) : (
            note.title
          )}
        </h2>
        <TagPills tags={tagsOf(note)} activeTag={activeTag} onTag={onTag} />
      </div>
      {meta}
      <p ref={ref} className={`note-text ${expanded ? 'is-scrollable' : 'is-clamped'}`}>{note.text}</p>
      <div className="note-buttons">
        {buttons}
        {overflows && (
          <button type="button" className="link-button note-action" onClick={expanded ? collapse : expand}>
            {expanded ? 'Show less' : 'Show more'}
          </button>
        )}
      </div>
    </li>
  )
}

type NoteItemProps = {
  note: Note
  // isEditing is true while the note is open in the editor.
  isEditing: boolean
  activeTag: string
  onTag: (tag: string) => void
  onEdit: (note: Note) => void
  onDeleted: (id: string) => void
  onError: (message: string | null) => void
}

// NoteItem shows one note, with Edit (which opens it in the editor) and a
// two-step delete.
function NoteItem({ note, isEditing, activeTag, onTag, onEdit, onDeleted, onError }: NoteItemProps) {
  const [confirming, setConfirming] = useState(false);
  const [busy, setBusy] = useState(false);

  const del = () => {
    setBusy(true);
    onError(null);
    Notebook.Delete(note.id)
      .then(() => onDeleted(note.id))
      .catch((err) => {
        onError(errorText(err));
        setBusy(false);
        setConfirming(false);
      });
  };

  const edited = note.updated !== note.created;

  return (
    <NoteBody
      note={note}
      className={`note${isEditing ? ' is-editing' : ''}`}
      activeTag={activeTag}
      onTag={onTag}
      onOpen={() => {
        if (!isEditing) onEdit(note);
      }}
      meta={
        <div className="note-meta">
          <time dateTime={note.created}>{formatDateTime(note.created)}</time>
          {edited && <span title={`Edited ${formatDateTime(note.updated)}`}> · edited</span>}
        </div>
      }
      buttons={confirming ? (
        <>
          <span className="note-confirm">Delete this note?</span>
          <button type="button" className="link-button note-action" onClick={del} disabled={busy}>
            Delete
          </button>
          <button type="button" className="link-button note-action" onClick={() => setConfirming(false)}>
            Keep
          </button>
        </>
      ) : isEditing ? (
        <span className="note-confirm">Editing above</span>
      ) : (
        <>
          <button type="button" className="link-button note-action" onClick={() => onEdit(note)}>
            Edit
          </button>
          <button type="button" className="link-button note-action" onClick={() => setConfirming(true)}>
            Delete
          </button>
        </>
      )}
    />
  )
}

type EmptyTrashProps = {
  // count is how many notes are in the trash, including any the filters hide.
  count: number
  filtered: boolean
  onEmptied: () => void
  onError: (message: string | null) => void
}

// EmptyTrash permanently deletes every note in the trash, after asking.
function EmptyTrash({ count, filtered, onEmptied, onError }: EmptyTrashProps) {
  const [confirming, setConfirming] = useState(false);
  const [busy, setBusy] = useState(false);

  const empty = () => {
    setBusy(true);
    onError(null);
    Notebook.EmptyTrash()
      .then(onEmptied)
      .catch((err) => onError(errorText(err)))
      .finally(() => {
        setBusy(false);
        setConfirming(false);
      });
  };

  const notes = count === 1 ? '1 note' : `${count} notes`;
  return (
    <div className="notes-empty-trash">
      {confirming ? (
        <>
          <span className="note-confirm">
            Permanently delete {filtered ? `all ${notes} in the trash, not just the ones shown` : notes}? This can't be undone.
          </span>
          <button type="button" className="link-button note-action" onClick={empty} disabled={busy}>
            Delete forever
          </button>
          <button type="button" className="link-button note-action" onClick={() => setConfirming(false)}>
            Cancel
          </button>
        </>
      ) : (
        <button type="button" className="link-button note-action" onClick={() => setConfirming(true)}>
          Empty trash
        </button>
      )}
    </div>
  )
}

type TrashItemProps = {
  note: Note
  activeTag: string
  onTag: (tag: string) => void
  onRestored: (note: Note) => void
  onError: (message: string | null) => void
}

// TrashItem shows one deleted note, with when it was deleted and a button
// to restore it.
function TrashItem({ note, activeTag, onTag, onRestored, onError }: TrashItemProps) {
  const [busy, setBusy] = useState(false);

  const restore = () => {
    setBusy(true);
    onError(null);
    Notebook.Restore(note.id)
      .then(onRestored)
      .catch((err) => {
        onError(errorText(err));
        setBusy(false);
      });
  };

  return (
    <NoteBody
      note={note}
      className="note is-trashed"
      activeTag={activeTag}
      onTag={onTag}
      meta={
        <div className="note-meta">
          <time dateTime={note.created}>{formatDateTime(note.created)}</time>
          {note.deleted && (
            <>
              {' · deleted '}
              <time dateTime={note.deleted}>{formatDateTime(note.deleted)}</time>
            </>
          )}
        </div>
      }
      buttons={
        <button type="button" className="link-button note-action" onClick={restore} disabled={busy}>
          Restore
        </button>
      }
    />
  )
}

export default Notes
