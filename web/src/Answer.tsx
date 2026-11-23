import { useState } from "react";
import type { AskResponse, Cite, Claim, Fetch, Notice, Person, Term } from "./api";
import { isCited, requestAccess } from "./api";
import { msg, t, tj, tn } from "./i18n";
import { numberCites, numberOf } from "./Copy";

const notices: Record<Exclude<Notice, null>, string> = {
  no_access: msg("Some sources you don't have access to may answer this. Ask their owner."),
  no_sources: msg("Nothing indexed answers this yet."),
  most_uncited: msg("Most of this answer isn't backed by your sources."),
};

// "Finance · restricted": the spaces that hold a source and its label, in words.
export function placeOf(cite: { spaces?: string[]; label?: string }): string {
  const spaces = (cite.spaces ?? []).map((s) => s.charAt(0).toUpperCase() + s.slice(1)).join(", ");
  return [spaces, cite.label ?? ""].filter((x) => x !== "").join(" · ");
}

export function CitationChip({ cite, trace }: { cite: Cite; trace: string }) {
  const flagged = cite.flags.includes("injection_suspect");
  const place = placeOf(cite);
  const age = cite.stale && cite.age_days !== undefined ? tn(cite.age_days, "{n} day old", "{n} days old") : "";
  const label = [
    t("Source {cite}", { cite: cite.cite }),
    cite.title ?? "",
    place,
    age,
    flagged ? t("flagged") : "",
    cite.url ? t("opens in a new tab") : t("shows the answer's trace"),
  ]
    .filter((x) => x !== "")
    .join(", ");
  const href = cite.url ?? `/v1/traces/${encodeURIComponent(trace)}`;
  return (
    <a className="chip" href={href} target={cite.url ? "_blank" : undefined} rel={cite.url ? "noopener noreferrer" : undefined} aria-label={label}>
      <span>{cite.title ?? cite.cite}</span>
      {place && <span className="place">{place}</span>}
      {age && <span className="place">{age}</span>}
      {flagged && <span className="flag">{t("Flagged")}</span>}
    </a>
  );
}

export function ClaimView({ claim, trace, nums }: { claim: Claim; trace: string; nums?: { cite: Cite; n: number }[] }) {
  if (!isCited(claim)) {
    return (
      <li className="claim uncited" title={t("No source you can open supports this.")}>
        <span className="badge">{t("Uncited")}</span> <span>{claim.text}</span>
      </li>
    );
  }
  return (
    <li className="claim cited">
      <span>{claim.text}</span>{" "}
      <span className="chips">
        {claim.cites.map((c, i) => (
          <CitationChip key={`${c.cite}-${i}`} cite={c} trace={trace} />
        ))}
      </span>
      {nums && <span className="print-only"> {claim.cites.map((c) => `[${numberOf(nums, c.cite)}]`).join(" ")}</span>}
    </li>
  );
}

// "ann@acme-logistics.example (Support lead)": an owner as the asker sees them.
export function PersonName({ person }: { person: Person }) {
  return (
    <span className="person">
      {person.email}
      {person.title ? ` (${person.title})` : ""}
    </span>
  );
}

// Who to ask (SB-1702): each owner with the documents of theirs this answer cites, and
// only those, since a cite is only ever a document the asker may open.
export function PeopleView({ answer }: { answer: AskResponse }) {
  const people = answer.people ?? [];
  if (people.length === 0) return null;
  const owned = (email: string) => {
    const names: string[] = [];
    for (const c of answer.answer.claims) {
      if (!isCited(c)) continue;
      for (const ci of c.cites) {
        const n = ci.title ?? ci.cite;
        if (ci.owner?.email === email && !names.includes(n)) names.push(n);
      }
    }
    return names;
  };
  return (
    <ul className="people" aria-label={t("Who to ask")}>
      {people.map((p) => (
        <li key={p.email}>
          {tj("Ask {person}, owner of {documents}", {
            person: <PersonName person={p} />,
            documents: owned(p.email).map((n, i) => (
              <span key={n}>
                {i > 0 && ", "}
                <em>{n}</em>
              </span>
            )),
          })}
        </li>
      ))}
    </ul>
  );
}

// The space's glossary terms a question used (SB-2203), at the foot of an answer or results.
export function TermsView({ terms }: { terms?: Term[] }) {
  if (!terms || terms.length === 0) return null;
  return (
    <p className="terms">
      {tj("Read with this space's glossary: {terms}.", {
        terms: terms.map((x, i) => (
          <span key={x.term}>
            {i > 0 && "; "}
            {tj("{term} means {means}", { term: <abbr title={x.means}>{x.term}</abbr> }, { means: x.means })}
          </span>
        )),
      })}
    </p>
  );
}

// Ask the owners for access (SB-2205): the owners hear which document; the asker sees counts.
export function AccessRequest({ trace, fetcher }: { trace: string; fetcher?: Fetch }) {
  const [state, setState] = useState<"idle" | "busy" | "failed" | { sent: number; already: number; no_owner: number }>("idle");
  if (typeof state === "object") {
    const parts = [
      state.sent > 0 ? tn(state.sent, "asked {n} owner", "asked {n} owners") : "",
      state.already > 0 ? t("{n} already asked this week", { n: state.already }) : "",
      state.no_owner > 0 ? t("{n} with no owner to ask", { n: state.no_owner }) : "",
    ].filter((p) => p !== "");
    return <span role="status">{parts.length > 0 ? t("Done: {parts}.", { parts: parts.join(", ") }) : t("There was nobody to ask.")}</span>;
  }
  return (
    <>
      <button
        type="button"
        disabled={state === "busy"}
        onClick={async () => {
          setState("busy");
          try {
            setState(await requestAccess(trace, fetcher));
          } catch {
            setState("failed");
          }
        }}
      >
        {t("Ask the owners for access")}
      </button>
      {state === "failed" && <span role="alert"> {t("The owners couldn't be asked just now.")}</span>}
    </>
  );
}

// Footnotes is the answer's cites, numbered, for print only (SB-2402).
function Footnotes({ nums }: { nums: { cite: Cite; n: number }[] }) {
  if (nums.length === 0) return null;
  return (
    <section className="print-only footnotes" aria-hidden="true">
      <h3>{t("Sources")}</h3>
      <ol>
        {nums.map(({ cite, n }) => (
          <li key={cite.cite} value={n}>
            {cite.title ?? cite.cite}
            {cite.url ? `, ${cite.url}` : ""}
            {cite.age_days !== undefined ? `, ${tn(cite.age_days, "{n} day old", "{n} days old")}` : ""}
          </li>
        ))}
      </ol>
    </section>
  );
}

export function AnswerView({ answer, fetcher }: { answer: AskResponse; fetcher?: Fetch }) {
  const nums = numberCites(answer);
  return (
    <article className="answer" aria-label={t("Answer")}>
      {answer.notice && (
        <p className="notice" role="note">
          {t(notices[answer.notice])}
        </p>
      )}
      <ul className="claims">
        {answer.answer.claims.map((c, i) => (
          <ClaimView key={i} claim={c} trace={answer.trace} nums={nums} />
        ))}
      </ul>
      <PeopleView answer={answer} />
      <TermsView terms={answer.terms} />
      {answer.refused > 0 && (
        <p className="refused">
          {tn(answer.refused, "{n} source was not shown because you can't open it.", "{n} sources were not shown because you can't open them.")}{" "}
          <AccessRequest trace={answer.trace} fetcher={fetcher} />
        </p>
      )}
      <p className="trace">{t("Trace {trace}", { trace: answer.trace })}</p>
      <Footnotes nums={nums} />
    </article>
  );
}
