import { useEffect, useState } from "react";
import type { Fetch } from "./api";
import { call } from "./Steward";
import { msg, t, tn } from "./i18n";

// Run the company (SB-4101 to SB-4103): a space's goals, its teams' latest updates and the
// policies its members are asked to read, on its Start tab; and the stewards' policy list.

export interface GoalRow {
  cite: string;
  title: string;
  url?: string;
  space: string;
  owner?: string;
  quarter?: string;
  status?: string;
  // No check-in published for three weeks (SB-4207).
  silent?: boolean;
  goals?: GoalRow[];
}

const statuses: Record<string, string> = { "on track": msg("on track"), "at risk": msg("at risk"), "off track": msg("off track"), done: msg("done") };

function GoalList({ goals }: { goals: GoalRow[] }) {
  return (
    <ul>
      {goals.map((g) => (
        <li key={g.cite}>
          {g.url ? (
            <a href={g.url} target="_blank" rel="noopener noreferrer">
              {g.title}
            </a>
          ) : (
            <span>{g.title}</span>
          )}{" "}
          <span className="meta">
            · {g.space}
            {g.status && ` · ${t(statuses[g.status] ?? g.status)}`}
            {g.owner && ` · ${g.owner}`}
          </span>
          {g.silent && <span className="meta overdue"> · {t("No check-in for 3 weeks")}</span>}
          {g.goals && g.goals.length > 0 && <GoalList goals={g.goals} />}
        </li>
      ))}
    </ul>
  );
}

// Goals: the branch's goals by quarter, nested, each one the member can open now (SB-4101).
export function Goals({ space, fetcher = fetch }: { space: string; fetcher?: Fetch }) {
  const [d, setD] = useState<{ quarters: { quarter: string; goals: GoalRow[] }[]; hidden: number }>();
  useEffect(() => {
    let live = true;
    call(fetcher, "GET", `/v1/spaces/${encodeURIComponent(space)}/goals`)
      .then((x) => live && setD({ quarters: (x.quarters as { quarter: string; goals: GoalRow[] }[]) ?? [], hidden: Number(x.hidden ?? 0) }))
      .catch(() => undefined);
    return () => {
      live = false;
    };
  }, [space, fetcher]);
  if (!d || (d.quarters.length === 0 && d.hidden === 0)) return null;
  return (
    <section aria-label={t("Goals")} className="start">
      <h2>{t("Goals")}</h2>
      {d.quarters.map((q) => (
        <div key={q.quarter || "none"}>
          <h3>{q.quarter || t("No quarter")}</h3>
          <GoalList goals={q.goals} />
        </div>
      ))}
      {d.hidden > 0 && <p className="refused">{tn(d.hidden, "{n} goal is not shown because you can't open it.", "{n} goals are not shown because you can't open them.")}</p>}
    </section>
  );
}

interface TeamUpdate {
  space: string;
  title: string;
  cite: string;
  url?: string;
  at: string;
}

// Team updates: each team's latest published update the member can open (SB-4102).
export function TeamUpdates({ space, fetcher = fetch }: { space: string; fetcher?: Fetch }) {
  const [items, setItems] = useState<TeamUpdate[]>([]);
  useEffect(() => {
    let live = true;
    call(fetcher, "GET", `/v1/spaces/${encodeURIComponent(space)}/updates`)
      .then((x) => live && setItems((x.updates as TeamUpdate[]) ?? []))
      .catch(() => undefined);
    return () => {
      live = false;
    };
  }, [space, fetcher]);
  if (items.length === 0) return null;
  return (
    <section aria-label={t("Latest updates")} className="start">
      <h2>{t("Latest updates")}</h2>
      <ul>
        {items.map((u) => (
          <li key={u.cite}>
            <strong>{u.space}</strong> ·{" "}
            {u.url ? (
              <a href={u.url} target="_blank" rel="noopener noreferrer">
                {u.title}
              </a>
            ) : (
              u.title
            )}{" "}
            <span className="meta">· {u.at.slice(0, 10)}</span>
          </li>
        ))}
      </ul>
    </section>
  );
}

interface PolicyItem {
  space: string;
  doc: string;
  cite: string;
  title: string;
  url?: string;
  version: number;
  read: boolean;
}

