import { useEffect, useState } from "react";
import { AskError, contextOf, exclude, forgetMemory, myMemory } from "./api";
import type { ContextPanel, Fetch, MemoryLine } from "./api";
import { errorText, t, tn } from "./i18n";

// See and steer the context (SB-5501 to SB-5504): what an answer was built from, from its
// trace and checked as me now; Keep and Drop for the rest of the conversation; Don't use
// this for me; and the memory lines it was given, each with Forget this.

export type SteerKind = "keep" | "drop";

const errMsg = (e: unknown) => (e instanceof AskError ? errorText(e.error).message : t("This is unavailable right now."));

export function ContextView({
  trace,
  fetcher,
  onSteer,
}: {
  trace: string;
  fetcher?: Fetch;
  onSteer?: (doc: string, kind: SteerKind) => void;
}) {
  const [open, setOpen] = useState(false);
  const [p, setP] = useState<ContextPanel>();
  const [err, setErr] = useState("");
  const [note, setNote] = useState("");
  const show = async () => {
    if (open) {
      setOpen(false);
      return;
    }
    setOpen(true);
    setErr("");
    try {
      setP(await contextOf(trace, fetcher));
    } catch (e) {
      setErr(errMsg(e));
    }
  };
  const notForMe = async (doc: string) => {
    try {
      await exclude(doc, true, fetcher);
      setNote(t("Won't be used for you. Undo it on My data."));
    } catch (e) {
      setNote(errMsg(e));
    }
  };
  return (
    <section className="context-panel">
      <button type="button" className="link" aria-expanded={open} onClick={() => void show()}>
        {t("Context")}
      </button>
      {open && (
        <div role="region" aria-label={t("What this answer was built from")}>
          {err && <p className="notice">{err}</p>}
          {p && (
            <>
              <ul>
                {p.sources.map((s) => (
                  <li key={s.cite}>
                    {s.title || s.cite}{" "}
                    {onSteer && (
                      <>
                        <button type="button" className="link" onClick={() => onSteer(s.cite, "keep")}>
                          {t("Keep")}
                        </button>{" "}
                        <button type="button" className="link" onClick={() => onSteer(s.cite, "drop")}>
                          {t("Drop")}
                        </button>{" "}
                      </>
                    )}
                    <button type="button" className="link" onClick={() => void notForMe(s.cite)}>
                      {t("Don't use this for me")}
                    </button>
                  </li>
                ))}
              </ul>
              {p.sources_hidden + p.documents_hidden > 0 && (
                <p className="meta">{tn(p.sources_hidden + p.documents_hidden, "{n} source you can't open now", "{n} sources you can't open now")}</p>
              )}
              {p.kept.length > 0 && <p className="meta">{t("Kept: {docs}", { docs: p.kept.map((d) => d.title || d.cite).join(", ") })}</p>}
              {p.dropped.length > 0 && <p className="meta">{t("Dropped: {docs}", { docs: p.dropped.map((d) => d.title || d.cite).join(", ") })}</p>}
              {p.memory.length > 0 && <p className="meta">{t("Your memory: {lines}", { lines: p.memory.map((m) => m.text).join("; ") })}</p>}
              <p className="meta">
                {t("Earlier questions read: {kept}, left out: {dropped}", { kept: p.earlier_kept, dropped: p.earlier_dropped })}
                {p.tier ? ` · ${t("Model: {tier}", { tier: p.tier })}` : ""}
                {p.skill ? ` · /${p.skill}` : ""}
              </p>
              {note && <p className="notice" role="status">{note}</p>}
            </>
          )}
        </div>
      )}
    </section>
  );
}

// MemoryUsed shows the memory lines an answer was given, under it, with Forget this (SB-5504).
export function MemoryUsed({ ids, fetcher }: { ids: string[]; fetcher?: Fetch }) {
  const [lines, setLines] = useState<MemoryLine[]>([]);
  useEffect(() => {
    let live = true;
    myMemory(fetcher).then(
      (m) => live && setLines(m.lines.filter((l) => ids.includes(l.id))),
      () => undefined,
    );
    return () => {
      live = false;
    };
  }, [ids, fetcher]);
  if (lines.length === 0) return null;
  return (
    <section className="memory-used" aria-label={t("Memory used")}>
      {lines.map((l) => (
        <p key={l.id} className="meta">
          {t("Used your memory: {line}", { line: l.text })}{" "}
          <button
            type="button"
            className="link"
            onClick={() => void forgetMemory(l.id, fetcher).then(() => setLines((xs) => xs.filter((x) => x.id !== l.id)))}
          >
            {t("Forget this")}
          </button>
        </p>
      ))}
    </section>
  );
}
