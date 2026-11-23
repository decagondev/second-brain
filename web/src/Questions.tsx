import { useEffect, useId, useState } from "react";
import type { FormEvent } from "react";
import { AskError, addEvalCase, listQuestions, notifySettings, publishNote, setNotifySettings, replyToQuestion, reviewList, sendToStewards, spaceName, stewardQueue } from "./api";
import type { AskResponse, Fetch, MyQuestion, NoteResult, QueueQuestion, ReviewItem } from "./api";
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
      <NotifySettings fetcher={fetcher} />
    </section>
  );
}

const kinds: { id: string; name: string }[] = [
  { id: "replies", name: "Replies to my questions" },
  { id: "questions", name: "New questions for spaces I steward" },
  { id: "digest", name: "What changed in my spaces, weekly" },
  { id: "watch", name: "Changes to my watched questions' sources" },
];

// What the person is told about (sb-ask(3), Notifications; SB-2101, SB-2102).
export function NotifySettings({ fetcher }: { fetcher?: Fetch }) {
  const [off, setOff] = useState<string[]>();
  const [status, setStatus] = useState<string>();
  const base = useId();
  useEffect(() => {
    notifySettings(fetcher).then(setOff, () => setOff(undefined));
  }, [fetcher]);
  if (!off) return null;
  async function toggle(kind: string, on: boolean) {
    const next = on ? off!.filter((k) => k !== kind) : [...off!, kind];
    try {
      setOff(await setNotifySettings(next, fetcher));
      setStatus("Saved.");
    } catch (e) {
      setStatus(msg(e, "Your settings couldn't be saved."));
    }
  }
  return (
    <fieldset className="notify-settings">
      <legend>Tell me in chat about</legend>
      {kinds.map((k) => (
        <div key={k.id}>
          <input type="checkbox" id={`${base}-${k.id}`} checked={!off.includes(k.id)} onChange={(e) => toggle(k.id, e.target.checked)} />{" "}
          <label htmlFor={`${base}-${k.id}`}>{k.name}</label>
        </div>
      ))}
      {status && (
        <p className="feedback-status" role="status">
          {status}
        </p>
      )}
    </fieldset>
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

// Documents to review on the steward's screen (SB-1709): relied-on documents gone stale, or
// marked as wrong citations, each one the steward can open; the rest counted with an owner.
// A note whose readers are wider than its space now (SB-2004): the count, never who, and a
// republish to the members who can still read it, confirmed like any note.
function Narrow({ space, item, fetcher }: { space: string; item: ReviewItem; fetcher?: Fetch }) {
  const [preview, setPreview] = useState<NoteResult>();
  const [done, setDone] = useState(false);
  const [error, setError] = useState<string>();
  const [busy, setBusy] = useState(false);
  const id = item.question!;
  async function step(confirm?: number) {
    setBusy(true);
    setError(undefined);
    try {
      const r = await publishNote(space, id, "", confirm, fetcher);
      if (confirm === undefined) setPreview(r);
      else setDone(true);
    } catch (e) {
      setError(msg(e, "The note wasn't republished. Try again."));
    } finally {
      setBusy(false);
    }
  }
  const n = item.wider!;
  return (
    <div className="narrow">
      <p className="refused">
        {n === 1 ? "1 of this note's readers is" : `${n} of this note's readers are`} no longer a member of the space.
      </p>
      {done ? (
        <p role="status">Republished. It leaves this list once the share is next read.</p>
      ) : preview ? (
        <p>
          Republish to {preview.readers === 1 ? "1 reader" : `${preview.readers} readers`}
          {preview.narrower > 0 ? `; ${preview.narrower} of its sources some of them can't open` : ""}.{" "}
          <button type="button" disabled={busy} onClick={() => step(preview.narrower)}>
            Confirm republish
          </button>
        </p>
      ) : (
        <button type="button" disabled={busy} onClick={() => step()}>
          Republish to members only
        </button>
      )}
      {error && (
        <p className="error" role="alert">
          {error}
        </p>
      )}
    </div>
  );
}

// Add to evals (SB-2204): the steward names the document the reported answer should cite.
function AddToEvals({ space, item, fetcher }: { space: string; item: ReviewItem; fetcher?: Fetch }) {
  const id = useId();
  const [expect, setExpect] = useState("");
  const [done, setDone] = useState("");
  const [error, setError] = useState("");
  const report = item.reports?.[0] ?? "";
  const add = async (e: FormEvent) => {
    e.preventDefault();
    setError("");
    try {
      const c = await addEvalCase(space, report, expect.trim(), fetcher);
      setDone(`Added as eval case ${c.id}: every release now has to cite it for this question.`);
    } catch (err) {
      setError(err instanceof AskError ? err.error.message : "The case couldn't be added.");
    }
  };
  if (done) return <p role="status">{done}</p>;
  return (
    <form onSubmit={add} className="add-eval">
      <label htmlFor={id}>Should have cited (KIND:ID)</label>
      <input id={id} value={expect} onChange={(e) => setExpect(e.target.value)} placeholder="docs:Ops/right.md" />
      <button type="submit" disabled={expect.trim() === ""}>
        Add to evals
      </button>
      {error && <p role="alert">{error}</p>}
    </form>
  );
}

export function ReviewList({ space, fetcher }: { space: string; fetcher?: Fetch }) {
  const [list, setList] = useState<{ items: ReviewItem[]; hidden: { owner?: string; owner_gone?: boolean }[]; personal_data?: number; glossary_flagged?: string[] }>({ items: [], hidden: [] });
  useEffect(() => {
    reviewList(space, fetcher).then(setList);
  }, [space, fetcher]);
  return (
    <section aria-label="Documents to review">
      <h2>Documents to review</h2>
      {list.items.length === 0 && list.hidden.length === 0 && <p>Nothing to review: no relied-on document here is stale or marked wrong.</p>}
      {list.items.length > 0 && (
        <ul className="saved-list">
          {list.items.map((r) => (
            <li key={r.cite}>
              {r.url ? (
                <a href={r.url} target="_blank" rel="noopener noreferrer" aria-label={`${r.title ?? r.cite}, source ${r.cite}, opens in a new tab`}>
                  {r.title ?? r.cite}
                </a>
              ) : (
                <span>{r.title ?? r.cite}</span>
              )}
              <span className="meta">
                {" "}
                · cited {r.cited} times in 30 days{r.age_days !== undefined ? ` · ${r.age_days} days old` : ""}
                {r.flagged > 0 ? ` · marked wrong ${r.flagged} times` : ""}
                {r.owner_gone ? " · owner gone: find it a new owner" : ""}
              </span>
              {r.copies && r.copies.length > 0 && (
                <span className="meta">
                  {" "}
                  · the same text is also at:{" "}
                  {r.copies.map((c, i) => (
                    <span key={c.cite}>
                      {i > 0 ? ", " : ""}
                      {c.url ? (
                        <a href={c.url} target="_blank" rel="noopener noreferrer" aria-label={`Copy: ${c.title ?? c.cite}, source ${c.cite}, opens in a new tab`}>
                          {c.title ?? c.cite}
                        </a>
                      ) : (
                        <span>{c.title ?? c.cite}</span>
                      )}
                    </span>
                  ))}
                </span>
              )}
              {r.wider !== undefined && r.wider > 0 && r.question && <Narrow space={space} item={r} fetcher={fetcher} />}
              {r.reports && r.reports.length > 0 && <AddToEvals space={space} item={r} fetcher={fetcher} />}
            </li>
          ))}
        </ul>
      )}
      {list.hidden.length > 0 && (
        <p className="refused">
          {list.hidden.length === 1 ? "1 more document you can't open needs" : `${list.hidden.length} more documents you can't open need`} review
          {list.hidden.some((h) => h.owner_gone) ? `, ${list.hidden.filter((h) => h.owner_gone).length} of them with no owner` : ""}
          {list.hidden.some((h) => h.owner) ? `; ask their owners: ${Array.from(new Set(list.hidden.flatMap((h) => (h.owner ? [h.owner] : [])))).join(", ")}` : ""}.
        </p>
      )}
      {list.glossary_flagged && list.glossary_flagged.length > 0 && (
        <p className="refused">
          These glossary terms read like instructions to the model, so they aren't used: {list.glossary_flagged.join(", ")}. Propose a plain
          meaning for each.
        </p>
      )}
      {list.personal_data !== undefined && list.personal_data > 0 && (
        <p className="personal">
          {list.personal_data === 1 ? "1 document" : `${list.personal_data} documents`} in this space hold personal data (an IBAN or a card
          number). Answers and snippets show only the last four characters; check that the space is the right place for them.
        </p>
      )}
    </section>
  );
}
