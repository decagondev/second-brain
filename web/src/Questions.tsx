import { useEffect, useId, useState } from "react";
import type { FormEvent } from "react";
import { AskError, addEvalCase, claimQuestion, deleteTemplate, listQuestions, markReviewed, replyTemplates, restoreDocument, retireDocument, retiredList, uncitedList, saveTemplate, notifySettings, testNotification, publishNote, setNotifySettings, replyToQuestion, reviewList, sendToStewards, spaceName, stewardQueue } from "./api";
import type { AskResponse, Fetch, MyQuestion, NoteResult, NotifyPrefs, QueueQuestion, ReplyTemplate, Retired, ReviewItem, SameNameGroup, SearchResult } from "./api";
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
  { id: "tokens", name: msg("My API tokens about to expire") },
];

// What the person is told about (sb-ask(3), Notifications; SB-2101, SB-2102).
export function NotifySettings({ fetcher }: { fetcher?: Fetch }) {
  const [prefs, setPrefs] = useState<NotifyPrefs>();
  const [days, setDays] = useState("7");
  const [status, setStatus] = useState<string>();
  const base = useId();
  useEffect(() => {
    notifySettings(fetcher).then(setPrefs, () => setPrefs(undefined));
  }, [fetcher]);
  if (!prefs) return null;
  const off = prefs.off;
  async function save(next: string[], pauseDays?: number) {
    try {
      setPrefs(await setNotifySettings(next, fetcher, pauseDays));
      setStatus(t("Saved."));
    } catch (e) {
      setStatus(errMsg(e, t("Your settings couldn't be saved.")));
    }
  }
  async function test() {
    try {
      const r = await testNotification(fetcher);
      setStatus(r.sent ? t("Sent by {via}. If nothing arrives, tell your administrator.", { via: r.via }) : t("{via} doesn't know you by your email. Tell your administrator.", { via: r.via }));
    } catch (e) {
      setStatus(errMsg(e, t("The test couldn't be sent.")));
    }
  }
  const paused = prefs.paused_until && new Date(prefs.paused_until) > new Date() ? new Date(prefs.paused_until) : undefined;
  return (
    <fieldset className="notify-settings">
      <legend>{t("Tell me in chat about")}</legend>
      {kinds.map((k) => (
        <div key={k.id}>
          <input
            type="checkbox"
            id={`${base}-${k.id}`}
            checked={!off.includes(k.id)}
            onChange={(e) => save(e.target.checked ? off.filter((x) => x !== k.id) : [...off, k.id])}
          />{" "}
          <label htmlFor={`${base}-${k.id}`}>{t(k.name)}</label>
        </div>
      ))}
      {paused ? (
        <p>
          {t("Paused until {date}: you're told nothing, and the other stewards still are.", { date: paused.toLocaleDateString() })}{" "}
          <button type="button" onClick={() => save(off, 0)}>
            {t("Resume")}
          </button>
        </p>
      ) : (
        <p>
          <label htmlFor={`${base}-pause`}>{t("Pause for (days)")}</label>{" "}
          <input id={`${base}-pause`} type="number" min={1} max={30} value={days} onChange={(e) => setDays(e.target.value)} />{" "}
          <button type="button" disabled={!(Number(days) >= 1 && Number(days) <= 30)} onClick={() => save(off, Math.floor(Number(days)))}>
            {t("Pause")}
          </button>
        </p>
      )}
      <p>
        <button type="button" onClick={test}>
          {t("Send me a test")}
        </button>
      </p>
      {status && (
        <p className="feedback-status" role="status">
          {status}
        </p>
      )}
    </fieldset>
  );
}

// Count is a heading's count of what waits (SB-2504): a number to see, words to hear.
function Count({ n }: { n: number }) {
  return (
    <>
      {" "}
      <span className="count" aria-hidden="true">
        ({n})
      </span>
      <span className="visually-hidden">{tn(n, "{n} waiting", "{n} waiting")}</span>
    </>
  );
}

// Every five minutes, while the screen is open (SB-2504).
const Refresh = 5 * 60 * 1000;

