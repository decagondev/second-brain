import { useEffect, useId, useState } from "react";
import type { FormEvent } from "react";
import { AskError, listTokens, makeToken, revokeToken } from "./api";
import type { ApiToken, Fetch } from "./api";
import { ErrorBox } from "./ErrorBox";
import { t } from "./i18n";

// Read-only API tokens (sb-api(3); SB-2306): a person's own tokens for scripts, which ask
// and search as them and nothing else. A new token is shown once.
export function ApiTokens({ fetcher = fetch }: { fetcher?: Fetch }) {
  const [tokens, setTokens] = useState<ApiToken[]>([]);
  const [name, setName] = useState("");
  const [days, setDays] = useState(30);
  const [made, setMade] = useState<ApiToken>();
  const [error, setError] = useState<AskError>();
  const nameId = useId();
  const daysId = useId();

  useEffect(() => {
    let live = true;
    listTokens(fetcher).then(
      (x) => live && setTokens(x),
      () => undefined,
    );
    return () => {
      live = false;
    };
  }, [fetcher]);

  async function submit(e: FormEvent) {
    e.preventDefault();
    if (!name.trim()) return;
    setError(undefined);
    try {
      const x = await makeToken(name.trim(), days, fetcher);
      setMade(x);
      setName("");
      setTokens((ts) => [...ts, { ...x, token: undefined }]);
    } catch (err) {
      if (err instanceof AskError) setError(err);
    }
  }

  async function revoke(id: string) {
    setError(undefined);
    try {
      await revokeToken(id, fetcher);
      setTokens((ts) => ts.filter((x) => x.id !== id));
      if (made?.id === id) setMade(undefined);
    } catch (err) {
      if (err instanceof AskError) setError(err);
    }
  }

  return (
    <section className="tokens" aria-label={t("API tokens")}>
      <h2>{t("API tokens")}</h2>
      <p>{t("A token lets your own scripts ask and search as you, and nothing else.")}</p>
      {made?.token && (
        <div className="notice" role="status">
          <p>{t("Copy your new token now: it won't be shown again.")}</p>
          <code>{made.token}</code>
        </div>
      )}
      {tokens.length > 0 && (
        <ul>
          {tokens.map((x) => (
            <li key={x.id}>
              <span>{x.name}</span> <span className="meta">{t("expires {date}", { date: x.expires.slice(0, 10) })}</span>{" "}
              <button type="button" onClick={() => revoke(x.id)} aria-label={t("Revoke {name}", { name: x.name })}>
                {t("Revoke")}
              </button>
            </li>
          ))}
        </ul>
      )}
      <form onSubmit={submit}>
        <label htmlFor={nameId}>{t("Token name")}</label>
        <input id={nameId} value={name} maxLength={60} onChange={(e) => setName(e.target.value)} />
        <label htmlFor={daysId}>{t("Valid for")}</label>
        <select id={daysId} value={days} onChange={(e) => setDays(Number(e.target.value))}>
          {[7, 30, 90].map((d) => (
            <option key={d} value={d}>
              {t("{n} days", { n: d })}
            </option>
          ))}
        </select>
        <button type="submit" disabled={!name.trim()}>
          {t("Make a token")}
        </button>
      </form>
      {error && <ErrorBox error={error.error} />}
    </section>
  );
}
