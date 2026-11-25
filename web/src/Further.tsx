import { useId, useRef, useState } from "react";
import type { FormEvent } from "react";
import type { Fetch, Research } from "./api";
import { BRANCH } from "./api";
import { t, tn } from "./i18n";

// Agents that go further (SB-4604 to SB-4606): How this was made under an answer, draft or
// report, worked out as the reader when opened; and the account brief, in a sales or support
// space.

const enc = encodeURIComponent;

// call is one JSON request; a failure throws the cell's message (Steward's call, kept here so
// the answer view doesn't import the steward screens).
export async function call(f: Fetch, method: string, url: string, body?: unknown): Promise<Record<string, unknown>> {
  let res: Response;
  try {
    res = await f(url, { method, credentials: "same-origin", headers: { "Content-Type": "application/json" }, ...(body ? { body: JSON.stringify(body) } : {}) });
  } catch {
    throw new Error(t("The cell can't be reached right now."));
  }
  const data = (await res.json().catch(() => null)) as Record<string, unknown> | null;
  if (!res.ok) {
    const e = (data?.error ?? {}) as { message?: string };
    throw new Error(e.message ?? t("That didn't work."));
  }
  return data ?? {};
}
const msg = (e: unknown, fallback: string) => (e instanceof Error && e.message ? e.message : fallback);
const inSpace = (path: string, space?: string) => (space ? `${path}?space=${enc(space.endsWith(BRANCH) ? space.slice(0, -BRANCH.length) : space)}` : path);

export interface HowMade {
  trace?: string;
  report?: string;
  agent: string;
  release: string;
  skill?: string;
  tier?: string;
  steps?: string[];
  departments: { space: string; agent: string }[];
  not_consulted: { space?: string; why: string; count?: number }[];
  sources: { cite: string; title?: string }[];
  sources_hidden: number;
  refused: number;
  dropped: number;
  cost_usd: number;
}

// How this was made (SB-4606): opened on demand, checked as the reader each time.
export function HowThisWasMade({ trace, report, space, fetcher = fetch }: { trace?: string; report?: string; space?: string; fetcher?: Fetch }) {
  const [how, setHow] = useState<HowMade>();
  const [error, setError] = useState("");
  const asked = useRef(false);
  const open = async () => {
    if (asked.current) return;
    asked.current = true;
    setError("");
    try {
      const path = report ? `/v1/research/${enc(report)}/how` : `/v1/traces/${enc(trace ?? "")}/how`;
      setHow((await call(fetcher, "GET", inSpace(path, report ? space : undefined))) as unknown as HowMade);
    } catch (e) {
      asked.current = false; // try again on the next open
      setError(msg(e, t("That didn't work.")));
    }
  };
  if (!trace && !report) return null;
  return (
    <details className="how" onToggle={(e) => (e.currentTarget as HTMLDetailsElement).open && void open()}>
      <summary>{t("How this was made")}</summary>
      {error && <p role="alert">{error}</p>}
      {how && (
        <dl>
          <dt>{t("Agent")}</dt>
          <dd>
            {how.agent} · {t("release {release}", { release: how.release })}
            {how.tier ? ` · ${t("tier {tier}", { tier: how.tier })}` : ""}
          </dd>
          {how.skill && (
            <>
              <dt>{t("Skill")}</dt>
              <dd>/{how.skill}</dd>
            </>
          )}
          {how.steps && how.steps.length > 0 && (
            <>
              <dt>{t("Steps")}</dt>
              <dd>
                <ol>
                  {how.steps.map((s, i) => (
                    <li key={i}>{s}</li>
                  ))}
                </ol>
              </dd>
            </>
          )}
          {how.departments.length > 0 && (
            <>
              <dt>{t("Teams consulted")}</dt>
              <dd>{how.departments.map((d) => `${d.space} (${d.agent})`).join(", ")}</dd>
            </>
          )}
          {how.not_consulted.length > 0 && (
            <>
              <dt>{t("Not consulted")}</dt>
              <dd>{how.not_consulted.map((n) => (n.space ? `${n.space}: ${n.why}` : n.why)).join("; ")}</dd>
            </>
          )}
          <dt>{t("Sources")}</dt>
          <dd>
            {how.sources.length === 0 ? (
              t("None you can open")
            ) : (
              <ul>
                {how.sources.map((s) => (
                  <li key={s.cite}>{s.title || s.cite}</li>
                ))}
              </ul>
            )}
            {how.sources_hidden > 0 && <p className="meta">{tn(how.sources_hidden, "and {n} you can't open", "and {n} you can't open")}</p>}
          </dd>
          <dt>{t("Checks")}</dt>
          <dd>
            {tn(how.refused, "{n} candidate refused", "{n} candidates refused")} · {tn(how.dropped, "{n} uncited line left out", "{n} uncited lines left out")}
          </dd>
          <dt>{t("Cost")}</dt>
          <dd>${how.cost_usd.toFixed(4)}</dd>
        </dl>
      )}
    </details>
  );
}

export interface AccountFacts {
  name: string;
  domain?: string;
  renewal?: string;
  tickets: number;
  notes: number;
  documents: number;
  owners: string[];
}

// The account brief's facts, counted by code (SB-4605).
export function AccountFactsView({ account }: { account: AccountFacts }) {
  return (
    <p className="meta account">
      {account.domain ? `${account.domain} · ` : ""}
      {account.renewal ? `${t("renews {date}", { date: account.renewal })} · ` : ""}
      {tn(account.tickets, "{n} open ticket", "{n} open tickets")} · {tn(account.notes, "{n} recent note", "{n} recent notes")} ·{" "}
      {tn(account.documents, "{n} contract or proposal", "{n} contracts or proposals")}
      {account.owners.length > 0 && ` · ${t("Ask: {who}", { who: account.owners.join(", ") })}`}
    </p>
  );
}

// Account brief on one CRM account (SB-4605): only in a sales or support space.
export function AccountBriefForm({ space, fetcher = fetch, onMade }: { space?: string; fetcher?: Fetch; onMade: (r: Research) => void }) {
  const id = useId();
  const [account, setAccount] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  if (!space) return null;
  const submit = async (e: FormEvent) => {
    e.preventDefault();
    const a = account.trim();
    if (!a || busy) return;
    setBusy(true);
    setError("");
    try {
      onMade((await call(fetcher, "POST", inSpace("/v1/account-brief", space), { account: a.includes(":") ? a : `crm:${a}` })) as unknown as Research);
      setAccount("");
    } catch (err) {
      setError(msg(err, t("That didn't work.")));
    } finally {
      setBusy(false);
    }
  };
  return (
    <form onSubmit={submit} aria-label={t("Account brief")}>
      <h3>{t("Account brief")}</h3>
      <label htmlFor={id}>{t("CRM account, such as crm:companies/123")}</label>
      <input id={id} value={account} maxLength={200} onChange={(e) => setAccount(e.target.value)} />
      <button type="submit" disabled={busy || account.trim() === ""}>
        {busy ? t("Preparing…") : t("Brief me")}
      </button>
      {error && <p role="alert">{error}</p>}
    </form>
  );
}
