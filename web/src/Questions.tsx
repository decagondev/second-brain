import { useEffect, useId, useState } from "react";
import { AskError, listQuestions, publishNote, replyToQuestion, sendToStewards, spaceName, stewardQueue } from "./api";
import type { AskResponse, Fetch, MyQuestion, NoteResult, QueueQuestion } from "./api";
import { placeOf } from "./Answer";

const askable = new Set(["no_sources", "no_access", "most_uncited"]);

const msg = (e: unknown, fallback: string) => (e instanceof AskError ? e.error.message : fallback);

// Ask the stewards (sb-ask(3), Questions to the stewards; SB-1704): offered only when
// nothing answered, and it shows exactly what will be sent before sending it.
export function AskStewards({ question, answer, space, fetcher }: { question: string; answer: AskResponse; space?: string; fetcher?: Fetch }) {
  const [open, setOpen] = useState(false);
  const [status, setStatus] = useState<string>();
  if (!answer.notice || !askable.has(answer.notice)) return null;
  const who = space ? `the stewards of ${space.charAt(0).toUpperCase() + space.slice(1)}` : "the people who answer company-wide questions";

  async function send() {
    try {
      await sendToStewards(question, answer.trace, fetcher, space);
      setOpen(false);
      setStatus(`Sent to ${who}. Their reply will be in the Questions tab.`);
    } catch (e) {
      setStatus(msg(e, "The question couldn't be sent."));
    }
  }

  return (
    <div className="ask-stewards">
      {!open && !status && (
        <button type="button" onClick={() => setOpen(true)}>
          Ask {who}
        </button>
      )}
      {open && (
        <div role="group" aria-label="What will be sent">
          <p>Only this text will be sent to {who}, without your answer or its sources:</p>
          <blockquote className="snippet">{question}</blockquote>
          <button type="button" onClick={send}>
            Send question
          </button>{" "}
          <button type="button" onClick={() => setOpen(false)}>
            Cancel
          </button>
        </div>
      )}
      {status && (
        <p className="feedback-status" role="status">
          {status}
        </p>
      )}
    </div>
  );
}

