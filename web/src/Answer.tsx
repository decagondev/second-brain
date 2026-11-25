import { Fragment, useId, useState } from "react";
import type { ReactNode } from "react";
import type { About, AskResponse, Cite, Claim, Fetch, Notice, Passage, Person, Pinned, SeeAlso, Term } from "./api";
import { AskError, askOwner, citeName, docKey, isCited, passage, requestAccess } from "./api";
import { errorText, msg, t, tj, tn } from "./i18n";
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
  const age = cite.reviewed
    ? t("Reviewed {date}", { date: cite.reviewed })
    : cite.stale && cite.age_days !== undefined
      ? tn(cite.age_days, "{n} day old", "{n} days old")
      : "";
  const replaces = cite.replaces ? t("Replaces {title}", { title: cite.replaces }) : "";
  const replacedBy = cite.replaced_by ? t("Replaced by {title}", { title: cite.replaced_by.title ?? cite.replaced_by.cite }) : "";
  const label = [
    t("Source {cite}", { cite: cite.cite }),
    cite.title !== undefined ? citeName(cite) : "",
    place,
    age,
    replaces,
    replacedBy,
    flagged ? t("flagged") : "",
    cite.url ? t("opens in a new tab") : t("shows the answer's trace"),
  ]
    .filter((x) => x !== "")
    .join(", ");
  const href = cite.url ?? `/v1/traces/${encodeURIComponent(trace)}`;
  return (
    <a className="chip" href={href} target={cite.url ? "_blank" : undefined} rel={cite.url ? "noopener noreferrer" : undefined} aria-label={label}>
      <span>{citeName(cite)}</span>
      {place && <span className="place">{place}</span>}
      {age && <span className="place">{age}</span>}
      {replaces && <span className="place">{replaces}</span>}
      {replacedBy && <span className="place replaced">{replacedBy}</span>}
      {flagged && <span className="flag">{t("Flagged")}</span>}
    </a>
  );
}

// PassageView is Show passage under a chip: the cited text, fetched and checked live as the
// asker each time it is opened, shown as plain text in a quote (SB-3001, LAW-09).
export function PassageView({ cite, trace, fetcher }: { cite: Cite; trace: string; fetcher?: Fetch }) {
  const [open, setOpen] = useState(false);
  const [p, setP] = useState<Passage | null>(null);
  const [err, setErr] = useState("");
  const show = async () => {
    if (open) {
      setOpen(false);
      return;
    }
    setOpen(true);
    setP(null);
    setErr("");
    try {
      setP(await passage(trace, cite.cite, cite.span[0], fetcher));
    } catch (e) {
      const code = e instanceof AskError ? e.error.code : "EUNAVAIL";
      setErr(code === "EPERM" ? t("You can't open this source now.") : t("The passage can't be shown right now."));
    }
  };
  return (
    <span className="passage">
      <button type="button" className="link" aria-expanded={open} onClick={() => void show()}>
        {open ? t("Hide passage") : t("Show passage")}
      </button>
      {open && (
        <span className="passage-body" role="region" aria-label={t("Passage from {title}", { title: citeName(cite) })}>
          {err && <span className="notice">{err}</span>}
          {p?.gone && <span className="notice">{t("This version is no longer indexed.")}</span>}
          {p?.text && (
            <blockquote>
              {p.text}
              <cite>{citeName(cite)}</cite>
            </blockquote>
          )}
        </span>
      )}
    </span>
  );
}

// AboutButton is Ask about this document (SB-3004): the next questions search only it.
export function AboutButton({ cite, name, onAbout }: { cite: string; name: string; onAbout: (d: About) => void }) {
  return (
    <button type="button" className="link" aria-label={t("Ask about {title}", { title: name })} onClick={() => onAbout({ key: docKey(cite), name })}>
      {t("Ask about this document")}
    </button>
  );
}

// TermWord is a glossary term where a claim uses it, its meaning shown as plain text on
// hover and on keyboard focus (SB-3302).
function TermWord({ word, means }: { word: string; means: string }) {
  const id = useId();
  return (
    <span className="term" tabIndex={0} aria-describedby={id}>
      {word}
      <span role="tooltip" id={id} className="term-tip">
        {means}
      </span>
    </span>
  );
}

const escapeRe = (s: string) => s.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");

