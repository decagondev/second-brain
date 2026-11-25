import { useEffect, useId, useState } from "react";
import type { FormEvent } from "react";
import type { DraftSuggestion, Fetch, SavedQuestion } from "./api";
import { BRANCH, suggestedQuestions } from "./api";
import { call } from "./Steward";
import { t, tn } from "./i18n";

// Shape the work (SB-4601 to SB-4603): a space's skills, Ask for changes on an agent draft,
// and routines shared with a space.

const enc = encodeURIComponent;
const msg = (e: unknown, fallback: string) => (e instanceof Error && e.message ? e.message : fallback);

export interface Skill {
  name: string;
  instruction: string;
  from?: string;
}

// The space's skills while the question starts with "/": each loads with /NAME (SB-4601),
// and its starter questions (SB-5207), each asked as it is.
export function SkillHint({ space, question, fetcher = fetch, onPick }: { space?: string; question: string; fetcher?: Fetch; onPick: (q: string) => void }) {
  const [skills, setSkills] = useState<Skill[]>();
  const [starters, setStarters] = useState<string[]>([]);
  const typing = question.startsWith("/") && !question.includes(" ");
  useEffect(() => {
    if (!space || !typing || skills) return;
    let live = true;
    const name = space.endsWith(BRANCH) ? space.slice(0, -BRANCH.length) : space;
    call(fetcher, "GET", `/v1/spaces/${enc(name)}/skills`)
      .then((x) => live && setSkills((x.skills as Skill[]) ?? []))
      .catch(() => live && setSkills([]));
    suggestedQuestions(name, fetcher).then((qs) => live && setStarters(qs)).catch(() => undefined);
    return () => {
      live = false;
    };
  }, [space, typing, skills, fetcher]);
  if (!space || !typing || !skills) return null;
  const want = question.slice(1);
  const shown = skills.filter((s) => s.name.startsWith(want));
  const startShown = starters.filter((q) => q.toLowerCase().includes(want.toLowerCase())).slice(0, 5);
  if (shown.length === 0 && startShown.length === 0) return null;
  return (
    <ul className="results" aria-label={t("Skills")}>
      {startShown.map((q) => (
        <li key={"q:" + q}>
          <button type="button" onClick={() => onPick(q)}>
            {q}
          </button>{" "}
          <span className="meta">{t("starter question")}</span>
        </li>
      ))}
      {shown.map((s) => (
        <li key={s.name}>
          <button type="button" onClick={() => onPick(`/${s.name} `)}>
            /{s.name}
          </button>{" "}
          <span className="meta">
            {s.instruction}
            {s.from ? ` · ${t("from {space}", { space: s.from })}` : ""}
          </span>
        </li>
      ))}
    </ul>
  );
}

// Ask for changes on an agent draft (SB-4602): a note of at most 500 characters, at most 3
// rounds, from the same sources; the diff of the last round shows below.
export function AskForChanges({
  space,
  id,
  rounds = 0,
  diff,
  fetcher = fetch,
  onDone,
}: {
  space: string;
  id: string;
  rounds?: number;
  diff?: string[];
  fetcher?: Fetch;
  onDone: (text: string) => void;
}) {
  const [open, setOpen] = useState(false);
  const [note, setNote] = useState("");
  const [status, setStatus] = useState("");
  const [busy, setBusy] = useState(false);
  const f = useId();
  async function send(e: FormEvent) {
    e.preventDefault();
    setBusy(true);
    try {
      const r = await call(fetcher, "POST", `/v1/spaces/${enc(space)}/questions/${enc(id)}/changes`, { note });
      setStatus(t("Redrafted: round {n} of 3.", { n: String(r.rounds) }));
      setOpen(false);
      setNote("");
      onDone(String(r.text ?? ""));
    } catch (err) {
      setStatus(msg(err, t("The draft couldn't be redrafted.")));
    } finally {
      setBusy(false);
    }
  }
  return (
    <div>
      {(diff?.length ?? 0) > 0 && (
        <ul className="diff" aria-label={t("Changes in the last round")}>
          {diff!.map((l, i) => (
            <li key={i} className={l.startsWith("+") ? "added" : "removed"}>
              {l}
            </li>
          ))}
        </ul>
      )}
      {rounds > 0 && <p className="meta">{tn(rounds, "{n} round of changes", "{n} rounds of changes")}</p>}
      {rounds < 3 && !open && (
        <button type="button" onClick={() => setOpen(true)}>
          {t("Ask for changes")}
        </button>
      )}
      {open && (
        <form onSubmit={(e) => void send(e)} aria-label={t("Ask for changes")}>
          <label htmlFor={f}>{t("What should change?")}</label>{" "}
          <textarea id={f} rows={2} maxLength={500} value={note} onChange={(e) => setNote(e.target.value)} required />{" "}
          <button type="submit" disabled={busy}>
            {t("Redraft")}
          </button>
        </form>
      )}
      {status && (
        <p className="meta" role="status">
          {status}
        </p>
      )}
    </div>
  );
}

