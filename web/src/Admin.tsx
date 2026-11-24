import { useEffect, useId, useState } from "react";
import type { Fetch } from "./api";
import type { Review } from "./Steward";
import { t, tj, tn, msg } from "./i18n";
import { AdminTokens } from "./AdminTokens";
import { AdminAgents } from "./Agents";
import { AdminSpend } from "./Company";

interface Proposal {
  id: string;
  space: string;
  requester: string;
  hash: string;
  diff: string[];
}

// Why the last sync failed, by its class (SB-3201); a class it doesn't know is a source error.
const sourceError = msg("The source answered with an error.");
const failureText: Record<string, string> = {
  auth: msg("The source refused the connector's credentials."),
  rate_limited: msg("The source asked the connector to slow down."),
  unreachable: msg("The source didn't answer."),
  source_error: sourceError,
};

// The cell's health (sb-admin(3), GET /v1/admin/health; SB-1607).
export interface Health {
  jobs: { name: string; law?: string; last_finish: string | null; overdue: boolean; unreadable?: boolean }[];
  // failure is the last failed sync's class and fix its runbook section (SB-3201).
  connectors: { kind: string; last_success: string | null; last_failure: string | null; failures: number; stale: boolean; failure?: string; fix?: string }[];
  at: string;
  // The machine on a self-hosted cell (SB-1802); absent on a managed one.
  box?: { facts_at: string | null; verdict: string; checks: { check: string; verdict: string; detail: string; fix?: string }[] };
  notify?: { last_run: string | null; via: string; sent: number; skipped: number; failed: number; warning: boolean; reason?: string };
  // The index's leftovers (SB-2905) and input tokens per model call (SB-2906).
  index?: { at: string | null; gone: number; unheld: number; warning: boolean };
  prompt?: { p95: number; p95_before: number; warning: boolean };
  // Spaces with no active steward, and those the directory couldn't answer for (SB-3102).
  spaces_unstewarded?: { spaces: string[]; unknown: string[]; warning: boolean };
}

// One answer in the unhelpful-answer queue (SB-1609): counts, decisions and the asker's own
// comment, never the question or the answer.
export interface QueueItem {
  trace: string;
  at: string;
  user: string;
  rating: string;
  wrong_citation?: string;
  comment?: string;
  claims: number;
  uncited: number;
  refused: number;
  decisions: { verdict: string; reason: string; count: number }[];
  reviewed?: { by: string; at: string };
}

// The notify job's warnings as the cell words them (health.NotifyFrom; SB-2607).
msg("the notify job hasn't run");
msg("the last run failed");
msg("no run in the last hour");

function when(at: string | null): string {
  return at ? new Date(at).toISOString().replace("T", " ").slice(0, 16) + " UTC" : t("never");
}

interface Switch {
  scope: string;
  reason: string;
  actor: string;
}

