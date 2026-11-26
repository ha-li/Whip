import { useEffect, useMemo, useState, type FormEvent, type KeyboardEvent } from 'react'
import { Notes as NotesAPI, type Note } from '../../bindings/github.com/ha-li/Whip/internal/notes'
import './Notes.css'

// errorText turns a rejected service call into a message for the page.
function errorText(err: unknown): string {
  return err instanceof Error ? err.message : String(err);
}

const dateTime = new Intl.DateTimeFormat(undefined, { dateStyle: 'medium', timeStyle: 'short' });
// formatDateTime formats an ISO timestamp (e.g. a note's created time) in local time.
const formatDateTime = (iso: string) => dateTime.format(new Date(iso));

// submitOnCmdEnter submits the textarea's form on ⌘↩ (or Ctrl+Enter).
function submitOnCmdEnter(e: KeyboardEvent<HTMLTextAreaElement>) {
  if (e.key === 'Enter' && (e.metaKey || e.ctrlKey)) {
    e.preventDefault();
    e.currentTarget.form?.requestSubmit();
  }
}

// Notes lists the user's notes about stocks, newest first, optionally
// filtered to one ticker, and lets them add, edit and delete notes.
function Notes() {
  // null while loading.
  const [notes, setNotes] = useState<Note[] | null>(null);
  // The ticker to show notes for; '' shows all.
  const [filter, setFilter] = useState('');
  const [symbol, setSymbol] = useState('');
  const [draft, setDraft] = useState('');
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    const call = NotesAPI.List();
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

  // The tickers that have notes, for the filter.
  const symbols = useMemo(() => [...new Set((notes ?? []).map((n) => n.symbol))].sort(), [notes]);
  // A ticker whose last note was deleted falls back to showing all.
  const shown = symbols.includes(filter) ? filter : '';
  const visible = (notes ?? []).filter((n) => shown === '' || n.symbol === shown);

  const chooseFilter = (s: string) => {
    setFilter(s);
    if (s !== '') setSymbol(s);
  };

  const add = (e: FormEvent) => {
    e.preventDefault();
    if (symbol.trim() === '' || draft.trim() === '' || saving) return;
    setSaving(true);
    setError(null);
    NotesAPI.Add(symbol, draft)
      .then((note) => {
        setNotes((list) => [note, ...(list ?? [])]);
        setDraft('');
        // Keep the new note in view.
        if (shown !== '' && shown !== note.symbol) setFilter(note.symbol);
      })
      .catch((err) => setError(errorText(err)))
      .finally(() => setSaving(false));
  };

  const replace = (note: Note) => setNotes((list) => (list ?? []).map((n) => (n.id === note.id ? note : n)));
  const remove = (id: string) => setNotes((list) => (list ?? []).filter((n) => n.id !== id));

  return (
    <section className="notes">
      <div className="notes-header">
        <h1 className="page-title">Notes</h1>
        {symbols.length > 0 && (
          <select
            className="text-input notes-filter"
            aria-label="Show notes for"
            value={shown}
            onChange={(e) => chooseFilter(e.target.value)}
          >
            <option value="">All tickers</option>
            {symbols.map((s) => (
              <option key={s} value={s}>{s}</option>
            ))}
          </select>
        )}
      </div>

      <form className="notes-new" onSubmit={add}>
        <input
          className="text-input notes-symbol"
          aria-label="Ticker"
          placeholder="Ticker, e.g. AAPL"
          autoCapitalize="characters"
          spellCheck={false}
          value={symbol}
          onChange={(e) => setSymbol(e.target.value)}
        />
        <textarea
          className="text-input notes-textarea"
          aria-label="New note"
          placeholder="Add a note…"
          value={draft}
          onChange={(e) => setDraft(e.target.value)}
          onKeyDown={submitOnCmdEnter}
        />
        <div className="notes-actions">
          <span className="notes-hint">⌘↩ to add</span>
          <button type="submit" className="button" disabled={saving || symbol.trim() === '' || draft.trim() === ''}>
            Add note
          </button>
        </div>
      </form>

      {error && <p className="notes-error" role="alert">{error}</p>}

      {notes === null && <p className="page-subtitle">Loading notes…</p>}
      {notes !== null && visible.length === 0 && !error && <p className="page-subtitle">No notes yet.</p>}
      {visible.length > 0 && (
        <ul className="notes-list">
          {visible.map((note) => (
            <NoteItem
              key={note.id}
              note={note}
              onFilter={chooseFilter}
              onUpdated={replace}
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
  onFilter: (symbol: string) => void
  onUpdated: (note: Note) => void
  onDeleted: (id: string) => void
  onError: (message: string | null) => void
}

// NoteItem shows one note, with inline editing and a two-step delete.
function NoteItem({ note, onFilter, onUpdated, onDeleted, onError }: NoteItemProps) {
  const [mode, setMode] = useState<'view' | 'edit' | 'confirm-delete'>('view');
  const [text, setText] = useState(note.text);
  const [busy, setBusy] = useState(false);

  const cancelEdit = () => {
    setText(note.text);
    setMode('view');
  };

  const save = (e: FormEvent) => {
    e.preventDefault();
    if (text.trim() === '' || busy) return;
    setBusy(true);
    onError(null);
    NotesAPI.Update(note.id, text)
      .then((updated) => {
        onUpdated(updated);
        setMode('view');
      })
      .catch((err) => onError(errorText(err)))
      .finally(() => setBusy(false));
  };

  const del = () => {
    setBusy(true);
    onError(null);
    NotesAPI.Delete(note.id)
      .then(() => onDeleted(note.id))
      .catch((err) => {
        onError(errorText(err));
        setBusy(false);
        setMode('view');
      });
  };

  const edited = note.updated !== note.created;

  return (
    <li className="note">
      <div className="note-meta">
        <button
          type="button"
          className="link-button note-symbol"
          title={`Show only ${note.symbol}`}
          onClick={() => onFilter(note.symbol)}
        >
          {note.symbol}
        </button>
        {' · '}
        <time dateTime={note.created}>{formatDateTime(note.created)}</time>
        {edited && <span title={`Edited ${formatDateTime(note.updated)}`}> · edited</span>}
      </div>

      {mode === 'edit' ? (
        <form onSubmit={save}>
          <textarea
            className="text-input notes-textarea"
            aria-label={`Edit note about ${note.symbol}`}
            value={text}
            onChange={(e) => setText(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Escape') cancelEdit();
              else submitOnCmdEnter(e);
            }}
            autoFocus
          />
          <div className="notes-actions">
            <button type="button" className="link-button note-action" onClick={cancelEdit}>
              Cancel
            </button>
            <button type="submit" className="button" disabled={busy || text.trim() === ''}>
              Save
            </button>
          </div>
        </form>
      ) : (
        <>
          <p className="note-text">{note.text}</p>
          <div className="note-buttons">
            {mode === 'confirm-delete' ? (
              <>
                <span className="note-confirm">Delete this note?</span>
                <button type="button" className="link-button note-action" onClick={del} disabled={busy}>
                  Delete
                </button>
                <button type="button" className="link-button note-action" onClick={() => setMode('view')}>
                  Keep
                </button>
              </>
            ) : (
              <>
                <button type="button" className="link-button note-action" onClick={() => setMode('edit')}>
                  Edit
                </button>
                <button type="button" className="link-button note-action" onClick={() => setMode('confirm-delete')}>
                  Delete
                </button>
              </>
            )}
          </div>
        </>
      )}
    </li>
  )
}

export default Notes
