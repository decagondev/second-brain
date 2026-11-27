import { useEffect, useState } from "react";
import { AskError, contextOf, exclude, forgetMemory, myMemory, retrievalOf, whyNot } from "./api";
import type { FormEvent } from "react";
import type { ContextPanel, Fetch, MemoryLine, RetrievalRow } from "./api";
import { errorText, msg, t, tn } from "./i18n";

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

// WhyNotView is Why not...? under an answer (SB-5506): a title, link or KIND:ID, and why the
// answer didn't use it, checked as me now.
export function WhyNotView({ trace, fetcher }: { trace: string; fetcher?: Fetch }) {
  const [doc, setDoc] = useState("");
  const [answer, setAnswer] = useState("");
  const ask = async (e: FormEvent) => {
    e.preventDefault();
    try {
      setAnswer((await whyNot(trace, doc.trim(), fetcher)).text);
    } catch (x) {
      setAnswer(errMsg(x));
    }
  };
  return (
    <form className="why-not inline" onSubmit={(e) => void ask(e)}>
      <input aria-label={t("Why not this document? Its title, link or id")} value={doc} onChange={(e) => setDoc(e.target.value)} />
      <button type="submit" disabled={!doc.trim()}>
        {t("Why not this document?")}
      </button>
      {answer && <span role="status"> {answer}</span>}
    </form>
  );
}

// whyLabels are the retrieval view's reasons in words.
const whyLabels: Record<string, string> = {
  kept: msg("kept"),
  budget: msg("below the cut"),
  per_document: msg("too many from one document"),
  retired: msg("retired"),
  held: msg("held"),
  suspect: msg("suspect"),
};

// RetrievalView is the replay of a reported or unhelpful answer, for its space's stewards
// (SB-5507): ranks, the fused score and why each passage was kept or cut.
export function RetrievalView({ space, trace, fetcher }: { space: string; trace: string; fetcher?: Fetch }) {
  const [rows, setRows] = useState<RetrievalRow[]>();
  const [hidden, setHidden] = useState(0);
  const [err, setErr] = useState("");
  const show = async () => {
    try {
      const v = await retrievalOf(space, trace, fetcher);
      setRows(v.rows);
      setHidden(v.hidden);
    } catch (e) {
      setErr(errMsg(e));
    }
  };
  return (
    <div className="retrieval">
      {!rows && (
        <button type="button" className="link" onClick={() => void show()}>
          {t("Show the retrieval")}
        </button>
      )}
      {err && <p className="notice">{err}</p>}
      {rows && (
        <table>
          <thead>
            <tr>
              <th>{t("Rank")}</th>
              <th>{t("Passage")}</th>
              <th>{t("Score")}</th>
              <th>{t("Kept or cut")}</th>
            </tr>
          </thead>
          <tbody>
            {rows.map((r) => (
              <tr key={r.rank}>
                <td>{r.rank}</td>
                <td>{r.title || r.cite}</td>
                <td>{r.score.toFixed(4)}</td>
                <td>{t(whyLabels[r.why] ?? r.why)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
      {rows && hidden > 0 && <p className="meta">{tn(hidden, "{n} source you can't open now", "{n} sources you can't open now")}</p>}
    </div>
  );
}
