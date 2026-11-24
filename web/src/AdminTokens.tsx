import { useEffect, useId, useState } from "react";
import type { Fetch } from "./api";
import { t } from "./i18n";

// One live API token as the admin sees it (sb-admin(3), GET /v1/admin/tokens; SB-3803).
export interface AdminToken {
  id: string;
  owner: string;
  name: string;
  created: string;
  last_used?: string;
  expires: string;
}

// Every token in one place: oldest unused first, each with Revoke and a reason.
export function AdminTokens({ fetcher = fetch }: { fetcher?: Fetch }) {
  const [items, setItems] = useState<AdminToken[]>([]);
  const [error, setError] = useState<string>();
  const [status, setStatus] = useState<string>();
  const [reason, setReason] = useState("");
  const reasonId = useId();

  async function load() {
    try {
      const res = await fetcher("/v1/admin/tokens", { credentials: "same-origin" });
      const data = (await res.json().catch(() => null)) as { items?: AdminToken[]; error?: { message?: string } } | null;
      if (!res.ok) throw new Error(data?.error?.message ?? t("The tokens can't be read right now."));
      setItems(data?.items ?? []);
      setError(undefined);
    } catch (e) {
      setError((e as Error).message);
    }
  }
  useEffect(() => {
    void load();
  }, []);

  async function revoke(tok: AdminToken) {
    if (!reason.trim()) {
      setStatus(t("A reason is required; it goes into the audit log."));
      return;
    }
    try {
      const res = await fetcher(`/v1/admin/tokens/${encodeURIComponent(tok.id)}`, {
        method: "DELETE",
        credentials: "same-origin",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ reason: reason.trim() }),
      });
      const data = (await res.json().catch(() => null)) as { error?: { message?: string } } | null;
      if (!res.ok) throw new Error(data?.error?.message ?? t("That didn't work."));
      setStatus(t("Revoked {name}.", { name: tok.name }));
      await load();
    } catch (e) {
      setStatus((e as Error).message);
    }
  }

  return (
    <section aria-label={t("API tokens")}>
      <h2>{t("API tokens")}</h2>
      {error ? (
        <p role="alert">{error}</p>
      ) : items.length === 0 ? (
        <p>{t("No live API tokens.")}</p>
      ) : (
        <>
          <label htmlFor={reasonId}>{t("Reason (required)")}</label>
          <input id={reasonId} type="text" value={reason} onChange={(e) => setReason(e.target.value)} />
          <ul>
            {items.map((tok) => (
              <li key={tok.id}>
                <strong>{tok.name}</strong>{" "}
                {t("by {owner}, made {created}, {used}, expires {expires}", {
                  owner: tok.owner,
                  created: tok.created.slice(0, 10),
                  used: tok.last_used ? t("last used {date}", { date: tok.last_used }) : t("never used"),
                  expires: tok.expires.slice(0, 10),
                })}{" "}
                <button type="button" onClick={() => revoke(tok)}>
                  {t("Revoke {name}", { name: tok.name })}
                </button>
              </li>
            ))}
          </ul>
        </>
      )}
      {status && <p role="status">{status}</p>}
    </section>
  );
}