async function call(f: Fetch, method: string, url: string, body?: unknown) {
  const res = await f(url, { method, credentials: "same-origin", headers: { "Content-Type": "application/json" }, ...(body ? { body: JSON.stringify(body) } : {}) });
  const data = (await res.json().catch(() => null)) as Record<string, unknown> | null;
  if (!res.ok) throw new Error(((data?.error as { message?: string })?.message ?? t("That didn't work.")) + "");
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
  const [health, setHealth] = useState<Health>();
  const [healthError, setHealthError] = useState<string>();
  const [queue, setQueue] = useState<QueueItem[]>([]);
  const [queueError, setQueueError] = useState<string>();
  const ids = { scope: useId(), name: useId(), reason: useId(), reject: useId() };

  // Health and the queue load on their own, so one the cell can't answer never hides the rest.
  async function loadHealth() {
    try {
      const h = (await call(fetcher, "GET", "/v1/admin/health")) as Partial<Health>;
      setHealth({ jobs: h.jobs ?? [], connectors: h.connectors ?? [], at: h.at ?? "", box: h.box, notify: h.notify, index: h.index, prompt: h.prompt, spaces_unstewarded: h.spaces_unstewarded });
      setHealthError(undefined);
    } catch (e) {
      setHealth(undefined);
      setHealthError((e as Error).message);
    }
  }
  async function loadQueue() {
    try {
      const q = await call(fetcher, "GET", "/v1/admin/feedback?rating=down");
      setQueue((q.items as QueueItem[]) ?? []);
      setQueueError(undefined);
    } catch (e) {
      setQueueError((e as Error).message);
    }
  }
  async function reviewed(item: QueueItem) {
    try {
      await call(fetcher, "POST", `/v1/admin/feedback/${encodeURIComponent(item.trace)}/reviewed`);
      setStatus(t("Marked {trace} reviewed.", { trace: item.trace }));
      await loadQueue();
    } catch (e) {
      setStatus((e as Error).message);
    }
  }

  async function load() {
    void loadHealth();
    void loadQueue();
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
      setStatus(t("A reason is required; it goes into the audit log."));
      return;
    }
    try {
      await call(fetcher, "POST", set ? "/v1/admin/kill" : "/v1/admin/unkill", { scope: s, name: n, reason: reason.trim() });
      setStatus(set ? t("Killed {scope} {name}. Cells stop within 10 seconds.", { scope: s, name: n }) : t("Cleared {scope} {name}.", { scope: s, name: n }));
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
      setStatus(t("Approved {id}. Apply the manifest below with {command} to make it so.", { id: p.id, command: "sb tenant apply" }));
      await load();
    } catch (e) {
      setStatus((e as Error).message);
    }
  }

  async function reject(p: Proposal) {
    if (!rejectReason.trim()) {
      setStatus(t("A rejection needs a reason; it goes into the audit log."));
      return;
    }
    try {
      await call(fetcher, "POST", `/v1/admin/proposals/${encodeURIComponent(p.id)}/reject`, { reason: rejectReason.trim() });
      setStatus(t("Rejected {id}.", { id: p.id }));
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
      setStatus(t("This is a proposed manifest change. Apply it with {command} to make it permanent.", { command: "sb tenant apply" }));
    } catch (e) {
      setStatus((e as Error).message);
    }
  }

  return (
    <main>
      <h1>{t("Admin console")}</h1>
      <section aria-label={t("Cell health")}>
        <h2>{t("Cell health")}</h2>
        {healthError ? (
          <p className="error">{t("Health can't be shown: {error}", { error: healthError })}</p>
        ) : !health ? (
          <p>{t("Loading health…")}</p>
        ) : (
          <>
            <h3>{t("Connectors")}</h3>
            {health.connectors.length === 0 ? (
              <p>{t("No connectors.")}</p>
            ) : (
              <ul>
                {health.connectors.map((c) => (
                  <li key={c.kind} className={c.stale ? "attention" : undefined}>
                    <strong>{c.kind}</strong>: {t(c.stale ? "stale, last synced {at}" : "fresh, last synced {at}", { at: when(c.last_success) })}
                    {c.failures > 0 ? `, ${tn(c.failures, "{n} failed sync since (last {at})", "{n} failed syncs since (last {at})", { at: when(c.last_failure) })}` : ""}
                    {c.failure ? `. ${t(failureText[c.failure] ?? sourceError)}${c.fix ? " " + t("See {fix}.", { fix: c.fix }) : ""}` : ""}
                  </li>
                ))}
              </ul>
            )}
            <h3>{t("Scheduled jobs")}</h3>
            <ul>
              {health.jobs.map((j) => (
                <li key={j.name} className={j.overdue ? "attention" : undefined}>
                  <strong>{j.name}</strong>
                  {j.law ? ` (${j.law})` : ""}:{" "}
                  {t(j.overdue ? (j.unreadable ? "overdue, its record can't be read, last finished {at}" : "overdue, last finished {at}") : "on time, last finished {at}", {
                    at: when(j.last_finish),
                  })}
                </li>
              ))}
            </ul>
            {health.notify ? (
              <>
                <h3>{t("Notifications")}</h3>
                <p className={health.notify.warning ? "attention" : undefined}>
                  {health.notify.last_run
                    ? t("Last run {at} by {via}: {sent} sent, {skipped} skipped, {failed} failed.", {
                        at: when(health.notify.last_run),
                        via: health.notify.via,
                        sent: String(health.notify.sent),
                        skipped: String(health.notify.skipped),
                        failed: String(health.notify.failed),
                      })
                    : t("The notify job hasn't run yet.")}
                  {health.notify.warning && health.notify.reason ? ` ${t("Warning: {reason}.", { reason: t(health.notify.reason) })}` : ""}
                </p>
              </>
            ) : null}
            {health.index && (
              <>
                <h3>{t("Index")}</h3>
                <p className={health.index.warning ? "attention" : undefined}>
                  {health.index.at
                    ? t("Counted {at}: {gone} documents their source no longer lists, {unheld} no space holds. sb index prune --dry-run lists them.", {
                        at: when(health.index.at),
                        gone: String(health.index.gone),
                        unheld: String(health.index.unheld),
                      })
                    : t("The index's leftovers haven't been counted yet.")}
                </p>
              </>
            )}
            {health.prompt && (
              <>
                <h3>{t("Prompt size")}</h3>
                <p className={health.prompt.warning ? "attention" : undefined}>
                  {t("Input tokens per model call, 95th percentile: {now} this week, {before} the week before.", {
                    now: String(health.prompt.p95),
                    before: String(health.prompt.p95_before),
                  })}
                  {health.prompt.warning ? ` ${t("Growing: over half again last week's.")}` : ""}
                </p>
              </>
            )}
            {health.spaces_unstewarded?.warning && (
              <>
                <h3>{t("Stewards")}</h3>
                {health.spaces_unstewarded.spaces.length > 0 && (
                  <p className="attention">{t("Spaces with no active steward: {spaces}", { spaces: health.spaces_unstewarded.spaces.join(", ") })}</p>
                )}
                {health.spaces_unstewarded.unknown.length > 0 && (
                  <p className="attention">{t("Couldn't check stewards for: {spaces}", { spaces: health.spaces_unstewarded.unknown.join(", ") })}</p>
                )}
              </>
            )}
            <h3>{t("Box")}</h3>
            {!health.box ? (
              <p>{t("Managed by the cloud: no machine to check.")}</p>
            ) : (
              <>
                <p>
                  {health.box.verdict === "pass"
                    ? t("Every check passes, from facts gathered {at}.", { at: when(health.box.facts_at) })
                    : t("Worst check: {verdict}, from facts gathered {at}.", { verdict: health.box.verdict, at: when(health.box.facts_at) })}
                </p>
                <ul>
                  {health.box.checks.map((c) => (
                    <li key={c.check} className={c.verdict === "pass" ? undefined : "attention"}>
                      <strong>{c.check}</strong>: {c.verdict}, {c.detail}
                      {c.fix ? (
                        <>
                          {" "}
                          ({tj("fix: {command}", { command: <code>{c.fix}</code> })})
                        </>
                      ) : null}
                    </li>
                  ))}
                </ul>
              </>
            )}
          </>
        )}
      </section>
      <section aria-label={t("Unhelpful answers")}>
        <h2>{t("Unhelpful answers")}</h2>
        {queueError ? (
          <p className="error">{t("The queue can't be shown: {error}", { error: queueError })}</p>
        ) : queue.length === 0 ? (
          <p>{t("No unhelpful answers to review.")}</p>
        ) : (
          <ul>
            {queue.map((q) => (
              <li key={q.trace}>
                <article aria-label={t("Feedback on {trace}", { trace: q.trace })}>
                  <p>
                    {tj("{trace} from {user}, {at}: {claims} claims, {uncited} uncited, {refused} refused", { trace: <strong>{q.trace}</strong> }, {
                      user: q.user,
                      at: when(q.at),
                      claims: q.claims,
                      uncited: q.uncited,
                      refused: q.refused,
                    })}
                    {q.wrong_citation ? `; ${t("wrong citation {cite}", { cite: q.wrong_citation })}` : ""}
                  </p>
                  {q.comment && <blockquote>{q.comment}</blockquote>}
                  <p>
                    {t("Decisions: {decisions}", { decisions: q.decisions.length === 0 ? t("none recorded") : q.decisions.map((d) => `${d.verdict} ${d.reason} ×${d.count}`).join(", ") })}
                  </p>
                  <button type="button" onClick={() => reviewed(q)}>
                    {t("Mark {trace} reviewed", { trace: q.trace })}
                  </button>
                </article>
              </li>
            ))}
          </ul>
        )}
      </section>
      <section aria-label={t("Kill switches")}>
        <h2>{t("Kill switches")}</h2>
        {switches.length === 0 ? (
          <p>{t("No kill switches are set.")}</p>
        ) : (
          <ul>
            {switches.map((s) => (
              <li key={s.scope}>
                <strong>{s.scope}</strong>: {t("{reason} (by {actor})", { reason: s.reason, actor: s.actor })}{" "}
                <button type="button" onClick={() => kill(false, ...(s.scope === "tenant" ? ["tenant", ""] : (s.scope.split(":") as [string, string])))}>
                  {t("Clear")}
                </button>{" "}
                {s.scope !== "tenant" && (
                  <button type="button" onClick={() => makePermanent(s)}>
                    {t("Make permanent")}
                  </button>
                )}
              </li>
            ))}
          </ul>
        )}
        <div className="kill-form">
          <label htmlFor={ids.scope}>{t("Scope")}</label>
          <select id={ids.scope} value={scope} onChange={(e) => setScope(e.target.value)}>
            <option value="agent">{"agent"}</option>
            <option value="connector">{"connector"}</option>
            <option value="tenant">{"tenant"}</option>
          </select>
          <label htmlFor={ids.name}>{t("Name")}</label>
          <input id={ids.name} type="text" value={name} onChange={(e) => setName(e.target.value)} disabled={scope === "tenant"} />
          <label htmlFor={ids.reason}>{t("Reason (required)")}</label>
          <input id={ids.reason} type="text" value={reason} onChange={(e) => setReason(e.target.value)} />
          <button type="button" onClick={() => kill(true)}>
            {t("Kill now")}
          </button>
        </div>
      </section>
      <section aria-label={t("Steward proposals")}>
        <h2>{t("Steward proposals")}</h2>
        {proposals.length === 0 ? (
          <p>{t("No open proposals.")}</p>
        ) : (
          <>
            <label htmlFor={ids.reject}>{t("Reason, to reject")}</label>
            <input id={ids.reject} type="text" value={rejectReason} onChange={(e) => setRejectReason(e.target.value)} />
            {proposals.map((p) => (
              <article key={p.id} aria-label={t("Proposal {id}", { id: p.id })}>
                <h3>{t("{space}: {id} by {requester}", { space: p.space, id: p.id, requester: p.requester })}</h3>
                <pre className="payload">{p.diff.join("\n")}</pre>
                <p>{tj("Hash {hash}", { hash: <code>{p.hash}</code> })}</p>
                <button type="button" onClick={() => approve(p)}>
                  {t("Approve hash {hash}", { hash: p.hash.slice(0, 12) })}
                </button>{" "}
                <button type="button" onClick={() => reject(p)}>
                  {t("Reject {id}", { id: p.id })}
                </button>
              </article>
            ))}
          </>
        )}
        {applyText && <pre className="payload" aria-label={t("Manifest to apply")}>{applyText}</pre>}
      </section>
      <AdminAgents fetcher={fetcher} />
      <AdminSpend fetcher={fetcher} />
      <AdminTokens fetcher={fetcher} />
      <section aria-label={t("Access reviews")}>
        <h2>{t("Access reviews")}</h2>
        {reviews.length === 0 ? (
          <p>{t("No access reviews.")}</p>
        ) : (
          <ul>
            {reviews.map((r) => (
              <li key={r.id}>
                <strong>
                  {r.space} {r.quarter}
                </strong>
                : {r.overdue ? t("{state} (overdue), due {due}", { state: r.state, due: r.due }) : t("{state}, due {due}", { state: r.state, due: r.due })}{" "}
                <a href={`/v1/admin/reviews/${encodeURIComponent(r.id)}/evidence`}>{t("Evidence for {space} {quarter}", { space: r.space, quarter: r.quarter })}</a>
              </li>
            ))}
          </ul>
        )}
      </section>
      {diff && (
        <section aria-label={t("Proposed manifest change")}>
          <h2>{t("Proposed manifest change")}</h2>
          <pre className="payload">{diff.join("\n")}</pre>
        </section>
      )}
      <section aria-label={t("Applied manifest")}>
        <h2>{t("Applied manifest")}</h2>
        <pre className="payload">{manifest}</pre>
      </section>
      {status && <p role="status">{status}</p>}
    </main>
  );
}