// The redraft's suggestions (SB-5202): each open one with what the section held and what the
// agent wrote, and Accept or Reject; a draft with an open one can't be published.
export function Suggestions({
  space,
  id,
  items,
  fetcher = fetch,
  onDone,
}: {
  space: string;
  id: string;
  items?: DraftSuggestion[];
  fetcher?: Fetch;
  onDone: (text: string) => void;
}) {
  const [status, setStatus] = useState("");
  const open = (items ?? []).filter((s) => s.state === "open");
  if (open.length === 0) return null;
  async function decide(n: number, accept: boolean) {
    try {
      const r = await call(fetcher, "POST", `/v1/spaces/${enc(space)}/questions/${enc(id)}/suggestions/${n}/${accept ? "accept" : "reject"}`);
      setStatus(accept ? t("Suggestion accepted.") : t("Suggestion rejected: the section is back as it was."));
      onDone(String(r.text ?? ""));
    } catch (err) {
      setStatus(msg(err, t("The suggestion couldn't be changed.")));
    }
  }
  return (
    <section aria-label={t("Suggestions")}>
      <p className="meta">{tn(open.length, "{n} suggestion to accept or reject before you publish", "{n} suggestions to accept or reject before you publish")}</p>
      <ol>
        {open.map((s) => (
          <li key={s.n}>
            <ul className="diff">
              <li className="removed">{s.old}</li>
              <li className="added">{s.new}</li>
            </ul>
            <button type="button" onClick={() => void decide(s.n, true)}>
              {t("Accept")}
            </button>{" "}
            <button type="button" onClick={() => void decide(s.n, false)}>
              {t("Reject")}
            </button>
          </li>
        ))}
      </ol>
      {status && (
        <p className="meta" role="status">
          {status}
        </p>
      )}
    </section>
  );
}

interface SharedRoutine {
  id: string;
  question: string;
  every: string;
  day?: string;
  joined: boolean;
  mine?: boolean;
  opted_in?: number;
}

// Shared routines on the Start tab (SB-4603): opt in to a steward's routine, run as you.
export function SharedRoutines({ space, fetcher = fetch }: { space: string; fetcher?: Fetch }) {
  const [items, setItems] = useState<SharedRoutine[]>([]);
  const [status, setStatus] = useState("");
  const [n, setN] = useState(0);
  useEffect(() => {
    let live = true;
    call(fetcher, "GET", `/v1/spaces/${enc(space)}/routines`)
      .then((x) => live && setItems((x.routines as SharedRoutine[]) ?? []))
      .catch(() => live && setItems([]));
    return () => {
      live = false;
    };
  }, [space, fetcher, n]);
  async function join(r: SharedRoutine, on: boolean) {
    try {
      await call(fetcher, on ? "POST" : "DELETE", `/v1/spaces/${enc(space)}/routines/${enc(r.id)}/join`);
      setStatus(on ? t("Opted in: it runs as you.") : t("Opted out."));
      setN((x) => x + 1);
    } catch (err) {
      setStatus(msg(err, t("That didn't work.")));
    }
  }
  if (items.length === 0) return null;
  return (
    <section aria-label={t("Shared routines")} className="start">
      <h2>{t("Shared routines")}</h2>
      <ul className="results">
        {items.map((r) => (
          <li key={r.id}>
            {r.question} <span className="meta">· {r.every === "daily" ? t("Each working morning") : t("Weekly on {day}", { day: r.day ?? "" })}</span>{" "}
            {r.mine ? (
              <span className="meta">· {tn(r.opted_in ?? 0, "{n} member opted in", "{n} members opted in")}</span>
            ) : (
              <button type="button" onClick={() => void join(r, !r.joined)} aria-label={r.joined ? t("Opt out: {question}", { question: r.question }) : t("Opt in: {question}", { question: r.question })}>
                {r.joined ? t("Opt out") : t("Opt in")}
              </button>
            )}
          </li>
        ))}
      </ul>
      {status && (
        <p className="meta" role="status">
          {status}
        </p>
      )}
    </section>
  );
}

// Share with the space, on a steward's own routine in the Saved tab (SB-4603).
export function ShareRoutine({ q, fetcher = fetch, onChange }: { q: SavedQuestion; fetcher?: Fetch; onChange: () => void }) {
  const [err, setErr] = useState("");
  const name = q.space.replace(/^spc_/, "");
  if (!q.routine || q.from || !q.space || q.space === "company") return null;
  async function share(on: boolean) {
    try {
      await call(fetcher, on ? "PUT" : "DELETE", `/v1/spaces/${enc(name)}/routines/${enc(q.id)}`);
      setErr("");
      onChange();
    } catch (e) {
      setErr(msg(e, t("That didn't work.")));
    }
  }
  return (
    <>
      {q.shared ? (
        <button type="button" onClick={() => void share(false)} aria-label={t("Stop sharing: {question}", { question: q.question })}>
          {t("Stop sharing")}
        </button>
      ) : (
        <button type="button" onClick={() => void share(true)} aria-label={t("Share with the space: {question}", { question: q.question })}>
          {t("Share with the space")}
        </button>
      )}
      {err && <span className="refused"> {err}</span>}
    </>
  );
}
