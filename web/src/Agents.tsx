import { useEffect, useState } from "react";
import type { Fetch } from "./api";
import { call } from "./Steward";
import { t } from "./i18n";

// One agent as the registry counts it (sb-admin(3), GET /v1/admin/agents; SB-4004).
export interface AgentRow {
  name: string;
  department?: string;
  spaces: string[];
  runs: number;
  rated: number;
  unhelpful: number;
  eval: string;
  suite: string;
  killed: boolean;
}

function share(a: AgentRow): string {
  return a.rated === 0 ? "-" : `${Math.round((100 * a.unhelpful) / a.rated)}%`;
}

export function AgentsTable({ items }: { items: AgentRow[] }) {
  return (
    <table>
      <caption>{t("Runs in the last 30 days")}</caption>
      <thead>
        <tr>
          <th scope="col">{t("Agent")}</th>
          <th scope="col">{t("Department")}</th>
          <th scope="col">{t("Spaces")}</th>
          <th scope="col">{t("Runs")}</th>
          <th scope="col">{t("Rated unhelpful")}</th>
          <th scope="col">{t("Eval")}</th>
          <th scope="col">{t("State")}</th>
        </tr>
      </thead>
      <tbody>
        {items.map((a) => (
          <tr key={a.name}>
            <th scope="row">{a.name}</th>
            <td>{a.department ?? t("every space")}</td>
            <td>{a.spaces.join(", ")}</td>
            <td>{a.runs}</td>
            <td>{share(a)}</td>
            <td>{a.eval === "held" ? t("held ({suite})", { suite: a.suite }) : t("passed ({suite})", { suite: a.suite })}</td>
            <td>{a.killed ? t("killed") : t("running")}</td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}

// Every agent in one place, for operators.
export function AdminAgents({ fetcher = fetch }: { fetcher?: Fetch }) {
  const [items, setItems] = useState<AgentRow[]>([]);
  const [error, setError] = useState<string>();
  useEffect(() => {
    call(fetcher, "GET", "/v1/admin/agents")
      .then((d) => setItems((d.agents as AgentRow[]) ?? []))
      .catch((e: Error) => setError(e.message));
  }, [fetcher]);
  return (
    <section aria-label={t("Agents")}>
      <h2>{t("Agents")}</h2>
      {error ? <p role="alert">{error}</p> : <AgentsTable items={items} />}
    </section>
  );
}

// A space's agents, for its stewards, with a proposal to change which department agents
// it lists: an operator approves the exact manifest diff.
export function SpaceAgents({ base, fetcher = fetch }: { base: string; fetcher?: Fetch }) {
  const [items, setItems] = useState<AgentRow[]>([]);
  const [can, setCan] = useState<string[]>([]);
  const [listed, setListed] = useState<string[]>([]);
  const [status, setStatus] = useState<string>();
  const [diff, setDiff] = useState<string[]>([]);
  useEffect(() => {
    call(fetcher, "GET", `${base}/agents`)
      .then((d) => {
        setItems((d.agents as AgentRow[]) ?? []);
        setCan((d.can_list as string[]) ?? []);
        setListed((d.listed as string[]) ?? []);
      })
      .catch((e: Error) => setStatus(e.message));
  }, [base, fetcher]);
  function toggle(n: string) {
    setListed((l) => (l.includes(n) ? l.filter((x) => x !== n) : [...l, n]));
  }
  async function propose() {
    try {
      const p = await call(fetcher, "POST", `${base}/proposals/agents`, { agents: listed });
      setDiff((p.diff as string[]) ?? []);
      setStatus(t("Proposal {id} made. Nothing changes until an operator approves it and the manifest is applied.", { id: String(p.id) }));
    } catch (e) {
      setStatus((e as Error).message);
    }
  }
  return (
    <section aria-label={t("Agents")}>
      <h2>{t("Agents")}</h2>
      <AgentsTable items={items} />
      {can.length > 0 && (
        <fieldset>
          <legend>{t("Department agents this space lists")}</legend>
          {can.map((n) => (
            <label key={n}>
              <input type="checkbox" checked={listed.includes(n)} onChange={() => toggle(n)} /> {n}
            </label>
          ))}
          <button type="button" onClick={propose}>
            {t("Propose these agents")}
          </button>
        </fieldset>
      )}
      {diff.length > 0 && <pre className="payload">{diff.join("\n")}</pre>}
      {status && <p role="status">{status}</p>}
    </section>
  );
}
