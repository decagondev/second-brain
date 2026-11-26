import { useEffect, useId, useState } from "react";
import type { FormEvent } from "react";
import { AskError, agentRuns, ask, caseCandidates, collections, decideCase, forgetMemory, listSpaces, makeCollection, myInbox, myMemory, remember, removeCollection } from "./api";
import type { AgentRun, AskResponse, CaseCandidate, Collection, Fetch, Inbox, InboxItem, MemoryLine } from "./api";
import { errorText, msg, t } from "./i18n";
import { CatchUp } from "./Across";
import { SharedOnPurpose } from "./Sharing";

const errMsg = (e: unknown, fallback: string) => (e instanceof AskError ? errorText(e.error).message : fallback);

// What each kind of item is, in words (SB-4301).
const kinds: Record<string, string> = {
  draft: msg("Draft to publish"),
  question: msg("Question"),
  request: msg("Request"),
  policy: msg("Policy to read"),
  approval: msg("Approval"),
  action: msg("Action item"),
  fix: msg("Suggested edits"),
  research: msg("Research report"),
  routine: msg("Routine"),
  case: msg("Eval case to decide"),
  wide: msg("Shared too widely"), // SB-5401
};

function Items({ items, label, empty }: { items: InboxItem[]; label: string; empty: string }) {
  return (
    <section aria-label={label} className="inbox-list">
      <h2>
        {label} ({items.length})
      </h2>
      {items.length === 0 ? (
        <p className="notice">{empty}</p>
      ) : (
        <ul className="saved-list">
          {items.map((x) => (
            <li key={x.kind + x.id}>
              <span className="meta">{t(kinds[x.kind] ?? x.kind)}</span> {x.title}{" "}
              <span className="meta">
                ({x.space || t("company-wide")}
                {x.at ? `, ${x.at.slice(0, 10)}` : ""}
                {x.due ? `, ${t("due {due}", { due: x.due })}` : ""})
              </span>
              {x.overdue && <strong> {t("Overdue")}</strong>}
              {x.why && <span className="refused"> {x.why}</span>}{" "}
              <a href={x.link} aria-label={t("Open: {title}", { title: x.title })}>
                {t("Open")}
              </a>
              {x.kind === "wide" && <SharedOnPurpose cite={x.id} />}
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}

// The inbox (sb-ask(3), One inbox; SB-4301): what waits on me in every space, built when
// shown, as me. It only lists; each item opens where it's acted on.
export function InboxView({ fetcher }: { fetcher?: Fetch }) {
  const [inbox, setInbox] = useState<Inbox>();
  const [error, setError] = useState<string>();
  useEffect(() => {
    myInbox(fetcher).then(setInbox, (e) => setError(errMsg(e, t("Your inbox is unavailable right now."))));
  }, [fetcher]);
  return (
    <section aria-label={t("Inbox")} className="inbox">
      {error && (
        <p className="error" role="alert">
          {error}
        </p>
      )}
      {inbox?.catch_up && <CatchUpOffer since={inbox.catch_up.since} fetcher={fetcher} />}
      {inbox && (
        <>
          <Items items={inbox.needs_you} label={t("Needs you")} empty={t("Nothing waits on you.")} />
          {inbox.failed.length > 0 && <Items items={inbox.failed} label={t("Failed")} empty="" />}
          <Items items={inbox.done} label={t("Done")} empty={t("Nothing done in the last 30 days.")} />
        </>
      )}
    </section>
  );
}

// About me (sb-ask(3), Memory I control; SB-4303): lines I wrote about myself, sent with my
// own asks only, never a document's text.
export function AboutMe({ fetcher }: { fetcher?: Fetch }) {
  const [on, setOn] = useState(true);
  const [lines, setLines] = useState<MemoryLine[]>();
  const [text, setText] = useState("");
  const [status, setStatus] = useState<string>();
  const id = useId();
  async function load() {
    try {
      const m = await myMemory(fetcher);
      setOn(m.on);
      setLines(m.lines);
    } catch (e) {
      setStatus(errMsg(e, t("Your memory is unavailable right now.")));
    }
  }
  useEffect(() => {
    void load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);
  async function add(e: FormEvent) {
    e.preventDefault();
    try {
      await remember(text, fetcher);
      setText("");
      setStatus(t("Remembered. Your own questions now carry it."));
      await load();
    } catch (err) {
      setStatus(errMsg(err, t("That couldn't be remembered.")));
    }
  }
  async function remove(line?: string) {
    try {
      await forgetMemory(line, fetcher);
      await load();
    } catch (err) {
      setStatus(errMsg(err, t("That couldn't be removed.")));
    }
  }
  return (
    <section aria-label={t("About me")} className="about-me">
      <h2>{t("About me")}</h2>
      {!on ? (
        <p className="notice">{t("Memory is off for your organisation.")}</p>
      ) : (
        <>
          <p className="meta">{t("A few lines about you, in your own words, sent with your own questions only. Never paste a document here.")}</p>
          {lines && lines.length > 0 && (
            <ul className="saved-list">
              {lines.map((l) => (
                <li key={l.id}>
                  {l.text}{" "}
                  <button type="button" onClick={() => remove(l.id)} aria-label={t("Remove: {question}", { question: l.text })}>
                    {t("Remove")}
                  </button>
                </li>
              ))}
            </ul>
          )}
          <form onSubmit={add}>
            <label htmlFor={id}>{t("Something about you")}</label>{" "}
            <input id={id} type="text" maxLength={200} value={text} onChange={(e) => setText(e.target.value)} />{" "}
            <button type="submit" disabled={!text.trim()}>
              {t("Add")}
            </button>
            {lines && lines.length > 0 && (
              <>
                {" "}
                <button type="button" onClick={() => remove(undefined)}>
                  {t("Remove all")}
                </button>
              </>
            )}
          </form>
        </>
      )}
      {status && (
        <p className="feedback-status" role="status">
          {status}
        </p>
      )}
    </section>
  );
}

// Remember something about me, under an answer (SB-4303): an empty field, never the
// answer's text.
export function RememberBox({ fetcher }: { fetcher?: Fetch }) {
  const [open, setOpen] = useState(false);
  const [text, setText] = useState("");
  const [status, setStatus] = useState<string>();
  const id = useId();
  async function add(e: FormEvent) {
    e.preventDefault();
    try {
      await remember(text, fetcher);
      setText("");
      setOpen(false);
      setStatus(t("Remembered. Your own questions now carry it."));
    } catch (err) {
      setStatus(errMsg(err, t("That couldn't be remembered.")));
    }
  }
  return (
    <div className="remember">
      {!open && (
        <button type="button" onClick={() => setOpen(true)}>
          {t("Remember something about me")}
        </button>
      )}
      {open && (
        <form onSubmit={add}>
          <label htmlFor={id}>{t("Something about you")}</label>{" "}
          <input id={id} type="text" maxLength={200} value={text} onChange={(e) => setText(e.target.value)} />{" "}
          <button type="submit" disabled={!text.trim()}>
            {t("Add")}
          </button>{" "}
          <button type="button" onClick={() => setOpen(false)}>
            {t("Cancel")}
          </button>
        </form>
      )}
      {status && (
        <p className="feedback-status" role="status">
          {status}
        </p>
      )}
    </div>
  );
}

// Agent runs (sb-ask(3), What agents did as me; SB-4305): what ran in my name, what it read
// that I can open now (the rest counted) and what it made. Nothing new is kept for it.
export function AgentRuns({ fetcher }: { fetcher?: Fetch }) {
  const [runs, setRuns] = useState<AgentRun[]>();
  const [error, setError] = useState<string>();
  useEffect(() => {
    agentRuns(fetcher).then(setRuns, (e) => setError(errMsg(e, t("Agent runs are unavailable right now."))));
  }, [fetcher]);
  return (
    <section aria-label={t("Agent runs")} className="inbox-list">
      <h2>{t("Agent runs")}</h2>
      {error && (
        <p className="error" role="alert">
          {error}
        </p>
      )}
      {runs && runs.length === 0 && <p className="notice">{t("No agent ran as you in the last 90 days.")}</p>}
      {runs && runs.length > 0 && (
        <ul className="saved-list">
          {runs.map((r, i) => (
            <li key={r.kind + r.at + i}>
              <span className="meta">{r.agent}</span> {r.title}{" "}
              <span className="meta">
                ({r.space || t("company-wide")}, {r.at.slice(0, 10)})
              </span>{" "}
              {r.drafted}
              {(r.read.length > 0 || r.hidden > 0) && (
                <p className="meta">
                  {t("Read: {titles}", { titles: r.read.map((d) => d.title || d.cite).join("; ") || "-" })}
                  {r.hidden > 0 && ` ${t("and {n} you can't open", { n: String(r.hidden) })}`}
                </p>
              )}{" "}
              <a href={r.link} aria-label={t("Open: {title}", { title: r.title })}>
                {t("Open")}
              </a>
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}

// Cases from feedback (sb-evals(7); SB-4401): a down-rated answer of an agent I answer for,
// shown only as far as I can open it. Accepting picks the document it should have cited.
export function AgentCases({ fetcher }: { fetcher?: Fetch }) {
  const [items, setItems] = useState<CaseCandidate[]>();
  const [status, setStatus] = useState<string>();
  const [expect, setExpect] = useState<Record<string, string>>({});
  async function load() {
    try {
      setItems(await caseCandidates(fetcher));
    } catch (e) {
      setStatus(errMsg(e, t("Candidate cases are unavailable right now.")));
    }
  }
  useEffect(() => {
    void load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);
  const docs = (c: CaseCandidate) => {
    const seen = new Map<string, string>();
    for (const cl of c.claims) for (const ci of cl.cites) seen.set(ci.cite.split("@")[0] ?? ci.cite, ci.title || ci.cite);
    return [...seen.entries()];
  };
  async function decide(c: CaseCandidate, dismiss: boolean) {
    try {
      const pick = expect[c.id] ?? docs(c)[0]?.[0] ?? "";
      await decideCase(c.id, dismiss ? { dismiss: true } : { expect: pick }, fetcher);
      setStatus(dismiss ? t("Candidate dismissed.") : t("Case added."));
      await load();
    } catch (e) {
      setStatus(errMsg(e, t("That candidate couldn't be decided.")));
    }
  }
  if (items && items.length === 0 && !status) return null;
  return (
    <section aria-label={t("Candidate cases")} className="inbox-list">
      <h2>{t("Candidate cases")}</h2>
      {status && <p role="status">{status}</p>}
      {items && items.length > 0 && (
        <ul className="saved-list">
          {items.map((c) => (
            <li key={c.id}>
              <span className="meta">{c.agent}</span> {c.question}{" "}
              <span className="meta">
                ({c.space.replace(/^spc_/, "") || t("company-wide")}, {c.at.slice(0, 10)})
              </span>
              <ul>
                {c.claims.map((cl, i) => (
                  <li key={i}>
                    {cl.text} <span className="meta">{cl.cites.map((ci) => ci.title || ci.cite).join("; ")}</span>
                  </li>
                ))}
              </ul>
              {c.hidden > 0 ? (
                <p className="meta">{t("{n} claims cite what you can't open, so this can't be a case.", { n: String(c.hidden) })}</p>
              ) : (
                docs(c).length > 0 && (
                  <label>
                    {t("Should cite")}{" "}
                    <select
                      aria-label={t("Should cite: {question}", { question: c.question })}
                      value={expect[c.id] ?? docs(c)[0]?.[0]}
                      onChange={(e) => setExpect({ ...expect, [c.id]: e.target.value })}
                    >
                      {docs(c).map(([k, title]) => (
                        <option key={k} value={k}>
                          {title}
                        </option>
                      ))}
                    </select>
                  </label>
                )
              )}{" "}
              {c.hidden === 0 && docs(c).length > 0 && (
                <button type="button" onClick={() => void decide(c, false)} aria-label={t("Add case: {question}", { question: c.question })}>
                  {t("Add case")}
                </button>
              )}{" "}
              <button type="button" onClick={() => void decide(c, true)} aria-label={t("Dismiss: {question}", { question: c.question })}>
                {t("Dismiss")}
              </button>
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}

// Collections (sb-ask(3), Collections; SB-4304): documents and saved questions for one piece
// of work. Each reader sees what they can open now; the rest is counted, never named.
export function CollectionsView({ fetcher }: { fetcher?: Fetch }) {
  const [items, setItems] = useState<Collection[]>();
  const [status, setStatus] = useState<string>();
  const [name, setName] = useState("");
  const [docs, setDocs] = useState("");
  const [people, setPeople] = useState("");
  const [asking, setAsking] = useState<string>();
  const [question, setQuestion] = useState("");
  const [answer, setAnswer] = useState<AskResponse>();
  const id = useId();
  async function load() {
    try {
      setItems(await collections(fetcher));
    } catch (e) {
      setStatus(errMsg(e, t("Your collections are unavailable right now.")));
    }
  }
  useEffect(() => {
    void load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);
  const split = (v: string) =>
    v
      .split(/[\s,]+/)
      .map((x) => x.trim())
      .filter(Boolean);
  async function make(e: FormEvent) {
    e.preventDefault();
    try {
      await makeCollection(name, split(docs), split(people), fetcher);
      setName("");
      setDocs("");
      setPeople("");
      setStatus(t("Collection made."));
      await load();
    } catch (err) {
      setStatus(errMsg(err, t("That collection couldn't be made.")));
    }
  }
  async function remove(c: string) {
    try {
      await removeCollection(c, fetcher);
      await load();
    } catch (err) {
      setStatus(errMsg(err, t("That collection couldn't be removed.")));
    }
  }
  async function askIt(e: FormEvent) {
    e.preventDefault();
    if (!asking) return;
    try {
      setAnswer(await ask(question, undefined, fetcher, undefined, undefined, undefined, undefined, undefined, undefined, undefined, asking));
    } catch (err) {
      setStatus(errMsg(err, t("That collection couldn't be asked.")));
    }
  }
  return (
    <section aria-label={t("Collections")} className="inbox-list">
      <h2>{t("Collections")}</h2>
      {items && items.length === 0 && <p className="notice">{t("No collections yet.")}</p>}
      {items && items.length > 0 && (
        <ul className="saved-list">
          {items.map((c) => (
            <li key={c.id}>
              <strong>{c.name}</strong> <span className="meta">({c.mine ? t("yours") : c.owner})</span>
              <ul>
                {c.items.map((x) => (
                  <li key={x.item}>{x.title || x.question || x.item}</li>
                ))}
              </ul>
              {c.hidden > 0 && <p className="meta">{t("and {n} you can't open", { n: String(c.hidden) })}</p>}
              {c.stale && <p className="notice">{t("No one has asked this collection for 90 days.")}</p>}
              <button type="button" onClick={() => setAsking(c.id)}>
                {t("Ask this collection")}
              </button>{" "}
              {c.mine && (
                <button type="button" onClick={() => void remove(c.id)} aria-label={t("Remove: {title}", { title: c.name })}>
                  {t("Remove")}
                </button>
              )}
            </li>
          ))}
        </ul>
      )}
      {asking && (
        <form onSubmit={askIt}>
          <label htmlFor={id + "q"}>{t("Your question about the collection")}</label>
          <input id={id + "q"} value={question} maxLength={4000} onChange={(e) => setQuestion(e.target.value)} />
          <button type="submit" disabled={!question.trim()}>
            {t("Ask")}
          </button>
        </form>
      )}
      {answer && (
        <ul className="saved-list" aria-label={t("Answer")}>
          {answer.answer.claims.map((cl, i) => (
            <li key={i}>
              {cl.text} {cl.uncited ? <span className="refused">{t("uncited")}</span> : <span className="meta">({cl.cites.map((ci) => ci.title || ci.cite).join("; ")})</span>}
            </li>
          ))}
        </ul>
      )}
      <form onSubmit={make}>
        <label htmlFor={id + "n"}>{t("Collection name")}</label>
        <input id={id + "n"} value={name} maxLength={120} onChange={(e) => setName(e.target.value)} />
        <label htmlFor={id + "d"}>{t("Documents and saved questions (ids)")}</label>
        <input id={id + "d"} value={docs} onChange={(e) => setDocs(e.target.value)} />
        <label htmlFor={id + "p"}>{t("Share with (emails)")}</label>
        <input id={id + "p"} value={people} onChange={(e) => setPeople(e.target.value)} />
        <button type="submit" disabled={!name.trim()}>
          {t("Make collection")}
        </button>
      </form>
      {status && <p role="status">{status}</p>}
    </section>
  );
}

// After a notification pause ends, the inbox offers Catch me up once, from the day it began
// (SB-4506), in one of the person's spaces.
function CatchUpOffer({ since, fetcher }: { since: string; fetcher?: Fetch }) {
  const [spaces, setSpaces] = useState<string[]>([]);
  const [space, setSpace] = useState("");
  const id = useId();
  useEffect(() => {
    listSpaces(fetcher).then((xs) => {
      setSpaces(xs.map((x) => x.name));
      setSpace(xs[0]?.name ?? "");
    }, () => undefined);
  }, [fetcher]);
  return (
    <section aria-label={t("Welcome back")} className="notice">
      <p>{t("Your notification pause has ended. Catch up on a space since {since}?", { since })}</p>
      {spaces.length > 0 && (
        <>
          <label htmlFor={id}>{t("Space")}</label>{" "}
          <select id={id} value={space} onChange={(e) => setSpace(e.target.value)}>
            {spaces.map((s) => (
              <option key={s} value={s}>
                {s}
              </option>
            ))}
          </select>
          {space && <CatchUp key={space} space={space} since={since} fetcher={fetcher} />}
        </>
      )}
    </section>
  );
}