// Policies to read, until the member presses I've read it (SB-4103).
export function PoliciesToRead({ space, fetcher = fetch }: { space: string; fetcher?: Fetch }) {
  const [items, setItems] = useState<PolicyItem[]>([]);
  const [status, setStatus] = useState<string>();
  const base = `/v1/spaces/${encodeURIComponent(space)}/policies`;
  async function load() {
    try {
      const x = await call(fetcher, "GET", base);
      setItems((x.policies as PolicyItem[]) ?? []);
    } catch {
      setItems([]);
    }
  }
  useEffect(() => {
    void load();
  }, [space, fetcher]);
  async function read(p: PolicyItem) {
    try {
      await call(fetcher, "POST", `${base}/read`, { doc: p.doc, version: p.version });
      setStatus(t("Recorded: you've read {title}.", { title: p.title }));
      await load();
    } catch (e) {
      setStatus((e as Error).message);
    }
  }
  const open = items.filter((p) => !p.read);
  if (open.length === 0 && !status) return null;
  return (
    <section aria-label={t("Policies to read")} className="start">
      <h2>{t("Policies to read")}</h2>
      <ul>
        {open.map((p) => (
          <li key={p.doc}>
            {p.url ? (
              <a href={p.url} target="_blank" rel="noopener noreferrer">
                {p.title}
              </a>
            ) : (
              p.title
            )}{" "}
            <button type="button" onClick={() => read(p)}>
              {t("I've read {title}", { title: p.title })}
            </button>
          </li>
        ))}
      </ul>
      {status && <p role="status">{status}</p>}
    </section>
  );
}

interface PolicyStatus extends PolicyItem {
  not_read: string[];
  read: boolean;
  cant_open: number;
}

// The stewards' policies: mark one, and see who hasn't read each by name (SB-4103).
export function StewardPolicies({ base, fetcher = fetch }: { base: string; fetcher?: Fetch }) {
  const [items, setItems] = useState<(PolicyStatus & { read_count: number })[]>([]);
  const [doc, setDoc] = useState("");
  const [status, setStatus] = useState<string>();
  async function load() {
    try {
      const x = await call(fetcher, "GET", `${base}/policies/status`);
      const ps = (x.policies as (Omit<PolicyStatus, "read"> & { read: number })[]) ?? [];
      setItems(ps.map((p) => ({ ...p, read: false, read_count: p.read })));
    } catch (e) {
      setStatus((e as Error).message);
    }
  }
  useEffect(() => {
    void load();
  }, [base, fetcher]);
  async function mark() {
    try {
      const p = await call(fetcher, "POST", `${base}/policies`, { doc: doc.trim() });
      setStatus(t("{title} is now a policy to read.", { title: String(p.title) }));
      setDoc("");
      await load();
    } catch (e) {
      setStatus((e as Error).message);
    }
  }
  return (
    <section aria-label={t("Policies to read")}>
      <h2>{t("Policies to read")}</h2>
      <label htmlFor="policy-doc">{t("Document (KIND:ID)")}</label>
      <input id="policy-doc" value={doc} onChange={(e) => setDoc(e.target.value)} />
      <button type="button" disabled={doc.trim() === ""} onClick={mark}>
        {t("Mark as a policy")}
      </button>
      <ul>
        {items.map((p) => (
          <li key={p.doc}>
            <strong>{p.title}</strong>{" "}
            <span className="meta">
              · {tn(p.read_count, "{n} has read it", "{n} have read it")}
              {p.cant_open > 0 && ` · ${tn(p.cant_open, "{n} member can't open it", "{n} members can't open it")}`}
            </span>
            {p.not_read.length > 0 && <p>{t("Not read yet: {people}", { people: p.not_read.join(", ") })}</p>}
          </li>
        ))}
      </ul>
      {status && <p role="status">{status}</p>}
    </section>
  );
}

interface MyRequest {
  id: string;
  question: string;
  space: string;
  state: string;
  request?: { category: string; due: string };
}