// The Questions tab: the asker's own questions and the stewards' replies, each cite one
// they can open now.
export function QuestionsView({ fetcher }: { fetcher?: Fetch }) {
  const [items, setItems] = useState<MyQuestion[]>();
  const [error, setError] = useState<string>();
  useEffect(() => {
    listQuestions(fetcher).then(setItems, (e) => setError(msg(e, "Your questions are unavailable right now.")));
  }, [fetcher]);
  return (
    <section aria-label="Questions to stewards" className="questions">
      {error && (
        <p className="error" role="alert">
          {error}
        </p>
      )}
      {items && items.length === 0 && <p className="notice">No questions yet. When nothing answers, you can ask the stewards.</p>}
      {items && items.length > 0 && (
        <ul className="saved-list">
          {items.map((q) => (
            <li key={q.id}>
              <p className="question">
                {q.question} <span className="meta">({spaceName(q.space) || "company-wide"}, {q.state})</span>
              </p>
              {q.reply && (
                <div className="reply">
                  <p>{q.reply.text}</p>
                  {q.reply.cites.length > 0 && (
                    <span className="chips">
                      {q.reply.cites.map((c) => {
                        const place = placeOf(c);
                        const name = c.title ?? c.cite;
                        return c.url ? (
                          <a key={c.cite} className="chip" href={c.url} target="_blank" rel="noopener noreferrer" aria-label={`Source ${c.cite}, ${name}, opens in a new tab`}>
                            {name}
                            {place && <span className="place"> {place}</span>}
                          </a>
                        ) : (
                          <span key={c.cite} className="chip">
                            {name}
                          </span>
                        );
                      })}
                    </span>
                  )}
                  {q.reply.dropped > 0 && (
                    <p className="refused">{q.reply.dropped === 1 ? "1 source was" : `${q.reply.dropped} sources were`} not shown because you can't open them.</p>
                  )}
                  {q.reply.note && <p className="meta">Published as a note, so the next person who asks gets this answer with its cite.</p>}
                </div>
              )}
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}

// The steward's queue on their screen (sb-spaces(7); SB-1704, SB-1705).
export function StewardQuestions({ space, fetcher }: { space: string; fetcher?: Fetch }) {
  const [items, setItems] = useState<QueueQuestion[]>([]);
  const [status, setStatus] = useState<string>();
  const [text, setText] = useState<Record<string, string>>({});
  const [cites, setCites] = useState<Record<string, string>>({});
  const [title, setTitle] = useState<Record<string, string>>({});
  const [preview, setPreview] = useState<Record<string, NoteResult>>({});
  const ids = useId();

  async function load() {
    try {
      setItems(await stewardQueue(space, fetcher));
    } catch {
      setItems([]); // not a steward here, or an older cell: the section stays empty
    }
  }
  useEffect(() => {
    void load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [space]);

  async function reply(id: string) {
    try {
      const cs = (cites[id] ?? "").split(/[\s,]+/).filter(Boolean);
      await replyToQuestion(space, id, text[id] ?? "", cs, fetcher);
      setStatus("Replied. The asker sees each cite only if they can open it.");
      await load();
    } catch (e) {
      setStatus(msg(e, "The reply couldn't be sent."));
    }
  }

  async function note(id: string) {
    try {
      const p = preview[id];
      const r = await publishNote(space, id, title[id] ?? "", p?.narrower, fetcher);
      if (!p) {
        setPreview((x) => ({ ...x, [id]: r }));
        setStatus(`The note will have ${r.readers} readers. ${r.narrower} of its cites some of them can't open. Publish to confirm.`);
      } else {
        setPreview((x) => {
          const y = { ...x };
          delete y[id];
          return y;
        });
        setStatus(r.state === "Ran" ? `Published as ${r.path}.` : `The note didn't run (${r.state}).`);
        await load();
      }
    } catch (e) {
      setStatus(msg(e, "The note couldn't be published."));
    }
  }

  return (
    <section aria-label="Questions">
      <h2>Questions</h2>
      {items.length === 0 ? (
        <p>No questions for this space.</p>
      ) : (
        <ul className="saved-list">
          {items.map((q, i) => (
            <li key={q.id}>
              <p className="question">{q.question}</p>
              {q.state === "open" ? (
                <div className="decide">
                  <label htmlFor={`${ids}-r${i}`}>Reply to: {q.question}</label>
                  <textarea id={`${ids}-r${i}`} rows={3} maxLength={4000} value={text[q.id] ?? ""} onChange={(e) => setText((x) => ({ ...x, [q.id]: e.target.value }))} />
                  <label htmlFor={`${ids}-c${i}`}>Cites, from your own search (optional)</label>
                  <input id={`${ids}-c${i}`} type="text" value={cites[q.id] ?? ""} onChange={(e) => setCites((x) => ({ ...x, [q.id]: e.target.value }))} />
                  <button type="button" onClick={() => reply(q.id)}>
                    Send reply
                  </button>
                </div>
              ) : (
                <div className="decide">
                  <p>{q.reply?.text}</p>
                  {q.reply?.note ? (
                    <p className="meta">Published as a note.</p>
                  ) : (
                    <>
                      <label htmlFor={`${ids}-t${i}`}>Note title for: {q.question}</label>
                      <input id={`${ids}-t${i}`} type="text" maxLength={200} value={title[q.id] ?? ""} onChange={(e) => setTitle((x) => ({ ...x, [q.id]: e.target.value }))} />
                      <button type="button" onClick={() => note(q.id)}>
                        {preview[q.id] ? `Publish, accepting ${preview[q.id]!.narrower} narrower cites` : "Publish as note"}
                      </button>
                    </>
                  )}
                </div>
              )}
            </li>
          ))}
        </ul>
      )}
      {status && <p role="status">{status}</p>}
    </section>
  );
}
