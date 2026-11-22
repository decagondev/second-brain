import { useEffect, useId, useState } from "react";
import type { Fetch } from "./api";
import { StewardQuestions } from "./Questions";

export interface ReviewItem {
  user: string;
  via: string[];
  last_use: string;
  decision?: string;
  by?: string;
  reason?: string;
}

export interface ReviewRequest {
  kind: string;
  user: string;
  group: string;
  proposal?: string;
}

export interface Review {
  id: string;
  space: string;
  quarter: string;
  due: string;
  state: string;
  overdue?: boolean;
  items: ReviewItem[];
  requests: ReviewRequest[] | null;
}

interface Proposed {
  id: string;
  hash: string;
  diff: string[];
  plan: string;
}

// call sends one request and returns its JSON, or throws the cell's own message and the
// manifest errors it lists, if any.
export async function call(f: Fetch, method: string, url: string, body?: unknown): Promise<Record<string, unknown>> {
  let res: Response;
  try {
    res = await f(url, { method, credentials: "same-origin", headers: { "Content-Type": "application/json" }, ...(body ? { body: JSON.stringify(body) } : {}) });
  } catch {
    throw new Error("The cell can't be reached right now.");
  }
  const data = (await res.json().catch(() => null)) as Record<string, unknown> | null;
  if (!res.ok) {
    const e = (data?.error ?? {}) as { message?: string; errors?: { message?: string; field?: string }[] };
    const why = (e.errors ?? []).map((x) => `${x.field ? x.field + ": " : ""}${x.message ?? ""}`).filter(Boolean);
    throw new Error([e.message ?? "That didn't work.", ...why].join(" "));
  }
  return data ?? {};
}

// The steward's screen for one space (sb-spaces(7), Stewards in the web UI; SB-1509): a
// proposal of the space's new definition, which only an operator can approve, and the
// space's access reviews, decided one person at a time.
export function StewardView({ space, fetcher = fetch }: { space: string; fetcher?: Fetch }) {
  const [definition, setDefinition] = useState('{\n  "members": ["group:..."],\n  "stewards": ["group:..."]\n}');
  const [proposed, setProposed] = useState<Proposed>();
  const [reviews, setReviews] = useState<Review[]>([]);
  const [status, setStatus] = useState<string>();
  const [removing, setRemoving] = useState<{ review: string; user: string }>();
  const [reason, setReason] = useState("");
  const ids = { def: useId(), reason: useId() };
  const base = `/v1/spaces/${encodeURIComponent(space)}`;

  async function load() {
    try {
      const r = await call(fetcher, "GET", `${base}/reviews`);
      setReviews((r.items as Review[]) ?? []);
    } catch (e) {
      setStatus((e as Error).message);
    }
  }
  useEffect(() => {
    void load();
  }, [space]);

  async function propose() {
    let parsed: unknown;
    try {
      parsed = JSON.parse(definition);
    } catch {
      setStatus("The definition must be JSON, in the fields of sb-space(5).");
      return;
    }
    try {
      const p = await call(fetcher, "POST", `${base}/proposals`, { space: parsed });
      setProposed({ id: String(p.id), hash: String(p.hash), diff: (p.diff as string[]) ?? [], plan: String(p.plan ?? "") });
      setStatus("Proposed. Nothing changes until an operator approves this hash and the manifest is applied.");
    } catch (e) {
      setProposed(undefined);
      setStatus((e as Error).message);
    }
  }

  async function decide(review: string, user: string, decision: "certify" | "remove", why = "") {
    if (decision === "remove" && !why.trim()) {
      setRemoving({ review, user });
      setStatus("A removal needs a reason; it goes into the audit log.");
      return;
    }
    try {
      const next = (await call(fetcher, "POST", `${base}/reviews/${encodeURIComponent(review)}/decide`, { user, decision, reason: why.trim() })) as unknown as Review;
      setReviews((rs) => rs.map((r) => (r.id === next.id ? next : r)));
      setRemoving(undefined);
      setReason("");
      setStatus(next.state === "complete" ? "The review is complete. Its change requests are listed below." : `Recorded ${decision} for ${user}.`);
    } catch (e) {
      setStatus((e as Error).message);
    }
  }

  return (
    <main>
      <h1>Space {space}</h1>
      <section aria-label="Propose a change">
        <h2>Propose a change</h2>
        <label htmlFor={ids.def}>The space's new definition (JSON)</label>
        <textarea id={ids.def} rows={8} value={definition} onChange={(e) => setDefinition(e.target.value)} />
        <button type="button" onClick={propose}>
          Propose
        </button>
        {proposed && (
          <div>
            <h3>Proposal {proposed.id}</h3>
            <p>
              An operator approves hash <code>{proposed.hash}</code>.
            </p>
            <pre className="payload">{proposed.diff.join("\n")}</pre>
            <pre className="payload">{proposed.plan}</pre>
          </div>
        )}
      </section>
      <StewardQuestions space={space} fetcher={fetcher} />
      <section aria-label="Access reviews">
        <h2>Access reviews</h2>
        {reviews.length === 0 ? (
          <p>No access reviews for this space.</p>
        ) : (
          reviews.map((r) => (
            <div key={r.id}>
              <h3>
                {r.quarter}: {r.state}
                {r.overdue ? " (overdue)" : ""}, due {r.due}
              </h3>
              <table className="diff">
                <caption className="visually-hidden">People with access to {r.space}</caption>
                <thead>
                  <tr>
                    <th scope="col">Person</th>
                    <th scope="col">Via</th>
                    <th scope="col">Last use</th>
                    <th scope="col">Decision</th>
                  </tr>
                </thead>
                <tbody>
                  {r.items.map((it) => (
                    <tr key={it.user}>
                      <th scope="row">{it.user}</th>
                      <td>{it.via.join(", ")}</td>
                      <td>{it.last_use}</td>
                      <td>
                        {it.decision ? (
                          `${it.decision} by ${it.by ?? "?"}${it.reason ? ": " + it.reason : ""}`
                        ) : (
                          <>
                            <button type="button" onClick={() => decide(r.id, it.user, "certify")} aria-label={`Certify ${it.user}`}>
                              Certify
                            </button>{" "}
                            <button type="button" onClick={() => decide(r.id, it.user, "remove", removing?.user === it.user ? reason : "")} aria-label={`Remove ${it.user}`}>
                              Remove
                            </button>
                          </>
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
              {r.requests && r.requests.length > 0 && (
                <ul aria-label="Change requests">
                  {r.requests.map((q) => (
                    <li key={`${q.user}/${q.group}`}>
                      {q.kind === "manifest" ? `Manifest proposal ${q.proposal}` : "IdP change"}: take {q.user} out of {q.group}
                    </li>
                  ))}
                </ul>
              )}
            </div>
          ))
        )}
        {removing && (
          <div>
            <label htmlFor={ids.reason}>Reason for removing {removing.user} (required)</label>
            <input id={ids.reason} type="text" value={reason} onChange={(e) => setReason(e.target.value)} />
            <button type="button" onClick={() => decide(removing.review, removing.user, "remove", reason)}>
              Confirm removal of {removing.user}
            </button>
          </div>
        )}
      </section>
      {status && <p role="status">{status}</p>}
    </main>
  );
}
