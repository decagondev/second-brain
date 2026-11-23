import { useEffect, useId, useState } from "react";
import type { Fetch } from "./api";
import { AskError } from "./api";
import type { ActionView } from "./actions";
import { approveAction, getAction, rejectAction } from "./actions";
import { errorText, t } from "./i18n";

const show = (v: unknown) => (v === null || v === undefined ? t("(none)") : typeof v === "string" ? v : JSON.stringify(v));

// The approver view (sb-actions(3), The approver view; SB-407). It echoes the hash it was
// shown, so a payload that changed after loading is refused, never approved unseen.
export function ApproverView({ id, fetcher }: { id: string; fetcher?: Fetch }) {
  const [action, setAction] = useState<ActionView>();
  const [error, setError] = useState<string>();
  const [note, setNote] = useState("");
  const [busy, setBusy] = useState(false);
  const noteId = useId();

  useEffect(() => {
    getAction(id, fetcher)
      .then(setAction)
      .catch((e) => setError(e instanceof AskError ? errorText(e.error).message : t("This action couldn't be loaded.")));
  }, [id, fetcher]);

  async function decide(kind: "approve" | "reject") {
    if (!action || busy) return;
    setBusy(true);
    try {
      const next = kind === "approve" ? await approveAction(action.id, action.payload_sha256, note, fetcher) : await rejectAction(action.id, note, fetcher);
      setAction({ ...next, diff: action.diff });
      setError(undefined);
    } catch (e) {
      if (e instanceof AskError) {
        const x = errorText(e.error);
        setError(`${x.message} ${x.remedy}`.trim());
      } else setError(t("That didn't work."));
    } finally {
      setBusy(false);
    }
  }

  if (!action) {
    return (
      <main>
        <h1>{t("Approval")}</h1>
        {error ? <p role="alert">{error}</p> : <p>{t("Loading…")}</p>}
      </main>
    );
  }
  const [system, object] = action.target.split(/:(.*)/s);
  const pending = action.state === "PendingApproval";
  return (
    <main>
      <h1>{t("Approve {workflow}", { workflow: action.workflow })}</h1>
      <dl className="action-facts">
        <dt>{t("Target system")}</dt>
        <dd>{system}</dd>
        <dt>{t("Target")}</dt>
        <dd>{object}</dd>
        <dt>{t("Requested by")}</dt>
        <dd>{action.requester}</dd>
        <dt>{t("Policy reason")}</dt>
        <dd>{action.policy_reason ?? t("(none)")}</dd>
        <dt>{t("Expires")}</dt>
        <dd>{action.expires_at ?? t("(none)")}</dd>
        <dt>{t("State")}</dt>
        <dd>{action.state}</dd>
        <dt>{t("Payload hash")}</dt>
        <dd>
          <code>{action.payload_sha256}</code>
        </dd>
      </dl>
      <h2>{t("Exact payload")}</h2>
      <pre className="payload">{JSON.stringify(action.payload, null, 2)}</pre>
      <h2>{t("Changes")}</h2>
      {action.diff === null ? (
        <p>{t("The target's current state can't be read, so there is no diff. Check the payload above.")}</p>
      ) : action.diff.length === 0 ? (
        <p>{t("No field changes.")}</p>
      ) : (
        <table className="diff">
          <caption className="visually-hidden">{t("Fields that change")}</caption>
          <thead>
            <tr>
              <th scope="col">{t("Field")}</th>
              <th scope="col">{t("Now")}</th>
              <th scope="col">{t("After approval")}</th>
            </tr>
          </thead>
          <tbody>
            {action.diff.map((c) => (
              <tr key={c.field}>
                <th scope="row">{c.field}</th>
                <td>
                  <del>{show(c.before)}</del>
                </td>
                <td>
                  <ins>{show(c.after)}</ins>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
      {pending && (
        <div className="decide" role="group" aria-label={t("Decision")}>
          <label htmlFor={noteId}>{t("Note (optional)")}</label>
          <textarea id={noteId} value={note} maxLength={2000} rows={2} onChange={(e) => setNote(e.target.value)} />
          <button type="button" disabled={busy} onClick={() => decide("approve")}>
            {t("Approve this payload")}
          </button>
          <button type="button" disabled={busy} onClick={() => decide("reject")}>
            {t("Reject")}
          </button>
        </div>
      )}
      {action.receipt && <p role="status">{t("Ran. Receipt {receipt}.", { receipt: action.receipt })}</p>}
      {action.state === "DidNotRun" && <p role="status">{t("Did not run: {reason}", { reason: action.reason ?? "" })}</p>}
      {action.state === "Rejected" && <p role="status">{t("Rejected.")}</p>}
      {error && <p role="alert">{error}</p>}
    </main>
  );
}