// Ask the department (SB-4104): a request in one of the space's categories goes to its
// stewards with a due date; the member's own open requests to it are listed.
export function Requests({ space, fetcher = fetch }: { space: string; fetcher?: Fetch }) {
  const [cats, setCats] = useState<string[]>([]);
  const [mine, setMine] = useState<MyRequest[]>([]);
  const [category, setCategory] = useState("");
  const [text, setText] = useState("");
  const [status, setStatus] = useState<string>();
  const base = `/v1/spaces/${encodeURIComponent(space)}/requests`;
  async function load() {
    try {
      const f = await call(fetcher, "GET", base);
      const cs = (f.categories as string[]) ?? [];
      setCats(cs);
      setCategory((c) => c || cs[0] || "");
      const q = await call(fetcher, "GET", "/v1/questions");
      setMine(((q.items as MyRequest[]) ?? []).filter((x) => x.request && x.state === "open" && x.space === `spc_${space}`));
    } catch {
      setCats([]);
    }
  }
  useEffect(() => {
    void load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [space, fetcher]);
  async function send() {
    try {
      const r = await call(fetcher, "POST", base, { category, text: text.trim() });
      setText("");
      setStatus(t("Sent. It is due {due}.", { due: String(r.due ?? "").slice(0, 10) }));
      await load();
    } catch {
      setStatus(t("The request couldn't be sent."));
    }
  }
  if (cats.length === 0 && mine.length === 0) return null;
  return (
    <section aria-label={t("Ask the department")} className="start">
      <h2>{t("Ask the department")}</h2>
      {mine.length > 0 && (
        <ul>
          {mine.map((r) => (
            <li key={r.id}>
              {r.question}{" "}
              <span className="meta">
                · {r.request!.category} · {t("due {due}", { due: r.request!.due.slice(0, 10) })}
              </span>
            </li>
          ))}
        </ul>
      )}
      {cats.length > 0 && (
        <form
          onSubmit={(e) => {
            e.preventDefault();
            if (text.trim()) void send();
          }}
        >
          <label>
            {t("Category")}{" "}
            <select value={category} onChange={(e) => setCategory(e.target.value)}>
              {cats.map((c) => (
                <option key={c} value={c}>
                  {c}
                </option>
              ))}
            </select>
          </label>
          <label>
            {t("What do you need?")} <textarea value={text} onChange={(e) => setText(e.target.value)} maxLength={4000} />
          </label>
          <button type="submit" disabled={!text.trim()}>
            {t("Send request")}
          </button>
        </form>
      )}
      {status && <p role="status">{status}</p>}
    </section>
  );
}

interface SpaceSpend {
  space: string;
  parent?: string;
  own_usd: number;
  spent_usd: number;
  budget_usd_month: number;
  forecast_usd?: number;
}

const usd = (v: number) => `$${v.toFixed(2)}`;

// Spend by space (SB-4105): each branch's spend this month, its budget and forecast.
export function AdminSpend({ fetcher = fetch, month = new Date().toISOString().slice(0, 7) }: { fetcher?: Fetch; month?: string }) {
  const [rows, setRows] = useState<SpaceSpend[]>([]);
  const [error, setError] = useState<string>();
  useEffect(() => {
    call(fetcher, "GET", `/v1/admin/costs/spaces?month=${month}`)
      .then((d) => setRows((d.spaces as SpaceSpend[]) ?? []))
      .catch((e: Error) => setError(e.message));
  }, [fetcher, month]);
  return (
    <section aria-label={t("Spend by space")}>
      <h2>{t("Spend by space")}</h2>
      {error ? (
        <p role="alert">{error}</p>
      ) : (
        <table>
          <thead>
            <tr>
              <th scope="col">{t("Space")}</th>
              <th scope="col">{t("Its own")}</th>
              <th scope="col">{t("With its teams")}</th>
              <th scope="col">{t("Budget")}</th>
              <th scope="col">{t("Forecast")}</th>
            </tr>
          </thead>
          <tbody>
            {rows.map((r) => (
              <tr key={r.space} className={r.budget_usd_month > 0 && r.spent_usd >= r.budget_usd_month ? "overdue" : undefined}>
                <th scope="row">{r.parent ? `${r.parent} / ${r.space}` : r.space}</th>
                <td>{usd(r.own_usd)}</td>
                <td>{usd(r.spent_usd)}</td>
                <td>{r.budget_usd_month > 0 ? usd(r.budget_usd_month) : "-"}</td>
                <td>{r.forecast_usd !== undefined ? usd(r.forecast_usd) : "-"}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </section>
  );
}

// The AI command centre (sb-admin(3); SB-5001): 12 weeks of answers and spend by feature,
// the personal limit's bands, never who, and the rows as CSV.
interface UsageTotal {
  feature?: string;
  uses: number;
  usd: number;
}
interface AIUsage {
  weeks: number;
  from: string;
  features: UsageTotal[];
  total: UsageTotal;
  limit?: { monthly_usd_per_person: number; under_80: string; from_80: string; at_limit: string };
}

const band = (b: string) => (b === "fewer than 5" ? t("fewer than 5") : b);

export function AdminAIUsage({ fetcher = fetch }: { fetcher?: Fetch }) {
  const [u, setU] = useState<AIUsage>();
  const [error, setError] = useState<string>();
  useEffect(() => {
    call(fetcher, "GET", "/v1/admin/ai/usage")
      .then((d) => {
        const x = d as unknown as AIUsage;
        if (x && x.total) setU(x);
      })
      .catch((e: Error) => setError(e.message));
  }, [fetcher]);
  return (
    <section aria-label={t("AI use")}>
      <h2>{t("AI use")}</h2>
      {error && <p role="alert">{error}</p>}
      {u && (
        <>
          <p className="meta">{t("{weeks} weeks from {from}", { weeks: String(u.weeks), from: u.from })}</p>
          <table>
            <thead>
              <tr>
                <th scope="col">{t("Feature")}</th>
                <th scope="col">{t("Answers")}</th>
                <th scope="col">{t("Spend")}</th>
              </tr>
            </thead>
            <tbody>
              {(u.features ?? []).map((f) => (
                <tr key={f.feature}>
                  <th scope="row">{f.feature}</th>
                  <td>{f.uses}</td>
                  <td>{usd(f.usd)}</td>
                </tr>
              ))}
              <tr>
                <th scope="row">{t("Total")}</th>
                <td>{u.total.uses}</td>
                <td>{usd(u.total.usd)}</td>
              </tr>
            </tbody>
          </table>
          {u.limit && (
            <p>
              {t("Personal limit {limit} a month: {under} under 80%, {near} from 80%, {at} at the limit", {
                limit: usd(u.limit.monthly_usd_per_person),
                under: band(u.limit.under_80),
                near: band(u.limit.from_80),
                at: band(u.limit.at_limit),
              })}
            </p>
          )}
          <p>
            <a href="/v1/admin/ai/usage?format=csv" download>
              {t("Download as CSV")}
            </a>
          </p>
        </>
      )}
    </section>
  );
}

// Guardrail insights (sb-observability(7); SB-4406): 12 weeks of counts, never text. A count
// of 1 to 4 comes as "fewer than 5" and is shown in the reader's language.
type GuardCount = number | string;
interface GuardWeek {
  week: string;
  flags: { source: string; space: string; count: GuardCount }[];
  flagged_uploads: GuardCount;
  refused_tools: GuardCount;
  denied_tools: GuardCount;
  dropped_uncited: GuardCount;
}

const shown = (c: GuardCount) => (typeof c === "number" ? String(c) : t("fewer than 5"));
const flagTotal = (w: GuardWeek) =>
  w.flags.some((f) => typeof f.count !== "number") ? t("{n} and more", { n: String(w.flags.reduce((n, f) => n + (typeof f.count === "number" ? f.count : 0), 0)) }) : String(w.flags.reduce((n, f) => n + (f.count as number), 0));

export function AdminGuardrails({ fetcher = fetch }: { fetcher?: Fetch }) {
  const [weeks, setWeeks] = useState<GuardWeek[]>([]);
  const [doubled, setDoubled] = useState<string[]>([]);
  const [error, setError] = useState<string>();
  useEffect(() => {
    call(fetcher, "GET", "/v1/admin/guardrails")
      .then((d) => {
        setWeeks((d.weeks as GuardWeek[]) ?? []);
        setDoubled((d.doubled as string[]) ?? []);
      })
      .catch((e: Error) => setError(e.message));
  }, [fetcher]);
  return (
    <section aria-label={t("Guardrails")}>
      <h2>{t("Guardrails")}</h2>
      {error && <p role="alert">{error}</p>}
      {doubled.length > 0 && <p className="notice">{t("Injection flags doubled last week in: {sources}", { sources: doubled.join(", ") })}</p>}
      {!error && (
        <table>
          <thead>
            <tr>
              <th scope="col">{t("Week")}</th>
              <th scope="col">{t("Injection flags")}</th>
              <th scope="col">{t("Flagged uploads")}</th>
              <th scope="col">{t("Tool calls refused")}</th>
              <th scope="col">{t("Tool calls denied")}</th>
              <th scope="col">{t("Uncited claims dropped")}</th>
            </tr>
          </thead>
          <tbody>
            {weeks.map((w) => (
              <tr key={w.week}>
                <th scope="row">{w.week}</th>
                <td title={w.flags.map((f) => `${f.source} / ${f.space}: ${shown(f.count)}`).join("; ")}>{flagTotal(w)}</td>
                <td>{shown(w.flagged_uploads)}</td>
                <td>{shown(w.refused_tools)}</td>
                <td>{shown(w.denied_tools)}</td>
                <td>{shown(w.dropped_uncited)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </section>
  );
}
