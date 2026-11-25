import { useEffect, useState } from "react";
import { AskError, listSaved, removeSaved, saveQuestion, setRoutine, shareQuestion, watchSaved } from "./api";
import type { Fetch, SavedQuestion } from "./api";
import { errorText, msg, t } from "./i18n";
import { ShareRoutine } from "./Shape";

// A routine's choices (SB-4302): off, each working morning, or one weekday.
const routines: { value: string; name: string }[] = [
  { value: "off", name: msg("Off") },
  { value: "daily", name: msg("Every working morning") },
  { value: "monday", name: msg("Mondays") },
  { value: "tuesday", name: msg("Tuesdays") },
  { value: "wednesday", name: msg("Wednesdays") },
  { value: "thursday", name: msg("Thursdays") },
  { value: "friday", name: msg("Fridays") },
];

// Save and Share on one answered turn (sb-ask(3), Saved and shared questions; SB-1703).
// Only the question is kept or passed on, never the answer.
export function TurnTools({ question, fetcher, space }: { question: string; fetcher?: Fetch; space?: string }) {
  const [status, setStatus] = useState<string>();
  const [link, setLink] = useState<string>();

  async function save() {
    try {
      await saveQuestion(question, fetcher, space);
      setStatus(t("Saved. Find it in the Saved tab."));
    } catch (e) {
      setStatus(e instanceof AskError ? errorText(e.error).message : t("The question couldn't be saved."));
    }
  }

  async function share() {
    try {
      const sh = await shareQuestion(question, fetcher, space);
      setLink(`${window.location.origin}${sh.url}`);
      setStatus(t("Copy this link. Whoever opens it asks the question under their own access."));
    } catch (e) {
      setStatus(e instanceof AskError ? errorText(e.error).message : t("The question couldn't be shared."));
    }
  }

  return (
    <div className="turn-tools" role="group" aria-label={t("Keep or pass on this question")}>
      <button type="button" onClick={save}>
        {t("Save question")}
      </button>
      <button type="button" onClick={share}>
        {t("Share question")}
      </button>
      {link && (
        <label>
          {t("Share link")} <input type="text" readOnly value={link} onFocus={(e) => e.currentTarget.select()} />
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
      setError(e instanceof AskError ? errorText(e.error).message : t("Saved questions are unavailable right now."));
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
      setError(e instanceof AskError ? errorText(e.error).message : t("The question couldn't be removed."));
    }
  }

  async function watch(id: string, on: boolean) {
    try {
      await watchSaved(id, on, fetcher);
      await load();
    } catch (e) {
      setError(e instanceof AskError ? errorText(e.error).message : t("The question couldn't be watched."));
    }
  }

  async function routine(id: string, v: string) {
    try {
      await setRoutine(id, v === "off" || v === "daily" ? v : "weekly", v === "off" || v === "daily" ? undefined : v, fetcher);
      await load();
    } catch (e) {
      setError(e instanceof AskError ? errorText(e.error).message : t("The routine couldn't be set."));
    }
  }

  return (
    <section aria-label={t("Saved questions")} className="saved">
      {error && (
        <p className="error" role="alert">
          {error}
        </p>
      )}
      {items && items.length === 0 && <p className="notice">{t("No saved questions yet. Use Save question under an answer.")}</p>}
      {items && items.length > 0 && (
        <ul className="saved-list">
          {items.map((q) => (
            <li key={q.id}>
              <span className="saved-question">{q.question}</span>{" "}
              <button type="button" onClick={() => onAsk(q.question, q.space)} aria-label={t("Ask again: {question}", { question: q.question })}>
                {t("Ask again")}
              </button>{" "}
              <button type="button" onClick={() => remove(q.id)} aria-label={t("Remove: {question}", { question: q.question })}>
                {t("Remove")}
              </button>{" "}
              <label>
                <input type="checkbox" aria-label={t("Watch: {question}", { question: q.question })} checked={!!q.watch} onChange={(e) => watch(q.id, e.target.checked)} />{" "}
                {t("Watch")}
              </label>{" "}
              <label>
                {t("Routine")}{" "}
                <select
                  aria-label={t("Routine: {question}", { question: q.question })}
                  value={!q.routine ? "off" : q.routine.every === "daily" ? "daily" : (q.routine.day ?? "monday")}
                  onChange={(e) => routine(q.id, e.target.value)}
                >
                  {routines.map((r) => (
                    <option key={r.value} value={r.value}>
                      {t(r.name)}
                    </option>
                  ))}
                </select>
              </label>{" "}
              <ShareRoutine q={q} fetcher={fetcher} onChange={() => void load()} />
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}
