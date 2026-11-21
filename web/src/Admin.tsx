import { useEffect, useId, useState } from "react";
import type { Fetch } from "./api";
import type { Review } from "./Steward";

interface Proposal {
  id: string;
  space: string;
  requester: string;
  hash: string;
  diff: string[];
}

interface Switch {
  scope: string;
  reason: string;
  actor: string;
}

async function call(f: Fetch, method: string, url: string, body?: unknown) {
  const res = await f(url, { method, credentials: "same-origin", headers: { "Content-Type": "application/json" }, ...(body ? { body: JSON.stringify(body) } : {}) });
  const data = (await res.json().catch(() => null)) as Record<string, unknown> | null;
  if (!res.ok) throw new Error(((data?.error as { message?: string })?.message ?? "That didn't work.") + "");
  return data ?? {};
}

// The admin console (sb-admin(3); SB-602). It is a client of the admin API and has no
// other powers. Kill switches act at once; making one permanent is a manifest diff.
export function AdminConsole({ fetcher = fetch }: { fetcher?: Fetch }) {
  const [manifest, setManifest] = useState("");
  const [switches, setSwitches] = useState<Switch[]>([]);
  const [diff, setDiff] = useState<string[]>();
  const [status, setStatus] = useState<string>();
  const [scope, setScope] = useState("agent");
  const [name, setName] = useState("");
  const [reason, setReason] = useState("");
  const [proposals, setProposals] = useState<Proposal[]>([]);
  const [reviews, setReviews] = useState<Review[]>([]);
  const [applyText, setApplyText] = useState<string>();
  const [rejectReason, setRejectReason] = useState("");
  const ids = { scope: useId(), name: useId(), reason: useId(), reject: useId() };

  async function load() {
    try {
      const m = await call(fetcher, "GET", "/v1/admin/manifest");
      setManifest(String(m.manifest ?? ""));
      const k = await call(fetcher, "GET", "/v1/admin/kill");
      setSwitches((k.switches as Switch[]) ?? []);
      const p = await call(fetcher, "GET", "/v1/admin/proposals");
      setProposals((p.items as Proposal[]) ?? []);
      const r = await call(fetcher, "GET", "/v1/admin/reviews");
      const rs = ((r.items as Review[]) ?? []).slice();
      rs.sort((a, b) => Number(!!b.overdue) - Number(!!a.overdue) || a.id.localeCompare(b.id));
      setReviews(rs);
    } catch (e) {
      setStatus((e as Error).message);
    }
  }
  useEffect(() => {
    void load();
  }, []);

  async function kill(set: boolean, s = scope, n = name) {
    if (!reason.trim()) {
      setStatus("A reason is required; it goes into the audit log.");
      return;
    }
    try {
      await call(fetcher, "POST", set ? "/v1/admin/kill" : "/v1/admin/unkill", { scope: s, name: n, reason: reason.trim() });
      setStatus(set ? `Killed ${s} ${n}. Cells stop within 10 seconds.` : `Cleared ${s} ${n}.`);
      await load();
    } catch (e) {
      setStatus((e as Error).message);
    }
  }

  // An approval sends back exactly the hash on screen: a proposal that changed since is
  // refused by the cell, never approved unseen (sb-spaces(7), Stewards).
  async function approve(p: Proposal) {
    try {
      const a = await call(fetcher, "POST", `/v1/admin/proposals/${encodeURIComponent(p.id)}/approve`, { hash: p.hash });
      setApplyText(String(a.manifest ?? ""));
      setStatus(`Approved ${p.id}. Apply the manifest below with sb tenant apply to make it so.`);
      await load();
    } catch (e) {
      setStatus((e as Error).message);
    }
  }

  async function reject(p: Proposal) {
    if (!rejectReason.trim()) {
      setStatus("A rejection needs a reason; it goes into the audit log.");
      return;
    }
    try {
      await call(fetcher, "POST", `/v1/admin/proposals/${encodeURIComponent(p.id)}/reject`, { reason: rejectReason.trim() });
      setStatus(`Rejected ${p.id}.`);
      setRejectReason("");
      await load();
    } catch (e) {
      setStatus((e as Error).message);
    }
  }

  async function makePermanent(sw: Switch) {
    const [kind, n] = sw.scope.split(":");
    const path = kind === "agent" ? `/agents/${n}` : `/connectors/${n}`;
    try {
      const d = await call(fetcher, "POST", "/v1/admin/manifest/diff", [{ op: "remove", path }]);
      setDiff(d.diff as string[]);
      setStatus("This is a proposed manifest change. Apply it with sb tenant apply to make it permanent.");
    } catch (e) {
      setStatus((e as Error).message);
    }
  }

  return (
    <main>
      <h1>Admin console</h1>
      <section aria-label="Kill switches">
        <h2>Kill switches</h2>
        {switches.length === 0 ? (
          <p>No kill switches are set.</p>
        ) : (
          <ul>
            {switches.map((s) => (
              <li key={s.scope}>
                <strong>{s.scope}</strong>: {s.reason} (by {s.actor}){" "}
                <button type="button" onClick={() => kill(false, ...(s.scope === "tenant" ? ["tenant", ""] : (s.scope.split(":") as [string, string])))}>
                  Clear
                </button>{" "}
                {s.scope !== "tenant" && (
                  <button type="button" onClick={() => makePermanent(s)}>
                    Make permanent
                  </button>
                )}
              </li>
            ))}
          </ul>
        )}
        <div className="kill-form">
          <label htmlFor={ids.scope}>Scope</label>
          <select id={ids.scope} value={scope} onChange={(e) => setScope(e.target.value)}>
            <option value="agent">agent</option>
            <option value="connector">connector</option>
            <option value="tenant">tenant</option>
          </select>
          <label htmlFor={ids.name}>Name</label>
          <input id={ids.name} type="text" value={name} onChange={(e) => setName(e.target.value)} disabled={scope === "tenant"} />
          <label htmlFor={ids.reason}>Reason (required)</label>
          <input id={ids.reason} type="text" value={reason} onChange={(e) => setReason(e.target.value)} />
          <button type="button" onClick={() => kill(true)}>
            Kill now
          </button>
        </div>
      </section>
      <section aria-label="Steward proposals">
        <h2>Steward proposals</h2>
        {proposals.length === 0 ? (
          <p>No open proposals.</p>
        ) : (
          <>
            <label htmlFor={ids.reject}>Reason, to reject</label>
            <input id={ids.reject} type="text" value={rejectReason} onChange={(e) => setRejectReason(e.target.value)} />
            {proposals.map((p) => (
              <article key={p.id} aria-label={`Proposal ${p.id}`}>
                <h3>
                  {p.space}: {p.id} by {p.requester}
                </h3>
                <pre className="payload">{p.diff.join("\n")}</pre>
                <p>
                  Hash <code>{p.hash}</code>
                </p>
                <button type="button" onClick={() => approve(p)}>
                  Approve hash {p.hash.slice(0, 12)}
                </button>{" "}
                <button type="button" onClick={() => reject(p)}>
                  Reject {p.id}
                </button>
              </article>
            ))}
          </>
        )}
        {applyText && <pre className="payload" aria-label="Manifest to apply">{applyText}</pre>}
      </section>
      <section aria-label="Access reviews">
        <h2>Access reviews</h2>
        {reviews.length === 0 ? (
          <p>No access reviews.</p>
        ) : (
          <ul>
            {reviews.map((r) => (
              <li key={r.id}>
                <strong>
                  {r.space} {r.quarter}
                </strong>
                : {r.state}
                {r.overdue ? " (overdue)" : ""}, due {r.due} <a href={`/v1/admin/reviews/${encodeURIComponent(r.id)}/evidence`}>Evidence for {r.space} {r.quarter}</a>
              </li>
            ))}
          </ul>
        )}
      </section>
      {diff && (
        <section aria-label="Proposed manifest change">
          <h2>Proposed manifest change</h2>
          <pre className="payload">{diff.join("\n")}</pre>
        </section>
      )}
      <section aria-label="Applied manifest">
        <h2>Applied manifest</h2>
        <pre className="payload">{manifest}</pre>
      </section>
      {status && <p role="status">{status}</p>}
    </main>
  );
}
