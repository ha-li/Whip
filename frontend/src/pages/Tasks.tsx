import { useEffect, useState, type FormEvent } from 'react'
import { Tasks as TasksAPI, type Task } from '../../bindings/github.com/ha-li/Whip/internal/tasks'
import { errorText, formatDateTime } from '../format'
import './Tasks.css'

type View = 'open' | 'completed'

// Tasks is a to-do list. Ticking a task completes it and moves it to the
// Completed view, where it can be reopened or deleted for good.
function Tasks() {
  const [view, setView] = useState<View>('open');
  // null while loading.
  const [tasks, setTasks] = useState<Task[] | null>(null);
  const [draft, setDraft] = useState('');
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Load the view's tasks each time it's opened: tasks completed or reopened
  // in the other view have moved.
  useEffect(() => {
    let cancelled = false;
    setTasks(null);
    const call = view === 'open' ? TasksAPI.List() : TasksAPI.ListCompleted();
    call
      .then((list) => {
        if (!cancelled) setTasks(list ?? []);
      })
      .catch((err) => {
        if (!cancelled) {
          setTasks([]);
          setError(errorText(err));
        }
      });
    return () => {
      cancelled = true;
      call.cancel();
    };
  }, [view]);

  const switchView = (v: View) => {
    setError(null);
    setView(v);
  };

  const add = (e: FormEvent) => {
    e.preventDefault();
    if (draft.trim() === '' || saving) return;
    setSaving(true);
    setError(null);
    TasksAPI.Add(draft)
      .then((task) => {
        setTasks((list) => [...(list ?? []), task]);
        setDraft('');
      })
      .catch((err) => setError(errorText(err)))
      .finally(() => setSaving(false));
  };

  // gone removes a task that left this view (completed, reopened or deleted).
  const gone = (id: string) => setTasks((list) => (list ?? []).filter((t) => t.id !== id));

  return (
    <section className="tasks">
      <div className="tasks-header">
        <h1 className="page-title">Tasks</h1>
        <div className="segmented" role="group" aria-label="Show">
          <button type="button" className="segmented-item" aria-pressed={view === 'open'} onClick={() => switchView('open')}>
            To do
          </button>
          <button type="button" className="segmented-item" aria-pressed={view === 'completed'} onClick={() => switchView('completed')}>
            Completed
          </button>
        </div>
      </div>

      {view === 'open' && (
        <form className="tasks-new" onSubmit={add}>
          <input
            className="text-input tasks-input"
            aria-label="New task"
            placeholder="Add a task…"
            value={draft}
            onChange={(e) => setDraft(e.target.value)}
          />
          <button type="submit" className="button" disabled={saving || draft.trim() === ''}>
            Add
          </button>
        </form>
      )}

      {error && <p className="tasks-error" role="alert">{error}</p>}

      {tasks === null && <p className="page-subtitle">Loading tasks…</p>}
      {tasks !== null && tasks.length === 0 && !error && (
        <p className="page-subtitle">{view === 'open' ? 'Nothing to do.' : 'No completed tasks.'}</p>
      )}
      {tasks !== null && tasks.length > 0 && (
        <ul className="tasks-list">
          {tasks.map((task) => (
            <TaskItem key={task.id} task={task} onGone={gone} onError={setError} />
          ))}
        </ul>
      )}
    </section>
  )
}

type TaskItemProps = {
  task: Task
  onGone: (id: string) => void
  onError: (message: string | null) => void
}

// TaskItem shows one task with its checkbox. Ticking an open task completes
// it; unticking a completed one reopens it. Completed tasks can be deleted
// for good, after a confirmation.
function TaskItem({ task, onGone, onError }: TaskItemProps) {
  const done = task.completed != null;
  const [busy, setBusy] = useState(false);
  const [confirming, setConfirming] = useState(false);

  // run calls the service and, on success, drops the task from this view.
  const run = (call: Promise<unknown>) => {
    setBusy(true);
    onError(null);
    call
      .then(() => onGone(task.id))
      .catch((err) => {
        onError(errorText(err));
        setBusy(false);
        setConfirming(false);
      });
  };

  const toggle = () => run(done ? TasksAPI.Reopen(task.id) : TasksAPI.Complete(task.id));

  return (
    <li className={`task${done ? ' is-done' : ''}`}>
      <label className="task-check">
        <input
          type="checkbox"
          checked={done}
          disabled={busy}
          onChange={toggle}
          aria-label={done ? `Reopen “${task.text}”` : `Complete “${task.text}”`}
        />
        <span className="task-text">{task.text}</span>
      </label>
      {done && (
        <div className="task-side">
          <time className="task-meta" dateTime={task.completed!}>
            {formatDateTime(task.completed!)}
          </time>
          {confirming ? (
            <>
              <span className="task-confirm">Delete forever?</span>
              <button type="button" className="link-button task-action" onClick={() => run(TasksAPI.Delete(task.id))} disabled={busy}>
                Delete
              </button>
              <button type="button" className="link-button task-action" onClick={() => setConfirming(false)}>
                Keep
              </button>
            </>
          ) : (
            <button type="button" className="link-button task-action" onClick={() => setConfirming(true)}>
              Delete
            </button>
          )}
        </div>
      )}
    </li>
  )
}

export default Tasks
