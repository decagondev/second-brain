import { useId, useState } from "react";
import type { FormEvent } from "react";
import { AskError, handover } from "./api";
import type { ApiError, Claim, Fetch, HandoverBrief } from "./api";
import { ClaimView } from "./Answer";

// The handover brief (sb-agents(7), Handover; SB-2002): a person's documents that the asker
// can open now, cited and grouped by where they live, and a count of the ones they can't.
export function HandoverView({ fetcher }: { fetcher?: Fetch }) {
  const [person, setPerson] = useState("");
  const [busy, setBusy] = useState(false);
  const [brief, setBrief] = useState<HandoverBrief>();
  const [error, setError] = useState<ApiError>();
  const inputId = useId();
  const hintId = useId();

  async function submit(e: FormEvent) {
    e.preventDefault();
    const p = person.trim();
    if (!p || busy) return;
    setBusy(true);
    setBrief(undefined);
    setError(undefined);
    try {
      setBrief(await handover(p, fetcher));
    } catch (err) {
      setError(err instanceof AskError ? err.error : { code: "EUNAVAIL", message: "Something went wrong.", trace: "", remedy: "Try again." });
    } finally {
      setBusy(false);
    }
  }

  return (
    <section aria-label="Handover" className="handover">
      <h2>Handover</h2>
      <p>For someone leaving or changing role: a brief of the documents they own that you can open.</p>
      <form onSubmit={submit}>
        <label htmlFor={inputId}>Person</label>
        <input id={inputId} aria-describedby={hintId} value={person} maxLength={80} onChange={(e) => setPerson(e.target.value)} />
        <p id={hintId} className="meta">
          Their user id, usr_…. You can brief yourself, someone you manage, or a member of a space you steward.
        </p>
        <button type="submit" disabled={busy || person.trim() === ""}>
          {busy ? "Briefing…" : "Brief"}
        </button>
      </form>
      {error && (
        <div className="error" role="alert">
          <p>{error.message}</p>
          {error.remedy && error.remedy.toLowerCase() !== error.message.toLowerCase() && <p>{error.remedy}</p>}
          {error.trace && <p>Quote trace {error.trace} to support.</p>}
        </div>
      )}
      {brief && <BriefView brief={brief} />}
    </section>
  );
}

function BriefView({ brief }: { brief: HandoverBrief }) {
  const groups = new Map<string, Claim[]>();
  for (const c of brief.answer.claims) {
    const where = (c.cites[0] && brief.locations?.[c.cites[0].cite]) || "";
    groups.set(where, [...(groups.get(where) ?? []), c]);
  }
  return (
    <article className="answer" aria-label={`Handover brief of ${brief.person}`} aria-live="polite">
      {[...groups].map(([where, claims]) => (
        <section key={where} aria-label={where || "Other documents"}>
          <h3>{where || "Other documents"}</h3>
          <ul className="claims">
            {claims.map((c, i) => (
              <ClaimView key={i} claim={c} trace={brief.trace} />
            ))}
          </ul>
        </section>
      ))}
      {brief.dropped > 0 && (
        <p className="meta">{brief.dropped === 1 ? "1 line was" : `${brief.dropped} lines were`} left out because no document supported it.</p>
      )}
      {brief.hidden > 0 && (
        <p className="refused" role="note">
          {brief.hidden === 1 ? "1 of their documents is" : `${brief.hidden} of their documents are`} not in this brief because you can't open{" "}
          {brief.hidden === 1 ? "it" : "them"}. Ask the owner of each source to move access to whoever takes the work over before {brief.person} leaves:
          once their account is removed, their access goes with it.
        </p>
      )}
      <p className="trace">Trace {brief.trace}</p>
    </article>
  );
}
