import { useEffect, useState } from "react";
import { AskError, listSaved, removeSaved, saveQuestion, shareQuestion } from "./api";
import type { Fetch, SavedQuestion } from "./api";

// Save and Share on one answered turn (sb-ask(3), Saved and shared questions; SB-1703).
// Only the question is kept or passed on, never the answer.
export function TurnTools({ question, fetcher, space }: { question: string; fetcher?: Fetch; space?: string }) {
  const [status, setStatus] = useState<string>();
  const [link, setLink] = useState<string>();

  async function save() {
    try {
      await saveQuestion(question, fetcher, space);
      setStatus("Saved. Find it in the Saved tab.");
    } catch (e) {
      setStatus(e instanceof AskError ? e.error.message : "The question couldn't be saved.");
    }
  }

  async function share() {
    try {
      const sh = await shareQuestion(question, fetcher, space);
      setLink(`${window.location.origin}${sh.url}`);
      setStatus("Copy this link. Whoever opens it asks the question under their own access.");
    } catch (e) {
      setStatus(e instanceof AskError ? e.error.message : "The question couldn't be shared.");
    }
  }

  return (
    <div className="turn-tools" role="group" aria-label="Keep or pass on this question">
      <button type="button" onClick={save}>
        Save question
      </button>
      <button type="button" onClick={share}>
        Share question
      </button>
      {link && (
        <label>
          Share link <input type="text" readOnly value={link} onFocus={(e) => e.currentTarget.select()} />
        </label>
      )}
      {status && (
        <p className="feedback-status" role="status">
          {status}
        </p>
      )}
    </div>
  );
}

// The Saved tab: the person's own saved questions; opening one asks it again, live.
export function SavedView({ fetcher, onAsk }: { fetcher?: Fetch; onAsk: (question: string, space: string) => void }) {
  const [items, setItems] = useState<SavedQuestion[]>();
  const [error, setError] = useState<string>();

  async function load() {
    try {
      setItems(await listSaved(fetcher));
      setError(undefined);
    } catch (e) {
      setError(e instanceof AskError ? e.error.message : "Saved questions are unavailable right now.");
    }
  }

  useEffect(() => {
    void load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  async function remove(id: string) {
    try {
      await removeSaved(id, fetcher);
      await load();
    } catch (e) {
      setError(e instanceof AskError ? e.error.message : "The question couldn't be removed.");
    }
  }

  return (
    <section aria-label="Saved questions" className="saved">
      {error && (
        <p className="error" role="alert">
          {error}
        </p>
      )}
      {items && items.length === 0 && <p className="notice">No saved questions yet. Use Save question under an answer.</p>}
      {items && items.length > 0 && (
        <ul className="saved-list">
          {items.map((q) => (
            <li key={q.id}>
              <span className="saved-question">{q.question}</span>{" "}
              <button type="button" onClick={() => onAsk(q.question, q.space)} aria-label={`Ask again: ${q.question}`}>
                Ask again
              </button>{" "}
              <button type="button" onClick={() => remove(q.id)} aria-label={`Remove: ${q.question}`}>
                Remove
              </button>
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}