// glossed is a claim's text with each glossary term underlined where it is used, as a whole
// word: a term with a capital letter only as written, a lower-case one in any case.
export function glossed(text: string, terms?: Term[]): ReactNode {
  const ts = (terms ?? []).filter((x) => x.term.trim() !== "");
  if (ts.length === 0) return text;
  const out: ReactNode[] = [];
  let rest = text;
  let key = 0;
  for (;;) {
    let best: { at: number; len: number; term: Term } | null = null;
    for (const x of ts) {
      const flags = x.term === x.term.toLowerCase() ? "iu" : "u";
      const m = new RegExp(`(?<![\\p{L}\\p{N}])${escapeRe(x.term)}(?![\\p{L}\\p{N}])`, flags).exec(rest);
      if (m && (best === null || m.index < best.at || (m.index === best.at && m[0].length > best.len))) best = { at: m.index, len: m[0].length, term: x };
    }
    if (best === null) break;
    out.push(<Fragment key={key++}>{rest.slice(0, best.at)}</Fragment>);
    out.push(<TermWord key={key++} word={rest.slice(best.at, best.at + best.len)} means={best.term.means} />);
    rest = rest.slice(best.at + best.len);
  }
  out.push(<Fragment key={key++}>{rest}</Fragment>);
  return out;
}

export function ClaimView({
  claim,
  trace,
  nums,
  fetcher,
  onAbout,
  terms,
}: {
  claim: Claim;
  trace: string;
  nums?: { cite: Cite; n: number }[];
  fetcher?: Fetch;
  onAbout?: (d: About) => void;
  terms?: Term[];
}) {
  if (!isCited(claim)) {
    return (
      <li className="claim uncited" title={t("No source you can open supports this.")}>
        <span className="badge">{t("Uncited")}</span> <span>{glossed(claim.text, terms)}</span>
      </li>
    );
  }
  return (
    <li className="claim cited">
      {claim.disagree && <strong className="disagree">{t("Sources disagree:")} </strong>}
      <span>{glossed(claim.text, terms)}</span>{" "}
      <span className="chips">
        {claim.cites.map((c, i) => (
          <span key={`${c.cite}-${i}`} className="cite-with-passage">
            <CitationChip cite={c} trace={trace} />
            {trace.startsWith("tr_") && <PassageView cite={c} trace={trace} fetcher={fetcher} />}
            {onAbout && <AboutButton cite={c.cite} name={citeName(c)} onAbout={onAbout} />}
          </span>
        ))}
      </span>
      {nums && <span className="print-only"> {claim.cites.map((c) => `[${numberOf(nums, c.cite)}]`).join(" ")}</span>}
    </li>
  );
}

// SeeAlsoView lists the documents the search found that no claim cites (SB-3301), by title
// and age, never as claims: their text is only ever a passage the asker opens.
export function SeeAlsoView({ answer, fetcher, onAbout }: { answer: AskResponse; fetcher?: Fetch; onAbout?: (d: About) => void }) {
  const items: SeeAlso[] = answer.see_also ?? [];
  if (items.length === 0) return null;
  return (
    <section className="see-also" aria-label={t("See also")}>
      <h3>{t("See also")}</h3>
      <ul>
        {items.map((x) => {
          const cite: Cite = { cite: x.cite, title: x.title, section: x.section, span: x.span, flags: [] };
          return (
            <li key={x.cite}>
              <span>{citeName(cite)}</span>
              {x.age_days !== undefined && <span className="place"> {tn(x.age_days, "{n} day old", "{n} days old")}</span>}{" "}
              {answer.trace.startsWith("tr_") && <PassageView cite={cite} trace={answer.trace} fetcher={fetcher} />}
              {onAbout && <AboutButton cite={x.cite} name={citeName(cite)} onAbout={onAbout} />}
            </li>
          );
        })}
      </ul>
    </section>
  );
}

// "ann@acme-logistics.example (Support lead)": an owner as the asker sees them.
export function PersonName({ person }: { person: Person }) {
  return (
    <span className="person">
      {person.email}
      {person.title ? ` (${person.title})` : ""}
      {person.caretaker ? ` · ${t("caretaker")}` : ""}
    </span>
  );
}

