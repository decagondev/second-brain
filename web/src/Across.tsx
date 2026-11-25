import { useEffect, useId, useState } from "react";
import type { FormEvent } from "react";
import type { Claim, Fetch } from "./api";
import { AskError } from "./api";
import { ClaimView } from "./Answer";
import { call } from "./Steward";
import { errorText, t, tn } from "./i18n";

const errMsg = (e: unknown, fallback: string) => (e instanceof AskError ? errorText(e.error).message : fallback);
const enc = encodeURIComponent;

export interface WaitItem {
  kind: string;
  id: string;
  title: string;
  space: string;
  on: string;
  reason: string;
  until: string;
  overdue?: boolean;
  can_clear?: boolean;
}

function WaitRow({ w, fetcher, onClear }: { w: WaitItem; fetcher: Fetch; onClear: () => void }) {
  const [err, setErr] = useState("");
  async function clear() {
    try {
      await call(fetcher, "DELETE", `/v1/spaces/${enc(w.space)}/waits?kind=${enc(w.kind)}&id=${enc(w.id)}`);
      onClear();
    } catch (e) {
      setErr(errMsg(e, t("That couldn't be cleared.")));
    }
  }
  return (
    <li>
      {w.title} <span className="meta">{t("{from} waits on {on} until {until}: {reason}", { from: w.space, on: w.on, until: w.until || "?", reason: w.reason })}</span>
      {w.overdue && <strong> {t("Overdue")}</strong>}{" "}
      {w.can_clear && (
        <button type="button" onClick={() => void clear()} aria-label={t("Clear the wait: {title}", { title: w.title })}>
          {t("Clear")}
        </button>
      )}
      {err && <span className="refused"> {err}</span>}
    </li>
  );
}

// Waiting on another department (SB-4504): what this space waits on and what waits on it,
// each named only to someone who can see it; the rest counted.
export function WaitsCard({ space, fetcher = fetch }: { space: string; fetcher?: Fetch }) {
  const [d, setD] = useState<{ waiting: WaitItem[]; waited_on: WaitItem[]; hidden: number }>();
  const [n, setN] = useState(0);
  useEffect(() => {
    let live = true;
    call(fetcher, "GET", `/v1/spaces/${enc(space)}/waits`)
      .then(
        (x) =>
          live &&
          setD({ waiting: (x.waiting as WaitItem[]) ?? [], waited_on: (x.waited_on as WaitItem[]) ?? [], hidden: Number(x.hidden ?? 0) }),
      )
      .catch(() => undefined);
    return () => {
      live = false;
    };
  }, [space, fetcher, n]);
  if (!d || (d.waiting.length === 0 && d.waited_on.length === 0 && d.hidden === 0)) return null;
  const reload = () => setN((x) => x + 1);
  return (
    <section aria-label={t("Waiting")} className="start">
      <h2>{t("Waiting")}</h2>
      {d.waiting.length > 0 && (
        <>
          <h3>{t("This space waits on")}</h3>
          <ul>
            {d.waiting.map((w) => (
              <WaitRow key={w.kind + w.id} w={w} fetcher={fetcher} onClear={reload} />
            ))}
          </ul>
        </>
      )}
      {d.waited_on.length > 0 && (
        <>
          <h3>{t("Waiting on this space")}</h3>
          <ul>
            {d.waited_on.map((w) => (
              <WaitRow key={w.kind + w.id} w={w} fetcher={fetcher} onClear={reload} />
            ))}
          </ul>
        </>
      )}
      {d.hidden > 0 && <p className="refused">{tn(d.hidden, "{n} more you can't see.", "{n} more you can't see.")}</p>}
    </section>
  );
}

