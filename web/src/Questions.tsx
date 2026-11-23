import { useEffect, useId, useState } from "react";
import type { FormEvent } from "react";
import { AskError, addEvalCase, listQuestions, notifySettings, publishNote, setNotifySettings, replyToQuestion, reviewList, sendToStewards, spaceName, stewardQueue } from "./api";
import type { AskResponse, Fetch, MyQuestion, NoteResult, QueueQuestion, ReviewItem } from "./api";
import { placeOf } from "./Answer";
import { errorText, msg, t, tn } from "./i18n";

const askable = new Set(["no_sources", "no_access", "most_uncited"]);

// A question's state, in words.
const states: Record<string, string> = { open: msg("open"), answered: msg("answered") };

const errMsg = (e: unknown, fallback: string) => (e instanceof AskError ? errorText(e.error).message : fallback);

// Ask the stewards (sb-ask(3), Questions to the stewards; SB-1704): offered only when
// nothing answered, and it shows exactly what will be sent before sending it.
export function AskStewards({ question, answer, space, fetcher }: { question: string; answer: AskResponse; space?: string; fetcher?: Fetch }) {
  const [open, setOpen] = useState(false);
  const [status, setStatus] = useState<string>();
  if (!answer.notice || !askable.has(answer.notice)) return null;
  const who = space ? t("the stewards of {space}", { space: space.charAt(0).toUpperCase() + space.slice(1) }) : t("the people who answer company-wide questions");

  async function send() {
    try {
      await sendToStewards(question, answer.trace, fetcher, space);
      setOpen(false);
      setStatus(t("Sent to {who}. Their reply will be in the Questions tab.", { who }));
    } catch (e) {
      setStatus(errMsg(e, t("The question couldn't be sent.")));
    }
  }

  return (
    <div className="ask-stewards">
      {!open && !status && (
        <button type="button" onClick={() => setOpen(true)}>
          {t("Ask {who}", { who })}
        </button>
      )}
      {open && (
        <div role="group" aria-label={t("What will be sent")}>
          <p>{t("Only this text will be sent to {who}, without your answer or its sources:", { who })}</p>
          <blockquote className="snippet">{question}</blockquote>
          <button type="button" onClick={send}>
            {t("Send question")}
          </button>{" "}
          <button type="button" onClick={() => setOpen(false)}>
            {t("Cancel")}
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
    listQuestions(fetcher).then(setItems, (e) => setError(errMsg(e, t("Your questions are unavailable right now."))));
  }, [fetcher]);
  return (
    <section aria-label={t("Questions to stewards")} className="questions">
      {error && (
        <p className="error" role="alert">
          {error}
        </p>
      )}
      {items && items.length === 0 && <p className="notice">{t("No questions yet. When nothing answers, you can ask the stewards.")}</p>}
      {items && items.length > 0 && (
        <ul className="saved-list">
          {items.map((q) => (
            <li key={q.id}>
              <p className="question">
                {q.question} <span className="meta">({spaceName(q.space) || t("company-wide")}, {t(states[q.state] ?? q.state)})</span>
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
                          <a key={c.cite} className="chip" href={c.url} target="_blank" rel="noopener noreferrer" aria-label={t("Source {cite}, {name}, opens in a new tab", { cite: c.cite, name })}>
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
                    <p className="refused">
                      {tn(q.reply.dropped, "{n} source was not shown because you can't open it.", "{n} sources were not shown because you can't open them.")}
                    </p>
                  )}
                  {q.reply.note && <p className="meta">{t("Published as a note, so the next person who asks gets this answer with its cite.")}</p>}
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
  { id: "replies", name: msg("Replies to my questions") },
  { id: "questions", name: msg("New questions for spaces I steward") },
  { id: "digest", name: msg("What changed in my spaces, weekly") },
  { id: "watch", name: msg("Changes to my watched questions' sources") },
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
      setStatus(t("Saved."));
    } catch (e) {
      setStatus(errMsg(e, t("Your settings couldn't be saved.")));
    }
  }
  return (
    <fieldset className="notify-settings">
      <legend>{t("Tell me in chat about")}</legend>
      {kinds.map((k) => (
        <div key={k.id}>
          <input type="checkbox" id={`${base}-${k.id}`} checked={!off.includes(k.id)} onChange={(e) => toggle(k.id, e.target.checked)} />{" "}
          <label htmlFor={`${base}-${k.id}`}>{t(k.name)}</label>
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
      setStatus(t("Replied. The asker sees each cite only if they can open it."));
      await load();
    } catch (e) {
      setStatus(errMsg(e, t("The reply couldn't be sent.")));
    }
  }

  async function note(id: string) {
    try {
      const p = preview[id];
      const r = await publishNote(space, id, title[id] ?? "", p?.narrower, fetcher);
      if (!p) {
        setPreview((x) => ({ ...x, [id]: r }));
        setStatus(t("The note will have {readers} readers. {narrower} of its cites some of them can't open. Publish to confirm.", { readers: r.readers, narrower: r.narrower }));
      } else {
        setPreview((x) => {
          const y = { ...x };
          delete y[id];
          return y;
        });
        setStatus(r.state === "Ran" ? t("Published as {path}.", { path: r.path ?? "" }) : t("The note didn't run ({state}).", { state: r.state ?? "" }));
        await load();
      }
    } catch (e) {
      setStatus(errMsg(e, t("The note couldn't be published.")));
    }
  }

  return (
    <section aria-label={t("Questions")}>
      <h2>{t("Questions")}</h2>
      {items.length === 0 ? (
        <p>{t("No questions for this space.")}</p>
      ) : (
        <ul className="saved-list">
          {items.map((q, i) => (
            <li key={q.id}>
              <p className="question">{q.question}</p>
              {q.state === "open" ? (
                <div className="decide">
                  <label htmlFor={`${ids}-r${i}`}>{t("Reply to: {question}", { question: q.question })}</label>
                  <textarea id={`${ids}-r${i}`} rows={3} maxLength={4000} value={text[q.id] ?? ""} onChange={(e) => setText((x) => ({ ...x, [q.id]: e.target.value }))} />
                  <label htmlFor={`${ids}-c${i}`}>{t("Cites, from your own search (optional)")}</label>
                  <input id={`${ids}-c${i}`} type="text" value={cites[q.id] ?? ""} onChange={(e) => setCites((x) => ({ ...x, [q.id]: e.target.value }))} />
                  <button type="button" onClick={() => reply(q.id)}>
                    {t("Send reply")}
                  </button>
                </div>
              ) : (
                <div className="decide">
                  <p>{q.reply?.text}</p>
                  {q.reply?.note ? (
                    <p className="meta">{t("Published as a note.")}</p>
                  ) : (
                    <>
                      <label htmlFor={`${ids}-t${i}`}>{t("Note title for: {question}", { question: q.question })}</label>
                      <input id={`${ids}-t${i}`} type="text" maxLength={200} value={title[q.id] ?? ""} onChange={(e) => setTitle((x) => ({ ...x, [q.id]: e.target.value }))} />
                      <button type="button" onClick={() => note(q.id)}>
                        {preview[q.id] ? t("Publish, accepting {n} narrower cites", { n: preview[q.id]!.narrower }) : t("Publish as note")}
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
      setError(errMsg(e, t("The note wasn't republished. Try again.")));
    } finally {
      setBusy(false);
    }
  }
  const n = item.wider!;
  return (
    <div className="narrow">
      <p className="refused">
        {tn(n, "{n} of this note's readers is no longer a member of the space.", "{n} of this note's readers are no longer a member of the space.")}
      </p>
      {done ? (
        <p role="status">{t("Republished. It leaves this list once the share is next read.")}</p>
      ) : preview ? (
        <p>
          {preview.narrower > 0
            ? tn(preview.readers, "Republish to {n} reader; {narrower} of its sources some of them can't open.", "Republish to {n} readers; {narrower} of its sources some of them can't open.", {
                narrower: preview.narrower,
              })
            : tn(preview.readers, "Republish to {n} reader.", "Republish to {n} readers.")}{" "}
          <button type="button" disabled={busy} onClick={() => step(preview.narrower)}>
            {t("Confirm republish")}
          </button>
        </p>
      ) : (
        <button type="button" disabled={busy} onClick={() => step()}>
          {t("Republish to members only")}
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
      setDone(t("Added as eval case {id}: every release now has to cite it for this question.", { id: c.id }));
    } catch (err) {
      setError(errMsg(err, t("The case couldn't be added.")));
    }
  };
  if (done) return <p role="status">{done}</p>;
  return (
    <form onSubmit={add} className="add-eval">
      <label htmlFor={id}>{t("Should have cited ({form})", { form: "KIND:ID" })}</label>
      <input id={id} value={expect} onChange={(e) => setExpect(e.target.value)} placeholder="docs:Ops/right.md" />
      <button type="submit" disabled={expect.trim() === ""}>
        {t("Add to evals")}
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
    <section aria-label={t("Documents to review")}>
      <h2>{t("Documents to review")}</h2>
      {list.items.length === 0 && list.hidden.length === 0 && <p>{t("Nothing to review: no relied-on document here is stale or marked wrong.")}</p>}
      {list.items.length > 0 && (
        <ul className="saved-list">
          {list.items.map((r) => (
            <li key={r.cite}>
              {r.url ? (
                <a href={r.url} target="_blank" rel="noopener noreferrer" aria-label={t("{name}, source {cite}, opens in a new tab", { name: r.title ?? r.cite, cite: r.cite })}>
                  {r.title ?? r.cite}
                </a>
              ) : (
                <span>{r.title ?? r.cite}</span>
              )}
              <span className="meta">
                {" "}
                · {tn(r.cited, "cited {n} time in 30 days", "cited {n} times in 30 days")}
                {r.age_days !== undefined ? ` · ${tn(r.age_days, "{n} day old", "{n} days old")}` : ""}
                {r.flagged > 0 ? ` · ${tn(r.flagged, "marked wrong {n} time", "marked wrong {n} times")}` : ""}
                {r.owner_gone ? ` · ${t("owner gone: find it a new owner")}` : ""}
              </span>
              {r.copies && r.copies.length > 0 && (
                <span className="meta">
                  {" "}
                  · {t("the same text is also at:")}{" "}
                  {r.copies.map((c, i) => (
                    <span key={c.cite}>
                      {i > 0 ? ", " : ""}
                      {c.url ? (
                        <a href={c.url} target="_blank" rel="noopener noreferrer" aria-label={t("Copy: {name}, source {cite}, opens in a new tab", { name: c.title ?? c.cite, cite: c.cite })}>
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
          {tn(list.hidden.length, "{n} more document you can't open needs review", "{n} more documents you can't open need review")}
          {list.hidden.some((h) => h.owner_gone) ? `, ${t("{n} of them with no owner", { n: list.hidden.filter((h) => h.owner_gone).length })}` : ""}
          {list.hidden.some((h) => h.owner)
            ? `; ${t("ask their owners: {owners}", { owners: Array.from(new Set(list.hidden.flatMap((h) => (h.owner ? [h.owner] : [])))).join(", ") })}`
            : ""}
          .
        </p>
      )}
      {list.glossary_flagged && list.glossary_flagged.length > 0 && (
        <p className="refused">
          {t("These glossary terms read like instructions to the model, so they aren't used: {terms}. Propose a plain meaning for each.", {
            terms: list.glossary_flagged.join(", "),
          })}
        </p>
      )}
      {list.personal_data !== undefined && list.personal_data > 0 && (
        <p className="personal">
          {tn(
            list.personal_data,
            "{n} document in this space holds personal data (an IBAN, a card number or a national ID number). Answers and snippets show only the last four characters; check that the space is the right place for it.",
            "{n} documents in this space hold personal data (an IBAN, a card number or a national ID number). Answers and snippets show only the last four characters; check that the space is the right place for them.",
          )}
        </p>
      )}
    </section>
  );
}
