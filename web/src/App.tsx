import { useEffect, useId, useState } from "react";
import type { FormEvent } from "react";
import { ask, AskError, listSpaces } from "./api";
import type { ApiError, AskResponse, Fetch, Space } from "./api";
import { AnswerView } from "./Answer";
import { FeedbackBar } from "./Feedback";
import { DraftPanel } from "./Draft";

interface Turn {
  question: string;
  answer?: AskResponse;
  error?: ApiError;
}

export function App({ fetcher }: { fetcher?: Fetch }) {
  const [question, setQuestion] = useState("");
  const [turns, setTurns] = useState<Turn[]>([]);
  const [conversation, setConversation] = useState<string>();
  const [busy, setBusy] = useState(false);
  const [spaces, setSpaces] = useState<Space[]>([]);
  const [space, setSpace] = useState(""); // "" is the whole company
  const inputId = useId();
  const spaceId = useId();

  useEffect(() => {
    let live = true;
    listSpaces(fetcher).then((s) => live && setSpaces(s));
    return () => {
      live = false;
    };
  }, [fetcher]);

  async function submit(e: FormEvent) {
    e.preventDefault();
    const q = question.trim();
    if (!q || busy) return;
    setBusy(true);
    try {
      const a = await ask(q, conversation, fetcher, space || undefined);
      setConversation(a.conversation);
      setTurns((t) => [...t, { question: q, answer: a }]);
      setQuestion("");
    } catch (err) {
      const error = err instanceof AskError ? err.error : { code: "EUNAVAIL", message: "Something went wrong.", trace: "", remedy: "Try again." };
      setTurns((t) => [...t, { question: q, error }]);
    } finally {
      setBusy(false);
    }
  }

  return (
    <main>
      <h1>Second Brain</h1>
      <section aria-live="polite" aria-busy={busy} aria-label="Conversation" className="turns">
        {turns.map((t, i) => (
          <div className="turn" key={i}>
            <h2 className="question">{t.question}</h2>
            {t.answer && <AnswerView answer={t.answer} />}
            {t.answer && <FeedbackBar answer={t.answer} fetcher={fetcher} />}
            {t.error && (
              <div className="error" role="alert">
                <p>{t.error.message}</p>
                <p>{t.error.remedy}</p>
                {t.error.trace && <p>Quote trace {t.error.trace} to support.</p>}
              </div>
            )}
          </div>
        ))}
      </section>
      <form onSubmit={submit} className="ask">
        {spaces.length > 0 && (
          <>
            <label htmlFor={spaceId}>Ask in</label>
            <select
              id={spaceId}
              value={space}
              onChange={(e) => {
                setSpace(e.target.value);
                setConversation(undefined); // a new space starts a new conversation
                setTurns([]);
              }}
            >
              <option value="">Whole company</option>
              {spaces.map((s) => (
                <option key={s.name} value={s.name}>
                  {s.name.charAt(0).toUpperCase() + s.name.slice(1)} ({s.label})
                </option>
              ))}
            </select>
          </>
        )}
        <label htmlFor={inputId}>Ask a question</label>
        <textarea id={inputId} value={question} maxLength={4000} rows={3} onChange={(e) => setQuestion(e.target.value)} />
        <button type="submit" disabled={busy || question.trim() === ""}>
          {busy ? "Asking…" : "Ask"}
        </button>
        {conversation && (
          <button type="button" onClick={() => { setConversation(undefined); setTurns([]); }}>
            New conversation
          </button>
        )}
      </form>
      <DraftPanel fetcher={fetcher} />
    </main>
  );
}
