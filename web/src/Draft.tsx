import { useId, useState } from "react";
import type { ApiError, AskResponse, Fetch } from "./api";
import { AskError } from "./api";
import { AnswerView } from "./Answer";

interface DraftResponse extends AskResponse {
  context: string;
}

async function post<T>(f: Fetch, url: string, body: unknown): Promise<T> {
  let res: Response;
  try {
    res = await f(url, { method: "POST", credentials: "same-origin", headers: { "Content-Type": "application/json" }, body: JSON.stringify(body) });
  } catch {
    throw new AskError({ code: "EUNAVAIL", message: "Drafts are unavailable right now.", trace: "", remedy: "Try again in a minute." });
  }
  const data: unknown = await res.json().catch(() => null);
  if (!res.ok) {
    throw new AskError((data as { error?: ApiError } | null)?.error ?? { code: "EUNAVAIL", message: "Drafts are unavailable right now.", trace: "", remedy: "Try again in a minute." });
  }
  return data as T;
}

// The draft-reply panel (sb-actions(3), Drafts; SB-409). It never sends: the person copies
// the draft and sends it in the target app.
export function DraftPanel({ fetcher = fetch, copy }: { fetcher?: Fetch; copy?: (text: string) => Promise<void> }) {
  const [context, setContext] = useState("");
  const [instruction, setInstruction] = useState("");
  const [draft, setDraft] = useState<DraftResponse>();
  const [status, setStatus] = useState<string>();
  const [busy, setBusy] = useState(false);
  const ids = { ctx: useId(), instr: useId() };
  const write = copy ?? ((t: string) => navigator.clipboard.writeText(t));

  async function make() {
    if (busy) return;
    setBusy(true);
    setStatus(undefined);
    try {
      setDraft(await post<DraftResponse>(fetcher, "/v1/drafts", { context: context.trim(), instruction: instruction.trim() }));
    } catch (e) {
      setDraft(undefined);
      setStatus(e instanceof AskError ? `${e.error.message} ${e.error.remedy}` : "That didn't work.");
    } finally {
      setBusy(false);
    }
  }

  async function finish(outcome: "used" | "discarded") {
    if (!draft) return;
    if (outcome === "used") {
      await write(draft.answer.claims.map((c) => c.text).join(" "));
    }
    await post(fetcher, `/v1/drafts/${encodeURIComponent(draft.trace)}/outcome`, { outcome }).catch(() => undefined);
    setStatus(outcome === "used" ? "Copied. Paste it into the ticket or thread and send it there." : "Draft discarded.");
    if (outcome === "discarded") setDraft(undefined);
  }

  return (
    <section aria-label="Draft a reply" className="draft">
      <h2>Draft a reply</h2>
      <label htmlFor={ids.ctx}>Ticket or thread (for example tickets:OPS-1182)</label>
      <input id={ids.ctx} type="text" value={context} onChange={(e) => setContext(e.target.value)} />
      <label htmlFor={ids.instr}>What should the reply say?</label>
      <textarea id={ids.instr} value={instruction} maxLength={2000} rows={2} onChange={(e) => setInstruction(e.target.value)} />
      <button type="button" disabled={busy || !context.trim() || !instruction.trim()} onClick={make}>
        {busy ? "Drafting…" : "Draft"}
      </button>
      {draft && (
        <>
          <p>Draft for {draft.context}. Nothing is sent until you send it yourself.</p>
          <AnswerView answer={draft} />
          <button type="button" onClick={() => finish("used")}>
            Copy draft
          </button>
          <button type="button" onClick={() => finish("discarded")}>
            Discard
          </button>
        </>
      )}
      {status && <p role="status">{status}</p>}
    </section>
  );
}
