import { useId, useState } from "react";
import type { FormEvent } from "react";
import type { Fetch } from "./api";
import { t } from "./i18n";

// Draft the next action (SB-5304): the answer becomes a draft to edit, then Send. A ticket
// comment goes through the gated ticket-update workflow, approved like any action; a message
// is posted by the bot in one of the space's channels, marked as sent by the person.
// Nothing goes out before Send.

async function post(fetcher: Fetch, path: string, body: unknown): Promise<Record<string, unknown>> {
  const r = await fetcher(path, { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(body) });
  const out = (await r.json().catch(() => ({}))) as Record<string, unknown> & { error?: { message?: string } };
  if (!r.ok) throw new Error(out.error?.message ?? t("That didn't work. Try again."));
  return out;
}

export function NextAction({ trace, space, fetcher = fetch }: { trace: string; space?: string; fetcher?: Fetch }) {
  const [kind, setKind] = useState<"" | "ticket" | "message">("");
  const [text, setText] = useState("");
  const [target, setTarget] = useState("");
  const [cites, setCites] = useState<string[]>([]); // a confidential draft can't go to a channel (LAW-32)
  const [status, setStatus] = useState("");
  const [busy, setBusy] = useState(false);
  const ids = { text: useId(), target: useId() };
  async function draft(k: "ticket" | "message") {
    setBusy(true);
    setStatus("");
    try {
      const d = await post(fetcher, "/v1/actions/draft", { trace, kind: k });
      setKind(k);
      setText(String(d.text ?? ""));
      setCites(Array.isArray(d.cites) ? (d.cites as string[]) : []);
    } catch (e) {
      setStatus(e instanceof Error ? e.message : t("That didn't work. Try again."));
    } finally {
      setBusy(false);
    }
  }
  async function send(e: FormEvent) {
    e.preventDefault();
    setBusy(true);
    try {
      if (kind === "ticket") {
        const id = target.trim().replace(/^tickets:/, "");
        const r = await post(fetcher, "/v1/actions", { workflow: "ticket-update", target: `tickets:${id}`, payload: { comment: text } });
        setStatus(t("Sent for approval: {state}.", { state: String(r.state ?? "") }));
      } else {
        await post(fetcher, `/v1/spaces/${encodeURIComponent(space ?? "")}/messages`, { channel: target.trim(), text, cites });
        setStatus(t("Posted in {channel}, marked as sent by you.", { channel: target.trim() }));
      }
      setKind("");
    } catch (err) {
      setStatus(err instanceof Error ? err.message : t("That didn't work. Try again."));
    } finally {
      setBusy(false);
    }
  }
  return (
    <div className="next-action">
      {kind === "" && (
        <>
          <button type="button" disabled={busy} onClick={() => void draft("ticket")}>
            {t("Draft a ticket comment")}
          </button>{" "}
          {space && (
            <button type="button" disabled={busy} onClick={() => void draft("message")}>
              {t("Draft a message")}
            </button>
          )}
        </>
      )}
      {kind !== "" && (
        <form onSubmit={(e) => void send(e)} aria-label={kind === "ticket" ? t("Draft a ticket comment") : t("Draft a message")}>
          <label htmlFor={ids.target}>{kind === "ticket" ? t("Ticket") : t("Channel")}</label>{" "}
          <input id={ids.target} value={target} onChange={(e) => setTarget(e.target.value)} required placeholder={kind === "ticket" ? "OPS-1182" : "#ops"} />
          <label htmlFor={ids.text}>{t("Text")}</label>
          <textarea id={ids.text} rows={6} maxLength={4000} value={text} onChange={(e) => setText(e.target.value)} required />
          <button type="submit" disabled={busy}>
            {t("Send")}
          </button>{" "}
          <button type="button" onClick={() => setKind("")}>
            {t("Cancel")}
          </button>
        </form>
      )}
      {status && (
        <p className="meta" role="status">
          {status}
        </p>
      )}
    </div>
  );
}