// Mark a request or action item as waiting on another space (SB-4504).
export function WaitForm({ space, kind, id, fetcher = fetch }: { space: string; kind: "request" | "action"; id: string; fetcher?: Fetch }) {
  const [open, setOpen] = useState(false);
  const [on, setOn] = useState("");
  const [reason, setReason] = useState("");
  const [until, setUntil] = useState("");
  const [status, setStatus] = useState("");
  const f = useId();
  async function mark(e: FormEvent) {
    e.preventDefault();
    try {
      await call(fetcher, "PUT", `/v1/spaces/${enc(space)}/waits`, { kind, id, on, reason, until });
      setStatus(t("Marked as waiting on {space}.", { space: on }));
      setOpen(false);
    } catch (err) {
      setStatus(errMsg(err, t("That couldn't be marked.")));
    }
  }
  if (!open) {
    return (
      <>
        <button type="button" onClick={() => setOpen(true)}>
          {t("Waiting on…")}
        </button>
        {status && <span className="meta"> {status}</span>}
      </>
    );
  }
  return (
    <form onSubmit={(e) => void mark(e)} aria-label={t("Waiting on another space")}>
      <label htmlFor={f + "on"}>{t("Space")}</label> <input id={f + "on"} value={on} onChange={(e) => setOn(e.target.value)} required />{" "}
      <label htmlFor={f + "reason"}>{t("Why")}</label> <input id={f + "reason"} value={reason} maxLength={200} onChange={(e) => setReason(e.target.value)} required />{" "}
      <label htmlFor={f + "until"}>{t("Expected by")}</label> <input id={f + "until"} type="date" value={until} onChange={(e) => setUntil(e.target.value)} required />{" "}
      <button type="submit">{t("Mark waiting")}</button>
      {status && <span className="refused"> {status}</span>}
    </form>
  );
}

interface ServiceRow {
  category: string;
  week: string;
  count: number;
  first_reply_hours?: number;
  close_hours?: number;
  late_share?: number;
  fewer_than_5?: boolean;
}

interface ServiceTeam {
  space: string;
  count: number;
  closed: number;
  late: number;
  fewer_than_5?: boolean;
}

