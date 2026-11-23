import { useState } from "react";
import type { AskResponse, Cite, Claim, Fetch, Notice, Person, Term } from "./api";
import { isCited, requestAccess } from "./api";

const notices: Record<Exclude<Notice, null>, string> = {
  no_access: "Some sources you don't have access to may answer this. Ask their owner.",
  no_sources: "Nothing indexed answers this yet.",
  most_uncited: "Most of this answer isn't backed by your sources.",
};

// "Finance · restricted": the spaces that hold a source and its label, in words.
export function placeOf(cite: { spaces?: string[]; label?: string }): string {
  const spaces = (cite.spaces ?? []).map((s) => s.charAt(0).toUpperCase() + s.slice(1)).join(", ");
  return [spaces, cite.label ?? ""].filter((x) => x !== "").join(" · ");
}

export function CitationChip({ cite, trace }: { cite: Cite; trace: string }) {
  const flagged = cite.flags.includes("injection_suspect");
  const place = placeOf(cite);
  const age = cite.stale && cite.age_days !== undefined ? `${cite.age_days} days old` : "";
  const label = `Source ${cite.cite}${cite.title ? `, ${cite.title}` : ""}${place ? `, ${place}` : ""}${age ? `, ${age}` : ""}${flagged ? ", flagged" : ""}${cite.url ? ", opens in a new tab" : ", shows the answer's trace"}`;
  const href = cite.url ?? `/v1/traces/${encodeURIComponent(trace)}`;
  return (
    <a className="chip" href={href} target={cite.url ? "_blank" : undefined} rel={cite.url ? "noopener noreferrer" : undefined} aria-label={label}>
      <span>{cite.title ?? cite.cite}</span>
      {place && <span className="place">{place}</span>}
      {age && <span className="place">{age}</span>}
      {flagged && <span className="flag">Flagged</span>}
    </a>
  );
}

export function ClaimView({ claim, trace }: { claim: Claim; trace: string }) {
  if (!isCited(claim)) {
    return (
      <li className="claim uncited" title="No source you can open supports this.">
        <span className="badge">Uncited</span> <span>{claim.text}</span>
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
    <ul className="people" aria-label="Who to ask">
      {people.map((p) => (
        <li key={p.email}>
          Ask <PersonName person={p} />, owner of{" "}
          {owned(p.email).map((n, i) => (
            <span key={n}>
              {i > 0 && ", "}
              <em>{n}</em>
            </span>
          ))}
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
      Read with this space's glossary:{" "}
      {terms.map((t, i) => (
        <span key={t.term}>
          {i > 0 && "; "}
          <abbr title={t.means}>{t.term}</abbr> means {t.means}
        </span>
      ))}
      .
    </p>
  );
}

// Ask the owners for access (SB-2205): the owners hear which document; the asker sees counts.
export function AccessRequest({ trace, fetcher }: { trace: string; fetcher?: Fetch }) {
  const [state, setState] = useState<"idle" | "busy" | "failed" | { sent: number; already: number; no_owner: number }>("idle");
  if (typeof state === "object") {
    const parts = [
      state.sent > 0 ? `asked ${state.sent === 1 ? "1 owner" : `${state.sent} owners`}` : "",
      state.already > 0 ? `${state.already} already asked this week` : "",
      state.no_owner > 0 ? `${state.no_owner} with no owner to ask` : "",
    ].filter((p) => p !== "");
    return <span role="status">{parts.length > 0 ? `Done: ${parts.join(", ")}.` : "There was nobody to ask."}</span>;
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
        Ask the owners for access
      </button>
      {state === "failed" && <span role="alert"> The owners couldn't be asked just now.</span>}
    </>
  );
}

export function AnswerView({ answer, fetcher }: { answer: AskResponse; fetcher?: Fetch }) {
  return (
    <article className="answer" aria-label="Answer">
      {answer.notice && <p className="notice" role="note">{notices[answer.notice]}</p>}
      <ul className="claims">
        {answer.answer.claims.map((c, i) => (
          <ClaimView key={i} claim={c} trace={answer.trace} />
        ))}
      </ul>
      <PeopleView answer={answer} />
      <TermsView terms={answer.terms} />
      {answer.refused > 0 && (
        <p className="refused">
          {answer.refused === 1 ? "1 source was" : `${answer.refused} sources were`} not shown because you can't open them.{" "}
          <AccessRequest trace={answer.trace} fetcher={fetcher} />
        </p>
      )}
      <p className="trace">Trace {answer.trace}</p>
    </article>
  );
}
