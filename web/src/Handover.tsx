import { useId, useState } from "react";
import type { FormEvent } from "react";
import { AskError, briefable, handover } from "./api";
import type { ApiError, Briefable, Claim, Fetch, HandoverBrief } from "./api";
import { ClaimView } from "./Answer";
import { ErrorBox } from "./ErrorBox";
import { t, tn } from "./i18n";

// The handover brief (sb-agents(7), Handover; SB-2002): a person's documents that the asker
// can open now, cited and grouped by where they live, and a count of the ones they can't.
export function HandoverView({ fetcher }: { fetcher?: Fetch }) {
  const [person, setPerson] = useState("");
  const [busy, setBusy] = useState(false);
  const [brief, setBrief] = useState<HandoverBrief>();
  const [error, setError] = useState<ApiError>();
  const [choices, setChoices] = useState<Briefable[]>();
  const inputId = useId();
  const hintId = useId();

  const fail = (err: unknown) =>
    setError(err instanceof AskError ? err.error : { code: "EUNAVAIL", message: t("Something went wrong."), trace: "", remedy: t("Try again.") });

  async function brief1(id: string) {
    setBusy(true);
    setBrief(undefined);
    setError(undefined);
    setChoices(undefined);
    try {
      setBrief(await handover(id, fetcher));
    } catch (err) {
      fail(err);
    } finally {
      setBusy(false);
    }
  }

  // A user id briefs at once; a name or an email finds the people you may brief (SB-2106).
  async function submit(e: FormEvent) {
    e.preventDefault();
    const p = person.trim();
    if (!p || busy) return;
    if (/^usr_/.test(p)) return brief1(p);
    setBusy(true);
    setBrief(undefined);
    setError(undefined);
    setChoices(undefined);
    try {
      const found = await briefable(p, fetcher);
      if (found.length === 1) {
        setBusy(false);
        return brief1(found[0]!.id);
      }
      setChoices(found);
    } catch (err) {
      fail(err);
    } finally {
      setBusy(false);
    }
  }

  return (
    <section aria-label={t("Handover")} className="handover">
      <h2>{t("Handover")}</h2>
      <p>{t("For someone leaving or changing role: a brief of the documents they own that you can open.")}</p>
      <form onSubmit={submit}>
        <label htmlFor={inputId}>{t("Person")}</label>
        <input id={inputId} aria-describedby={hintId} value={person} maxLength={80} onChange={(e) => setPerson(e.target.value)} />
        <p id={hintId} className="meta">
          {t("Their name, email or user id. You can brief yourself, someone you manage, or a member of a space you steward.")}
        </p>
        <button type="submit" disabled={busy || person.trim() === ""}>
          {busy ? t("Briefing…") : t("Brief")}
        </button>
      </form>
      {error && <ErrorBox error={error} />}
      {choices && choices.length === 0 && <p className="notice">{t("No one you may brief matches that.")}</p>}
      {choices && choices.length > 0 && (
        <ul className="saved-list" aria-label={t("People you may brief")}>
          {choices.map((c) => (
            <li key={c.id}>
              <button type="button" onClick={() => brief1(c.id)}>
                {t("Brief {name}", { name: c.name })}
                {c.email !== c.name ? ` (${c.email})` : ""}
                {c.title ? `, ${c.title}` : ""}
              </button>
            </li>
          ))}
        </ul>
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
    <article className="answer" aria-label={t("Handover brief of {person}", { person: brief.person })} aria-live="polite">
      {[...groups].map(([where, claims]) => (
        <section key={where} aria-label={where || t("Other documents")}>
          <h3>{where || t("Other documents")}</h3>
          <ul className="claims">
            {claims.map((c, i) => (
              <ClaimView key={i} claim={c} trace={brief.trace} />
            ))}
          </ul>
        </section>
      ))}
      {brief.dropped > 0 && (
        <p className="meta">{tn(brief.dropped, "{n} line was left out because no document supported it.", "{n} lines were left out because no document supported them.")}</p>
      )}
      {brief.hidden > 0 && (
        <p className="refused" role="note">
          {tn(
            brief.hidden,
            "{n} of their documents is not in this brief because you can't open it. Ask the owner of each source to move access to whoever takes the work over before {person} leaves: once their account is removed, their access goes with it.",
            "{n} of their documents are not in this brief because you can't open them. Ask the owner of each source to move access to whoever takes the work over before {person} leaves: once their account is removed, their access goes with it.",
            { person: brief.person },
          )}
        </p>
      )}
      <p className="trace">{t("Trace {trace}", { trace: brief.trace })}</p>
    </article>
  );
}
