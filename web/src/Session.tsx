import { useState } from "react";
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
