import { useEffect, useRef, useState, type FormEvent, type KeyboardEvent } from 'react'
import { Notebook, type Note } from '../../bindings/github.com/ha-li/Whip/internal/notebook'
import { errorText, formatDateTime } from '../format'
import { TrashIcon } from '../icons'
// The layout and list styles are shared with the Portfolio page.
import './Portfolio.css'
import './Notes.css'

// Editing is the note open in the editor and what the editor held before,
// restored when the edit ends so an unsent new note isn't lost.
type Editing = {
  note: Note
  before: { title: string, draft: string }
}

// newestFirst orders notes the way List does: latest created first.
function newestFirst(a: Note, b: Note): number {
  if (a.created !== b.created) return Date.parse(b.created) - Date.parse(a.created);
  return a.id < b.id ? 1 : a.id > b.id ? -1 : 0;
}

// matches reports whether the note's title or text contains every word of
// query, ignoring case.
function matches(note: Note, query: string): boolean {
  const hay = `${note.title}\n${note.text}`.toLowerCase();
  return query.toLowerCase().split(/\s+/).filter(Boolean).every((w) => hay.includes(w));
}

type View = 'notes' | 'trash'

// Notes lists the user's titled notes, newest first, optionally narrowed by
// a search. One editor at the top adds notes and, after Edit on a note,
// changes that note's title and text. The trash view lists deleted notes,
// restores them, and empties the trash for good.
function Notes() {
  const [view, setView] = useState<View>('notes');
  // null while loading.
  const [notes, setNotes] = useState<Note[] | null>(null);
  // Deleted notes, most recently deleted first; null while loading.
  const [trash, setTrash] = useState<Note[] | null>(null);
  const [query, setQuery] = useState('');
  const [title, setTitle] = useState('');
  const [draft, setDraft] = useState('');
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
  const visible = (current ?? []).filter((n) => query.trim() === '' || matches(n, query));

  const switchView = (v: View) => {
    setError(null);
    setView(v);
  };

  const startEdit = (note: Note) => {
    setError(null);
    // Switching from one note to another keeps the original unsent draft.
    setEditing((cur) => ({ note, before: cur?.before ?? { title, draft } }));
    setTitle(note.title);
    setDraft(note.text);
  };

  const endEdit = () => {
    if (!editing) return;
    setTitle(editing.before.title);
    setDraft(editing.before.draft);
    setEditing(null);
  };

  const submit = (e: FormEvent) => {
    e.preventDefault();
    if (title.trim() === '' || draft.trim() === '' || saving) return;
    setSaving(true);
    setError(null);
    if (editing) {
      Notebook.Update(editing.note.id, title, draft)
        .then((note) => {
          setNotes((list) => (list ?? []).map((n) => (n.id === note.id ? note : n)));
          endEdit();
        })
        .catch((err) => setError(errorText(err)))
        .finally(() => setSaving(false));
      return;
    }
    Notebook.Add(title, draft)
      .then((note) => {
        setNotes((list) => [note, ...(list ?? [])]);
        setTitle('');
        setDraft('');
        // Keep the new note in view.
        if (!matches(note, query)) setQuery('');
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

  const searching = query.trim() !== '';

  return (
    <section className="notes">
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

      {error && <p className="notes-error" role="alert">{error}</p>}

      {current === null && <p className="page-subtitle">Loading notes…</p>}
      {current !== null && visible.length === 0 && !error && (
        <p className="page-subtitle">
          {searching ? 'No notes match.' : view === 'notes' ? 'No notes yet.' : 'The trash is empty.'}
        </p>
      )}
      {view === 'trash' && trash !== null && trash.length > 0 && (
        <EmptyTrash count={trash.length} filtered={searching} onEmptied={() => setTrash([])} onError={setError} />
      )}
      {view === 'trash' && visible.length > 0 && (
        <ul className="notes-list">
          {visible.map((note) => (
            <TrashItem key={note.id} note={note} onRestored={restored} onError={setError} />
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
              onEdit={startEdit}
              onDeleted={remove}
              onError={setError}
            />
          ))}
        </ul>
      )}
    </section>
  )
}

type NoteItemProps = {
  note: Note
  // isEditing is true while the note is open in the editor.
  isEditing: boolean
  onEdit: (note: Note) => void
  onDeleted: (id: string) => void
  onError: (message: string | null) => void
}

// NoteItem shows one note, with Edit (which opens it in the editor) and a
// two-step delete.
function NoteItem({ note, isEditing, onEdit, onDeleted, onError }: NoteItemProps) {
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
    <li className={`note${isEditing ? ' is-editing' : ''}`}>
      <h2 className="note-title">{note.title}</h2>
      <div className="note-meta">
        <time dateTime={note.created}>{formatDateTime(note.created)}</time>
        {edited && <span title={`Edited ${formatDateTime(note.updated)}`}> · edited</span>}
      </div>
      <p className="note-text">{note.text}</p>
      <div className="note-buttons">
        {confirming ? (
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
      </div>
    </li>
  )
}

type EmptyTrashProps = {
  // count is how many notes are in the trash, including any the search hides.
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
  onRestored: (note: Note) => void
  onError: (message: string | null) => void
}

// TrashItem shows one deleted note, with when it was deleted and a button
// to restore it.
function TrashItem({ note, onRestored, onError }: TrashItemProps) {
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
    <li className="note is-trashed">
      <h2 className="note-title">{note.title}</h2>
      <div className="note-meta">
        <time dateTime={note.created}>{formatDateTime(note.created)}</time>
        {note.deleted && (
          <>
            {' · deleted '}
            <time dateTime={note.deleted}>{formatDateTime(note.deleted)}</time>
          </>
        )}
      </div>
      <p className="note-text">{note.text}</p>
      <div className="note-buttons">
        <button type="button" className="link-button note-action" onClick={restore} disabled={busy}>
          Restore
        </button>
      </div>
    </li>
  )
}

export default Notes