// Department service levels (SB-4507): for the space's stewards only, counts and medians,
// every row under 5 masked. Nothing is shown to anyone else.
export function ServiceCard({ space, fetcher = fetch }: { space: string; fetcher?: Fetch }) {
  const [d, setD] = useState<{ rows: ServiceRow[]; teams: ServiceTeam[] }>();
  useEffect(() => {
    let live = true;
    call(fetcher, "GET", `/v1/spaces/${enc(space)}/service`)
      .then((x) => live && setD({ rows: (x.rows as ServiceRow[]) ?? [], teams: (x.teams as ServiceTeam[]) ?? [] }))
      .catch(() => undefined);
    return () => {
      live = false;
    };
  }, [space, fetcher]);
  if (!d || (d.rows.length === 0 && d.teams.length === 0)) return null;
  const hours = (h?: number) => (h === undefined ? "–" : t("{n} h", { n: String(h) }));
  return (
    <section aria-label={t("Service")} className="start">
      <h2>{t("Service")}</h2>
      {d.rows.length > 0 && (
        <table>
          <thead>
            <tr>
              <th scope="col">{t("Week")}</th>
              <th scope="col">{t("Category")}</th>
              <th scope="col">{t("Sent")}</th>
              <th scope="col">{t("First reply")}</th>
              <th scope="col">{t("Time to close")}</th>
              <th scope="col">{t("Closed late")}</th>
            </tr>
          </thead>
          <tbody>
            {d.rows.map((r) => (
              <tr key={r.week + r.category}>
                <td>{r.week}</td>
                <td>{r.category}</td>
                {r.fewer_than_5 ? (
                  <td colSpan={4}>{t("fewer than 5")}</td>
                ) : (
                  <>
                    <td>{r.count}</td>
                    <td>{hours(r.first_reply_hours)}</td>
                    <td>{hours(r.close_hours)}</td>
                    <td>{r.late_share === undefined ? "–" : `${Math.round(r.late_share * 100)}%`}</td>
                  </>
                )}
              </tr>
            ))}
          </tbody>
        </table>
      )}
      {d.teams.length > 0 && (
        <ul aria-label={t("Teams")}>
          {d.teams.map((x) => (
            <li key={x.space}>
              {x.space}:{" "}
              {x.fewer_than_5 ? t("fewer than 5") : t("{count} sent, {closed} closed, {late} late", { count: String(x.count), closed: String(x.closed), late: String(x.late) })}
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}

const visitKey = (space: string) => `sb.visit.${space}`;

// The day of the person's last visit to a space, kept in their browser (SB-4506).
export function lastVisit(space: string): string | undefined {
  try {
    return localStorage.getItem(visitKey(space)) ?? undefined;
  } catch {
    return undefined;
  }
}

export function markVisit(space: string, day = new Date().toISOString().slice(0, 10)) {
  try {
    localStorage.setItem(visitKey(space), day);
  } catch {
    // a browser that keeps nothing just gets the default week
  }
}

interface CatchUpAnswer {
  trace: string;
  answer: { claims: Claim[] };
  catch_up: { since: string; documents: number; waiting: number };
}

// Catch me up (SB-4506): one cited summary since a day, as me, from what changed that I can
// open now. The day defaults to my last visit here.
export function CatchUp({ space, branch, since: offered, fetcher = fetch }: { space?: string; branch?: boolean; since?: string; fetcher?: Fetch }) {
  const [since, setSince] = useState(offered ?? (space ? lastVisit(space) : undefined) ?? "");
  const [res, setRes] = useState<CatchUpAnswer>();
  const [err, setErr] = useState("");
  const id = useId();
  async function run(e: FormEvent) {
    e.preventDefault();
    setErr("");
    const q = space ? `?space=${enc(space)}${branch ? "&branch=true" : ""}` : "";
    try {
      setRes((await call(fetcher, "POST", `/v1/catch-up${q}`, since ? { since } : {})) as unknown as CatchUpAnswer);
    } catch (x) {
      setErr(errMsg(x, t("Catch me up is unavailable right now.")));
    }
  }
  return (
    <section aria-label={t("Catch me up")} className="start">
      <form onSubmit={(e) => void run(e)}>
        <label htmlFor={id}>{t("Since")}</label> <input id={id} type="date" value={since} onChange={(e) => setSince(e.target.value)} />{" "}
        <button type="submit">{t("Catch me up")}</button>
      </form>
      {err && (
        <p className="error" role="alert">
          {err}
        </p>
      )}
      {res && (
        <>
          <p className="meta">
            {tn(res.catch_up.documents, "{n} document changed since {since}.", "{n} documents changed since {since}.", { since: res.catch_up.since })}{" "}
            {res.catch_up.waiting > 0 && tn(res.catch_up.waiting, "{n} thing here waits on you.", "{n} things here wait on you.")}
          </p>
          <ul className="claims">
            {res.answer.claims.map((c, i) => (
              <ClaimView key={i} claim={c} trace={res.trace} fetcher={fetcher} />
            ))}
          </ul>
        </>
      )}
    </section>
  );
}

interface TrailDecision {
  cite: string;
  title: string;
  url?: string;
  owner?: string;
  at: string;
  supersedes?: { cite: string; title?: string };
  replaced_by?: { cite: string; title?: string };
}

// A topic's trail of decisions (SB-4505), oldest first, each one I can open.
export function Trail({ space, fetcher = fetch }: { space: string; fetcher?: Fetch }) {
  const [topic, setTopic] = useState("");
  const [d, setD] = useState<{ decisions: TrailDecision[]; hidden: number }>();
  const [err, setErr] = useState("");
  const id = useId();
  async function load(e: FormEvent) {
    e.preventDefault();
    try {
      const x = await call(fetcher, "GET", `/v1/spaces/${enc(space)}/decisions?topic=${enc(topic)}`);
      setD({ decisions: (x.decisions as TrailDecision[]) ?? [], hidden: Number(x.hidden ?? 0) });
    } catch (x) {
      setErr(errMsg(x, t("The decisions are unavailable right now.")));
    }
  }
  return (
    <section aria-label={t("Decision trail")} className="start">
      <form onSubmit={(e) => void load(e)}>
        <label htmlFor={id}>{t("Decisions about")}</label> <input id={id} value={topic} onChange={(e) => setTopic(e.target.value)} />{" "}
        <button type="submit">{t("Show the trail")}</button>
      </form>
      {err && <p className="refused">{err}</p>}
      {d && (
        <ol>
          {d.decisions.map((x) => (
            <li key={x.cite}>
              {x.url ? (
                <a href={x.url} target="_blank" rel="noopener noreferrer">
                  {x.title}
                </a>
              ) : (
                x.title
              )}{" "}
              <span className="meta">
                {x.at.slice(0, 10)}
                {x.owner ? `, ${x.owner}` : ""}
              </span>
              {x.replaced_by && <span className="meta"> {t("Replaced by {title}", { title: x.replaced_by.title ?? x.replaced_by.cite })}</span>}
            </li>
          ))}
        </ol>
      )}
      {d && d.decisions.length === 0 && <p className="meta">{t("No decisions found.")}</p>}
      {d && d.hidden > 0 && <p className="refused">{tn(d.hidden, "{n} decision is not shown because you can't open it.", "{n} decisions are not shown because you can't open them.")}</p>}
    </section>
  );
}
