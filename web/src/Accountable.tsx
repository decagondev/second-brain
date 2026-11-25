import { useEffect, useId, useState } from "react";
import type { FormEvent } from "react";
import type { Fetch } from "./api";
import { call } from "./Further";
import { t } from "./i18n";

// Every agent accountable (sb-agents(7), Value, Pause here, Spend caps; SB-4701 to SB-4703):
// what each agent's drafts came to over 12 weeks, and Pause here for a space's stewards and an
// agent's owner.

const enc = encodeURIComponent;

export interface ValueRow {
  agent: string;
  space: string;
  drafts: number;
  published: number;
  published_edited: number;
  discarded: number;
  rounds: number;
  median_hours?: number;
  rated: number;
  helpful_rate?: number;
  cost_usd: number;
  cost_per_published_usd?: number;
  paused?: boolean;
}

export interface AgentSpend {
  agent: string;
  month: string;
  spent_usd: number;
  monthly_usd?: number;
  capped?: boolean;
}

export interface ValueReport {
  weeks: number;
  from: string;
  rows: ValueRow[];
  spend: AgentSpend[];
}

const usd = (n?: number) => (n === undefined ? "-" : `$${n.toFixed(2)}`);
const pct = (n?: number) => (n === undefined ? "-" : `${Math.round(100 * n)}%`);

// PauseHere is one agent's switch in one space: a reason to pause, or Resume once paused.
export function PauseHere({
  space,
  agent,
  paused,
  onDone,
  fetcher = fetch,
}: {
  space: string;
  agent: string;
  paused?: boolean;
  onDone?: (paused: boolean) => void;
  fetcher?: Fetch;
}) {
  const id = useId();
  const [open, setOpen] = useState(false);
  const [reason, setReason] = useState("");
  const [status, setStatus] = useState("");
  const [now, setNow] = useState(!!paused);
  const send = async (on: boolean, e?: FormEvent) => {
    e?.preventDefault();
    setStatus("");
    try {
      await call(
        fetcher,
        "POST",
        `/v1/spaces/${enc(space)}/agents/${enc(agent)}/${on ? "pause" : "resume"}`,
        { reason: reason.trim() },
      );
      setNow(on);
      setOpen(false);
      setReason("");
      setStatus(
        on
          ? t("{agent} is paused in {space}: it drafts nothing more here.", {
              agent,
              space,
            })
          : t("{agent} runs in {space} again.", { agent, space }),
      );
      onDone?.(on);
    } catch (err) {
      setStatus(
        err instanceof Error && err.message
          ? err.message
          : t("That didn't work."),
      );
    }
  };
  return (
    <span className="pause-here">
      {now ? (
        <button type="button" onClick={() => void send(false)}>
          {t("Resume")}
        </button>
      ) : open ? (
        <form onSubmit={(e) => void send(true, e)}>
          <label htmlFor={`${id}-why`}>
            {t("Why pause {agent} here?", { agent })}
          </label>{" "}
          <input
            id={`${id}-why`}
            value={reason}
            maxLength={300}
            required
            onChange={(e) => setReason(e.target.value)}
          />{" "}
          <button type="submit" disabled={!reason.trim()}>
            {t("Pause")}
          </button>{" "}
          <button type="button" onClick={() => setOpen(false)}>
            {t("Cancel")}
          </button>
        </form>
      ) : (
        <button type="button" onClick={() => setOpen(true)}>
          {t("Pause here")}
        </button>
      )}
      {status && <span role="status"> {status}</span>}
    </span>
  );
}