// The steward's queue on their screen (sb-spaces(7); SB-1704, SB-1705).
export function StewardQuestions({ space, fetcher }: { space: string; fetcher?: Fetch }) {
  const [items, setItems] = useState<QueueQuestion[]>([]);
  const [status, setStatus] = useState<string>();
  const [text, setText] = useState<Record<string, string>>({});
  const [cites, setCites] = useState<Record<string, string>>({});
  const [title, setTitle] = useState<Record<string, string>>({});
  const [preview, setPreview] = useState<Record<string, NoteResult>>({});
  const [templates, setTemplates] = useState<ReplyTemplate[]>([]);
  const [steward, setSteward] = useState(false);
  const ids = useId();

  async function load() {
    setTemplates(await replyTemplates(space, fetcher));
    try {
      setItems(await stewardQueue(space, fetcher));
      setSteward(true);
    } catch {
      setSteward(false);
      setItems([]); // not a steward here, or an older cell: the section stays empty
    }
  }
  useEffect(() => {
    void load();
    const timer = window.setInterval(() => void load(), Refresh);
    return () => window.clearInterval(timer);
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

  async function claim(id: string, release: boolean) {
    try {
      const c = await claimQuestion(space, id, release, fetcher);
      setItems((xs) => xs.map((q) => (q.id === id ? { ...q, claim: c ?? undefined } : q)));
      if (c && !c.mine) setStatus(t("{who} is already answering this.", { who: c.by }));
    } catch (e) {
      setStatus(errMsg(e, t("The question couldn't be claimed.")));
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
      <h2>
        {t("Questions")}
        {steward && <Count n={items.filter((q) => q.state === "open").length} />}
      </h2>
      {items.length === 0 ? (
        <p>{t("No questions for this space.")}</p>
      ) : (
        <ul className="saved-list">
          {items.map((q, i) => (
            <li key={q.id}>
              <p className="question">{q.question}</p>
              {q.state === "open" ? (
                <div className="decide">
                  <p className="meta">
                    {q.claim && !q.claim.mine
                      ? t("{who} is answering this, until {time}.", { who: q.claim.by, time: new Date(q.claim.until).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" }) })
                      : q.claim?.mine
                        ? t("You said you'd answer this.")
                        : ""}{" "}
                    {q.claim?.mine ? (
                      <button type="button" onClick={() => claim(q.id, true)}>
                        {t("Let it go")}
                      </button>
                    ) : (
                      !q.claim && (
                        <button type="button" onClick={() => claim(q.id, false)}>
                          {t("I'll answer this")}
                        </button>
                      )
                    )}
                  </p>
                  {templates.length > 0 && (
                    <>
                      <label htmlFor={`${ids}-u${i}`}>{t("Use a template")}</label>
                      <select
                        id={`${ids}-u${i}`}
                        value=""
                        onChange={(e) => {
                          const tm = templates.find((x) => x.id === e.target.value);
                          if (tm) setText((x) => ({ ...x, [q.id]: tm.text }));
                        }}
                      >
                        <option value="">{t("Choose one")}</option>
                        {templates.map((tm) => (
                          <option key={tm.id} value={tm.id}>
                            {tm.name}
                          </option>
                        ))}
                      </select>
                    </>
                  )}
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
      {steward && <Templates space={space} items={templates} onChange={setTemplates} fetcher={fetcher} />}
      {status && <p role="status">{status}</p>}
    </section>
  );
}

// A space's reply templates (SB-2503): add, edit and delete, for its stewards only.
function Templates({ space, items, onChange, fetcher }: { space: string; items: ReplyTemplate[]; onChange: (ts: ReplyTemplate[]) => void; fetcher?: Fetch }) {
  const ids = useId();
  const [editing, setEditing] = useState<ReplyTemplate>({ id: "", name: "", text: "" });
  const [error, setError] = useState("");
  async function save(e: FormEvent) {
    e.preventDefault();
    setError("");
    try {
      await saveTemplate(space, editing.id, editing.name, editing.text, fetcher);
      onChange(await replyTemplates(space, fetcher));
      setEditing({ id: "", name: "", text: "" });
    } catch (err) {
      setError(errMsg(err, t("The template couldn't be saved.")));
    }
  }
  async function remove(id: string) {
    setError("");
    try {
      await deleteTemplate(space, id, fetcher);
      onChange(items.filter((x) => x.id !== id));
    } catch (err) {
      setError(errMsg(err, t("The template couldn't be deleted.")));
    }
  }
  return (
    <details className="templates">
      <summary>{t("Reply templates")}</summary>
      {items.length > 0 && (
        <ul>
          {items.map((tm) => (
            <li key={tm.id}>
              <span>{tm.name}</span>{" "}
              <button type="button" aria-label={t("Edit {name}", { name: tm.name })} onClick={() => setEditing(tm)}>
                {t("Edit")}
              </button>{" "}
              <button type="button" aria-label={t("Delete {name}", { name: tm.name })} onClick={() => remove(tm.id)}>
                {t("Delete")}
              </button>
            </li>
          ))}
        </ul>
      )}
      <form onSubmit={save}>
        <label htmlFor={`${ids}-n`}>{t("Template name")}</label>
        <input id={`${ids}-n`} maxLength={60} value={editing.name} onChange={(e) => setEditing({ ...editing, name: e.target.value })} />
        <label htmlFor={`${ids}-x`}>{t("Template text")}</label>
        <textarea id={`${ids}-x`} rows={3} maxLength={2000} value={editing.text} onChange={(e) => setEditing({ ...editing, text: e.target.value })} />
        <button type="submit" disabled={editing.name.trim() === "" || editing.text.trim() === ""}>
          {editing.id ? t("Save template") : t("Add template")}
        </button>
      </form>
      {error && <p role="alert">{error}</p>}
    </details>
  );
}

// Reviewed, still right (SB-2501): off the list until a new version, its age or a new report.
function MarkReviewed({ space, item, onDone, fetcher }: { space: string; item: ReviewItem; onDone: () => void; fetcher?: Fetch }) {
  const [error, setError] = useState("");
  async function mark() {
    setError("");
    try {
      await markReviewed(space, item.cite, fetcher);
      onDone();
    } catch (err) {
      setError(errMsg(err, t("The document couldn't be marked.")));
    }
  }
  return (
    <>
      {" "}
      <button type="button" aria-label={t("Reviewed, still right: {name}", { name: item.title ?? item.cite })} onClick={mark}>
        {t("Reviewed, still right")}
      </button>
      {error && <span role="alert"> {error}</span>}
    </>
  );
}

// Retire from answers (SB-2801): a reason, and the document that replaces it if any.
function Retire({ space, cite, name, onDone, fetcher }: { space: string; cite: string; name: string; onDone: () => void; fetcher?: Fetch }) {
  const reasonId = useId();
  const byId = useId();
  const [open, setOpen] = useState(false);
  const [reason, setReason] = useState("");
  const [by, setBy] = useState("");
  const [error, setError] = useState("");
  if (!open)
    return (
      <>
        {" "}
        <button type="button" aria-label={t("Retire from answers: {name}", { name })} onClick={() => setOpen(true)}>
          {t("Retire from answers")}
        </button>
      </>
    );
  const send = async (e: FormEvent) => {
    e.preventDefault();
    setError("");
    try {
      await retireDocument(space, cite, reason.trim(), by.trim(), fetcher);
      onDone();
    } catch (err) {
      setError(errMsg(err, t("The document couldn't be retired.")));
    }
  };
  return (
    <form onSubmit={send} className="retire">
      <label htmlFor={reasonId}>{t("Why it should leave this space's answers")}</label>
      <input id={reasonId} value={reason} maxLength={200} onChange={(e) => setReason(e.target.value)} />
      <label htmlFor={byId}>{t("Replaced by ({form}, optional)", { form: "KIND:ID" })}</label>
      <input id={byId} value={by} onChange={(e) => setBy(e.target.value)} placeholder="docs:Ops/right.md" />
      <button type="submit" disabled={reason.trim() === ""}>
        {t("Retire")}
      </button>
      <button type="button" onClick={() => setOpen(false)}>
        {t("Cancel")}
      </button>
      {error && <p role="alert">{error}</p>}
    </form>
  );
}

// The space's retired documents (SB-2801), each one brought back with a click.
function RetiredDocs({ space, fetcher, bump }: { space: string; fetcher?: Fetch; bump: number }) {
  const [list, setList] = useState<{ items: Retired[]; hidden: number }>({ items: [], hidden: 0 });
  const [error, setError] = useState("");
  useEffect(() => {
    void retiredList(space, fetcher).then(setList);
  }, [space, fetcher, bump]);
  if (list.items.length === 0 && list.hidden === 0) return null;
  const restore = async (doc: string) => {
    setError("");
    try {
      await restoreDocument(space, doc, fetcher);
      setList((l) => ({ ...l, items: l.items.filter((x) => x.document !== doc) }));
    } catch (err) {
      setError(errMsg(err, t("The document couldn't be brought back.")));
    }
  };
  return (
    <section aria-label={t("Retired from answers")}>
      <h3>
        {t("Retired from answers")}
        <Count n={list.items.length + list.hidden} />
      </h3>
      <ul className="saved-list">
        {list.items.map((r) => (
          <li key={r.document}>
            <span>{r.title ?? r.document}</span>
            <span className="meta">
              {" "}
              · {r.reason}
              {r.replaced_by ? ` · ${t("replaced by {cite}", { cite: r.replaced_by })}` : ""} · {r.by} · {r.at.slice(0, 10)}
            </span>{" "}
            <button type="button" aria-label={t("Bring back: {name}", { name: r.title ?? r.document })} onClick={() => void restore(r.document)}>
              {t("Bring back")}
            </button>
          </li>
        ))}
      </ul>
      {list.hidden > 0 && <p className="refused">{tn(list.hidden, "{n} more retired document you can't open", "{n} more retired documents you can't open")}</p>}
      {error && <p role="alert">{error}</p>}
    </section>
  );
}

function DocLink({ r }: { r: SearchResult }) {
  return r.url ? (
    <a href={r.url} target="_blank" rel="noopener noreferrer" aria-label={t("{name}, source {cite}, opens in a new tab", { name: r.title ?? r.cite, cite: r.cite })}>
      {r.title ?? r.cite}
    </a>
  ) : (
    <span>{r.title ?? r.cite}</span>
  );
}

// Same name, different text (SB-2802): one group per title, each text once.
function SameNames({ groups }: { groups: SameNameGroup[] }) {
  if (groups.length === 0) return null;
  return (
    <section aria-label={t("Same name, different text")}>
      <h3>{t("Same name, different text")}</h3>
      <ul className="saved-list">
        {groups.map((g) => (
          <li key={g.title}>
            <strong>{g.title}</strong>:{" "}
            {g.items.map((r, i) => (
              <span key={r.cite}>
                {i > 0 ? ", " : ""}
                <DocLink r={r} />
              </span>
            ))}
            {g.hidden > 0 ? ` · ${tn(g.hidden, "{n} more you can't open", "{n} more you can't open")}` : ""}
          </li>
        ))}
      </ul>
    </section>
  );
}

// Never cited (SB-2803): loaded only when the steward asks.
function NeverCited({ space, fetcher, onRetired }: { space: string; fetcher?: Fetch; onRetired: () => void }) {
  const [list, setList] = useState<{ items: SearchResult[]; hidden: { owner?: string }[] }>();
  if (!list)
    return (
      <button type="button" onClick={() => void uncitedList(space, fetcher).then(setList)}>
        {t("Show never cited")}
      </button>
    );
  return (
    <section aria-label={t("Never cited")}>
      <h3>
        {t("Never cited")}
        <Count n={list.items.length + list.hidden.length} />
      </h3>
      {list.items.length === 0 && list.hidden.length === 0 && <p>{t("Every document here older than 90 days was cited in the last 180 days.")}</p>}
      <ul className="saved-list">
        {list.items.map((r) => (
          <li key={r.cite}>
            <DocLink r={r} />
            {r.age_days !== undefined ? <span className="meta"> · {tn(r.age_days, "{n} day old", "{n} days old")}</span> : null}
            <Retire
              space={space}
              cite={r.cite}
              name={r.title ?? r.cite}
              fetcher={fetcher}
              onDone={() => {
                setList((l) => l && { ...l, items: l.items.filter((x) => x.cite !== r.cite) });
                onRetired();
              }}
            />
          </li>
        ))}
      </ul>
      {list.hidden.length > 0 && (
        <p className="refused">
          {tn(list.hidden.length, "{n} more document you can't open was never cited", "{n} more documents you can't open were never cited")}
          {list.hidden.some((h) => h.owner)
            ? `; ${t("ask their owners: {owners}", { owners: Array.from(new Set(list.hidden.flatMap((h) => (h.owner ? [h.owner] : [])))).join(", ") })}`
            : ""}
          .
        </p>
      )}
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

// csvCell quotes a cell, and makes one a spreadsheet would run as a formula plain text.
export function csvCell(v: string | number | boolean): string {
  let s = String(v);
  if (/^[=+\-@\t\r]/.test(s)) s = "'" + s;
  return `"${s.replace(/"/g, '""')}"`;
}

type ReviewData = { items: ReviewItem[]; hidden: { owner?: string; owner_gone?: boolean }[] };

// reviewCSV is the review list exactly as shown (SB-2505): a row per document the steward
// can open, and those they can't only as a count with their owners, never a title.
export function reviewCSV(list: ReviewData): string {
  const rows: (string | number | boolean)[][] = [["title", "cite", "cited_30_days", "age_days", "marked_wrong", "reported_outdated", "owner_gone", "owners"]];
  for (const r of list.items) rows.push([r.title ?? r.cite, r.cite, r.cited, r.age_days ?? "", r.flagged, r.outdated ?? 0, r.owner_gone === true, ""]);
  if (list.hidden.length > 0) {
    const owners = Array.from(new Set(list.hidden.flatMap((h) => (h.owner ? [h.owner] : [])))).join("; ");
    rows.push([tn(list.hidden.length, "{n} more document you can't open", "{n} more documents you can't open"), "", "", "", "", "", "", owners]);
  }
  return rows.map((r) => r.map(csvCell).join(",")).join("\r\n") + "\r\n";
}

export function ReviewList({ space, fetcher }: { space: string; fetcher?: Fetch }) {
  const [list, setList] = useState<{
    items: ReviewItem[];
    hidden: { owner?: string; owner_gone?: boolean }[];
    personal_data?: number;
    glossary_flagged?: string[];
    same_name?: SameNameGroup[];
  }>({ items: [], hidden: [] });
  const [bump, setBump] = useState(0);
  const retired = (cite: string) => {
    setList((l) => ({ ...l, items: l.items.filter((x) => x.cite !== cite) }));
    setBump((b) => b + 1);
  };
  useEffect(() => {
    const load = () => void reviewList(space, fetcher).then(setList);
    load();
    const timer = window.setInterval(load, Refresh);
    return () => window.clearInterval(timer);
  }, [space, fetcher]);
  const download = () => {
    const url = URL.createObjectURL(new Blob([reviewCSV(list)], { type: "text/csv;charset=utf-8" }));
    const a = document.createElement("a");
    a.href = url;
    a.download = `review-${space}.csv`;
    a.click();
    URL.revokeObjectURL(url);
  };
  return (
    <section aria-label={t("Documents to review")}>
      <h2>
        {t("Documents to review")}
        <Count n={list.items.length + list.hidden.length} />
      </h2>
      {(list.items.length > 0 || list.hidden.length > 0) && (
        <button type="button" onClick={download}>
          {t("Download CSV")}
        </button>
      )}
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
                {r.outdated ? ` · ${tn(r.outdated, "reported out of date {n} time", "reported out of date {n} times")}` : ""}
                {r.owner_gone ? ` · ${t("owner gone: find it a new owner")}` : ""}
                {r.sources_moved ? ` · ${tn(r.sources_moved, "{n} of its sources changed or is gone", "{n} of its sources changed or are gone")}` : ""}
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
              {!r.owner_gone && !(r.wider !== undefined && r.wider > 0) && !(r.copies && r.copies.length > 0) && (
                <MarkReviewed space={space} item={r} fetcher={fetcher} onDone={() => setList((l) => ({ ...l, items: l.items.filter((x) => x.cite !== r.cite) }))} />
              )}
              {!r.question && <Retire space={space} cite={r.cite} name={r.title ?? r.cite} fetcher={fetcher} onDone={() => retired(r.cite)} />}
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
      <SameNames groups={list.same_name ?? []} />
      <NeverCited space={space} fetcher={fetcher} onRetired={() => setBump((b) => b + 1)} />
      <RetiredDocs space={space} fetcher={fetcher} bump={bump} />
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
