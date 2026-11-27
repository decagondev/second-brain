import { useEffect, useId, useState } from "react";
import { exclude, exclusions } from "./api";
import type { Exclusions } from "./api";
import type { Fetch } from "./api";
import { tj, tn, t } from "./i18n";

// Sign out everywhere (sb-identity(7), Signing out everywhere; SB-1906): every session
// of the person ends, this one included.
export function SignOutEverywhere({ fetcher = fetch, onSignedOut }: { fetcher?: Fetch; onSignedOut?: () => void }) {
  const [state, setState] = useState<"idle" | "busy" | "done" | "failed">("idle");
  const [count, setCount] = useState(0);

  async function revoke() {
    setState("busy");
    try {
      const res = await fetcher("/v1/sessions/revoke", { method: "POST", credentials: "same-origin" });
      if (!res.ok) throw new Error(String(res.status));
      const data = (await res.json()) as { revoked?: number };
      setCount(typeof data.revoked === "number" ? data.revoked : 0);
      setState("done");
      onSignedOut?.();
    } catch {
      setState("failed");
    }
  }

  return (
    <footer className="session">
      {state === "done" ? (
        <p role="status">
          {tj(tn(count, "Signed out on {n} device, this one included. {link}", "Signed out on {n} devices, this one included. {link}"), {
            link: <a href="/login">{t("Sign in again")}</a>,
          })}
        </p>
      ) : (
        <>
          <button type="button" disabled={state === "busy"} onClick={revoke}>
            {t("Sign out everywhere")}
          </button>
          {state === "failed" && (
            <p className="error" role="alert">
              {t("Your other sessions weren't ended. Try again.")}
            </p>
          )}
        </>
      )}
    </footer>
  );
}

interface MySession {
  id: string;
  kind: "web" | "token";
  name?: string;
  started: string;
  last_used: string;
  expires: string;
  current?: boolean;
}

// Where I'm signed in (sb-identity(7); SB-3503): the person's own sessions and API tokens,
// with Sign out on each. Never an address or a token.
export function MySessions({ fetcher = fetch, onSignedOut }: { fetcher?: Fetch; onSignedOut?: () => void }) {
  const [items, setItems] = useState<MySession[]>();
  const [status, setStatus] = useState("");
  async function load() {
    try {
      const res = await fetcher("/v1/me/sessions", { credentials: "same-origin" });
      if (!res.ok) throw new Error(String(res.status));
      const d = (await res.json()) as { items?: MySession[] };
      setItems(Array.isArray(d.items) ? d.items : []);
    } catch {
      setStatus(t("Your sessions can't be read right now."));
    }
  }
  async function signOut(s: MySession) {
    try {
      const res = await fetcher(`/v1/me/sessions/${encodeURIComponent(s.id)}`, { method: "DELETE", credentials: "same-origin" });
      if (!res.ok) throw new Error(String(res.status));
      if (s.current) onSignedOut?.();
      setItems((xs) => (xs ?? []).filter((x) => x.id !== s.id));
      setStatus(t("Signed out."));
    } catch {
      setStatus(t("That session wasn't ended. Try again."));
    }
  }
  return (
    <details
      className="session"
      onToggle={(e) => {
        if ((e.target as HTMLDetailsElement).open && !items) void load();
      }}
    >
      <summary>{t("Where I'm signed in")}</summary>
      {items && items.length === 0 && <p>{t("No sessions.")}</p>}
      {items && items.length > 0 && (
        <ul>
          {items.map((s) => {
            const what = s.current ? t("This browser") : s.kind === "token" ? t("API token {name}", { name: s.name ?? "" }) : t("Web");
            return (
              <li key={s.id}>
                {what}{" "}
                <span className="meta">
                  · {t("since {date}", { date: new Date(s.started).toLocaleDateString() })} · {t("last used {date}", { date: s.last_used })} ·{" "}
                  {t("ends {date}", { date: new Date(s.expires).toLocaleDateString() })}
                </span>{" "}
                <button type="button" aria-label={t("Sign out: {what}", { what })} onClick={() => signOut(s)}>
                  {t("Sign out")}
                </button>
              </li>
            );
          })}
        </ul>
      )}
      {status && <p role="status">{status}</p>}
    </details>
  );
}

// What the brain keeps about me and Forget my questions (sb-ask(3), Your own data;
// SB-3501, SB-3502).
export function MyData({ fetcher = fetch }: { fetcher?: Fetch }) {
  const id = useId();
  const [phrase, setPhrase] = useState("");
  const [status, setStatus] = useState("");
  async function download() {
    try {
      const res = await fetcher("/v1/me/export", { credentials: "same-origin" });
      if (!res.ok) throw new Error(String(res.status));
      const blob = new Blob([await res.text()], { type: "application/json" });
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = "my-second-brain-data.json";
      a.click();
      URL.revokeObjectURL(url);
      setStatus(t("Downloaded."));
    } catch {
      setStatus(t("Your data couldn't be downloaded. Try again."));
    }
  }
  async function forget() {
    try {
      const res = await fetcher("/v1/me/forget", {
        method: "POST",
        credentials: "same-origin",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ confirm: phrase.trim() }),
      });
      if (!res.ok) throw new Error(String(res.status));
      const d = (await res.json()) as { held?: number };
      setPhrase("");
      setStatus(
        d.held ? tn(d.held, "Forgotten, except {n} record a legal hold keeps.", "Forgotten, except {n} records a legal hold keeps.") : t("Your questions are forgotten."),
      );
    } catch {
      setStatus(t("Not everything was forgotten. Try again."));
    }
  }
  return (
    <section aria-label={t("Your data")}>
      <h2>{t("Your data")}</h2>
      <button type="button" onClick={download}>
        {t("Download my data")}
      </button>
      <label htmlFor={id}>{t('Type "forget my questions" to erase your questions, answers and saved questions')}</label>
      <input id={id} type="text" value={phrase} onChange={(e) => setPhrase(e.target.value)} />
      <button type="button" disabled={phrase.trim() !== "forget my questions"} onClick={forget}>
        {t("Forget my questions")}
      </button>
      {status && <p role="status">{status}</p>}
      <NotForMe fetcher={fetcher} />
    </section>
  );
}

// NotForMe is the person's Don't use this for me list (SB-5503), with Undo on each.
export function NotForMe({ fetcher = fetch }: { fetcher?: Fetch }) {
  const [list, setList] = useState<Exclusions>();
  useEffect(() => {
    exclusions(fetcher).then(setList, () => undefined);
  }, [fetcher]);
  if (!list || !Array.isArray(list.items) || (list.items.length === 0 && !list.hidden)) return null;
  return (
    <section aria-label={t("Not used for me")}>
      <h3>{t("Not used for me")}</h3>
      <ul>
        {list.items.map((x) => (
          <li key={x.document}>
            {x.title || x.document}{" "}
            <button type="button" className="link" onClick={() => void exclude(x.document, false, fetcher).then(setList)}>
              {t("Undo")}
            </button>
          </li>
        ))}
      </ul>
      {list.hidden > 0 && <p className="meta">{tn(list.hidden, "{n} source you can't open now", "{n} sources you can't open now")}</p>}
    </section>
  );
}