// AgentValue is the value table: every agent for operators (admin), or the agents a person
// owns, with Pause here on each space's row. An owner with no agents sees nothing.
export function AgentValue({
  admin,
  fetcher = fetch,
}: {
  admin?: boolean;
  fetcher?: Fetch;
}) {
  const [v, setV] = useState<ValueReport>();
  const [error, setError] = useState("");
  useEffect(() => {
    call(
      fetcher,
      "GET",
      admin ? "/v1/admin/agents/value" : "/v1/agents/value",
    ).then(
      (d) =>
        setV({
          weeks: Number(d.weeks ?? 12),
          from: String(d.from ?? ""),
          rows: Array.isArray(d.rows) ? (d.rows as ValueRow[]) : [],
          spend: Array.isArray(d.spend) ? (d.spend as AgentSpend[]) : [],
        }),
      (e: Error) => setError(e.message),
    );
  }, [admin, fetcher]);
  if (!admin && !error && (!v || (v.rows.length === 0 && v.spend.length === 0)))
    return null;
  return (
    <section aria-label={t("What agents' drafts came to")}>
      <h2>
        {admin
          ? t("What agents' drafts came to")
          : t("What your agents' drafts came to")}
      </h2>
      {error && <p role="alert">{error}</p>}
      {v && v.rows.length === 0 && (
        <p className="notice">
          {t("No agent drafted anything in the last {n} weeks.", {
            n: v.weeks,
          })}
        </p>
      )}
      {v && v.rows.length > 0 && (
        <table>
          <caption>
            {t("The last {n} weeks, since {from}", {
              n: v.weeks,
              from: v.from,
            })}
          </caption>
          <thead>
            <tr>
              <th scope="col">{t("Agent")}</th>
              <th scope="col">{t("Space")}</th>
              <th scope="col">{t("Drafts")}</th>
              <th scope="col">{t("Published")}</th>
              <th scope="col">{t("Published edited")}</th>
              <th scope="col">{t("Discarded")}</th>
              <th scope="col">{t("Rounds of changes")}</th>
              <th scope="col">{t("Median hours to reply")}</th>
              <th scope="col">{t("Rated helpful")}</th>
              <th scope="col">{t("Cost")}</th>
              <th scope="col">{t("Cost per published")}</th>
              <th scope="col">{t("State")}</th>
            </tr>
          </thead>
          <tbody>
            {v.rows.map((r) => (
              <tr key={r.agent + "@" + r.space}>
                <th scope="row">{r.agent}</th>
                <td>{r.space}</td>
                <td>{r.drafts}</td>
                <td>{r.published}</td>
                <td>{r.published_edited}</td>
                <td>{r.discarded}</td>
                <td>{r.rounds}</td>
                <td>
                  {r.median_hours === undefined
                    ? "-"
                    : r.median_hours.toFixed(1)}
                </td>
                <td>
                  {r.rated === 0 ? "-" : `${pct(r.helpful_rate)} (${r.rated})`}
                </td>
                <td>{usd(r.cost_usd)}</td>
                <td>{usd(r.cost_per_published_usd)}</td>
                <td>
                  {admin ? (
                    r.paused ? (
                      t("paused")
                    ) : (
                      t("running")
                    )
                  ) : (
                    <PauseHere
                      space={r.space}
                      agent={r.agent}
                      paused={r.paused}
                      fetcher={fetcher}
                    />
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
      {v && v.spend.length > 0 && (
        <table>
          <caption>{t("Spend this month")}</caption>
          <thead>
            <tr>
              <th scope="col">{t("Agent")}</th>
              <th scope="col">{t("Spent")}</th>
              <th scope="col">{t("Monthly cap")}</th>
            </tr>
          </thead>
          <tbody>
            {v.spend.map((s) => (
              <tr key={s.agent}>
                <th scope="row">{s.agent}</th>
                <td>{usd(s.spent_usd)}</td>
                <td>
                  {s.monthly_usd === undefined
                    ? t("none")
                    : s.capped
                      ? t("{cap}: at its cap until next month", {
                          cap: usd(s.monthly_usd),
                        })
                      : usd(s.monthly_usd)}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </section>
  );
}

export interface ShadowSideView {
  chars: number;
  claims: number;
  cited: number;
  cites: number;
  dropped: number;
  cost_usd: number;
}

export interface ShadowItem {
  agent: string;
  release: string;
  prompt: string;
  from: string;
  until: string;
  runs: {
    id: string;
    at: string;
    space: string;
    current: ShadowSideView;
    shadow: ShadowSideView;
    text?: { current: string; shadow: string };
  }[];
}

// ShadowRuns (SB-4704) is an owner's open shadow windows: each run's numbers side by side,
// and the two texts only when the owner can open every cite of both. Nothing for no window.
export function ShadowRuns({ fetcher = fetch }: { fetcher?: Fetch }) {
  const [items, setItems] = useState<ShadowItem[]>([]);
  useEffect(() => {
    call(fetcher, "GET", "/v1/agents/shadow").then(
      (d) => setItems(Array.isArray(d.items) ? (d.items as ShadowItem[]) : []),
      () => setItems([]),
    );
  }, [fetcher]);
  if (items.length === 0) return null;
  const side = (s: ShadowSideView) =>
    t(
      "{chars} characters, {cited} of {claims} claims cited, {cites} sources, {dropped} dropped, {cost}",
      {
        chars: s.chars,
        cited: s.cited,
        claims: s.claims,
        cites: s.cites,
        dropped: s.dropped,
        cost: usd(s.cost_usd),
      },
    );
  return (
    <section aria-label={t("Shadow runs")}>
      <h2>{t("Shadow runs")}</h2>
      <p className="meta">
        {t("Never delivered to anyone, and deleted when the window ends.")}
      </p>
      {items.map((it) => (
        <div key={it.agent + it.release}>
          <h3>
            {t("{agent} with {prompt}, until {until}", {
              agent: it.agent,
              prompt: it.prompt,
              until: it.until.slice(0, 10),
            })}
          </h3>
          {it.runs.length === 0 && (
            <p className="notice">{t("No runs yet.")}</p>
          )}
          <ul className="saved-list">
            {it.runs.map((r) => (
              <li key={r.id}>
                <span className="meta">
                  {r.space}, {r.at.slice(0, 10)}
                </span>
                <p>{t("Current: {side}", { side: side(r.current) })}</p>
                <p>{t("Shadow: {side}", { side: side(r.shadow) })}</p>
                {r.text ? (
                  <div className="report">
                    <pre className="payload">{r.text.current}</pre>
                    <pre className="payload">{r.text.shadow}</pre>
                  </div>
                ) : (
                  <p className="meta">
                    {t(
                      "Numbers only: you can't open every source of this run.",
                    )}
                  </p>
                )}
              </li>
            ))}
          </ul>
        </div>
      ))}
    </section>
  );
}