// Who to ask (SB-1702): each owner with the documents of theirs this answer cites, and
// only those, since a cite is only ever a document the asker may open.
export function PeopleView({ answer, onOwner, ask }: { answer: AskResponse; onOwner?: (email: string) => void; ask?: { question: string; space?: string; fetcher?: Fetch } }) {
  const [asked, setAsked] = useState<Record<string, string>>({});
  async function askThem(email: string) {
    if (!ask) return;
    try {
      await askOwner(ask.question, answer.trace, email, ask.fetcher, ask.space);
      setAsked((x) => ({ ...x, [email]: t("Sent to {person}. Their reply shows under Questions.", { person: email }) }));
    } catch (e) {
      setAsked((x) => ({ ...x, [email]: e instanceof AskError ? errorText(e.error).message : t("The question couldn't be sent.") }));
    }
  }
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
          {onOwner && (
            <>
              {" "}
              <button type="button" className="link" aria-label={t("Documents of {person}", { person: p.email })} onClick={() => onOwner(p.email)}>
                {t("Their documents")}
              </button>
            </>
          )}
          {ask && (
            <>
              {" "}
              <button type="button" className="link" aria-label={t("Ask {person} this question", { person: p.email })} disabled={asked[p.email] !== undefined} onClick={() => askThem(p.email)}>
                {t("Ask them")}
              </button>
              {asked[p.email] && <span role="status"> {asked[p.email]}</span>}
            </>
          )}
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

// Freshness warns above the answer that most of its sources are stale, or that a source
// is behind on syncing (SB-2701, SB-2704).
function Freshness({ answer }: { answer: AskResponse }) {
  const s = answer.notice_stale;
  return (
    <>
      {answer.notice_single && (
        <p className="notice" role="note">
          {t("This answer rests on one document. Check it if it matters.")}
        </p>
      )}
      {s && (
        <p className="notice" role="note">
          {tn(
            s.oldest_days,
            "Most of the sources for this answer haven't been reviewed in a while (the oldest is {n} day old). Check with their owner before relying on it.",
            "Most of the sources for this answer haven't been reviewed in a while (the oldest is {n} days old). Check with their owner before relying on it.",
          )}
        </p>
      )}
      {(answer.notice_sync ?? []).map((n) => (
        <p key={n.kind} className="notice" role="note">
          {n.last_success
            ? t("{kind} last synced at {time}; newer changes may be missing.", {
                kind: n.kind,
                time: new Date(n.last_success).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" }),
              })
            : t("{kind}: when it last synced is unknown; newer changes may be missing.", { kind: n.kind })}
        </p>
      ))}
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
            {citeName(cite)}
            {cite.url ? `, ${cite.url}` : ""}
            {cite.age_days !== undefined ? `, ${tn(cite.age_days, "{n} day old", "{n} days old")}` : ""}
          </li>
        ))}
      </ol>
    </section>
  );
}

// PinnedView is the stewards' verified answer, first (SB-3601); every cite was checked as the asker.
function PinnedView({ pin }: { pin: Pinned }) {
  return (
    <section className="pinned" aria-label={t("Verified by the stewards")}>
      <h3>{t("Verified by the stewards on {date}", { date: pin.verified })}</h3>
      <p className="meta">{pin.question}</p>
      <p>{pin.text}</p>
      <ul className="cites">
        {pin.cites.map((c) => (
          <li key={c}>
            <code>{c}</code>
          </li>
        ))}
      </ul>
      <h3>{t("The brain's answer")}</h3>
    </section>
  );
}

export function AnswerView({
  answer,
  fetcher,
  onAbout,
  onOwner,
  question,
  space,
  onSuggest,
}: {
  answer: AskResponse;
  fetcher?: Fetch;
  onAbout?: (d: About) => void;
  onOwner?: (email: string) => void;
  // Ask the suggested department's agent there (SB-4007).
  onSuggest?: (space: string, agent: string) => void;
  // The question asked and its space, for Ask them (SB-3706).
  question?: string;
  space?: string;
}) {
  const nums = numberCites(answer);
  return (
    <article className="answer" aria-label={t("Answer")}>
      {answer.pinned && <PinnedView pin={answer.pinned} />}
      {answer.suggest && onSuggest && (
        <p className="notice" role="note">
          {t("{space} has an agent for this: {agent}.", { space: answer.suggest.space, agent: answer.suggest.agent })}{" "}
          <button type="button" onClick={() => onSuggest(answer.suggest!.space, answer.suggest!.agent)}>
            {t("Ask {agent} in {space}", { space: answer.suggest.space, agent: answer.suggest.agent })}
          </button>
        </p>
      )}
      <Freshness answer={answer} />
      {answer.notice && (
        <p className="notice" role="note">
          {t(notices[answer.notice])}
        </p>
      )}
      <ul className="claims">
        {answer.answer.claims.map((c, i) => (
          <ClaimView key={i} claim={c} trace={answer.trace} nums={nums} fetcher={fetcher} onAbout={onAbout} terms={answer.terms} />
        ))}
      </ul>
      {answer.verified && answer.verified.of > 0 && (
        <p className="meta verified-count">{t("{marked} of {of} sources verified", { marked: answer.verified.marked, of: answer.verified.of })}</p>
      )}
      <SeeAlsoView answer={answer} fetcher={fetcher} onAbout={onAbout} />
      <PeopleView answer={answer} onOwner={onOwner} ask={question ? { question, space, fetcher } : undefined} />
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
