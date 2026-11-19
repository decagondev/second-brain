import { useId, useState } from "react";
import type { FormEvent } from "react";
import { ask, AskError } from "./api";
import type { ApiError, AskResponse, Fetch } from "./api";
import { AnswerView } from "./Answer";
import { FeedbackBar } from "./Feedback";

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
  const inputId = useId();

  async function submit(e: FormEvent) {
    e.preventDefault();
    const q = question.trim();
    if (!q || busy) return;
    setBusy(true);
    try {
      const a = await ask(q, conversation, fetcher);
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
    </main>
  );
}
