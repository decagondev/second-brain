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
    <section aria-label={t("Team updates")} className="start">
      <h2>{t("Team updates")}</h2>
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
