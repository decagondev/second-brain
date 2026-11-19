import type { AskResponse, Cite, Claim, Notice } from "./api";
import { isCited } from "./api";

const notices: Record<Exclude<Notice, null>, string> = {
  no_access: "Some sources you don't have access to may answer this. Ask their owner.",
  no_sources: "Nothing indexed answers this yet.",
  most_uncited: "Most of this answer isn't backed by your sources.",
};

export function CitationChip({ cite, trace }: { cite: Cite; trace: string }) {
  const flagged = cite.flags.includes("injection_suspect");
  const label = `Source ${cite.cite}${cite.title ? `, ${cite.title}` : ""}${flagged ? ", flagged" : ""}${cite.url ? ", opens in a new tab" : ", shows the answer's trace"}`;
  const href = cite.url ?? `/v1/traces/${encodeURIComponent(trace)}`;
  return (
    <a className="chip" href={href} target={cite.url ? "_blank" : undefined} rel={cite.url ? "noopener noreferrer" : undefined} aria-label={label}>
      <span>{cite.title ?? cite.cite}</span>
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

export function AnswerView({ answer }: { answer: AskResponse }) {
  return (
    <article className="answer" aria-label="Answer">
      {answer.notice && <p className="notice" role="note">{notices[answer.notice]}</p>}
      <ul className="claims">
        {answer.answer.claims.map((c, i) => (
          <ClaimView key={i} claim={c} trace={answer.trace} />
        ))}
      </ul>
      {answer.refused > 0 && (
        <p className="refused">
          {answer.refused === 1 ? "1 source was" : `${answer.refused} sources were`} not shown because you can't open them.
        </p>
      )}
      <p className="trace">Trace {answer.trace}</p>
    </article>
  );
}
