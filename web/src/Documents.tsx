import { useEffect, useId, useState } from "react";
import type { FormEvent } from "react";
import { AskError, markOwnReviewed, search } from "./api";
import type { About, ApiError, Fetch, Filters, SearchResponse } from "./api";
import { AboutButton, AccessRequest, placeOf, PersonName, TermsView } from "./Answer";
import { errorText, msg, t, tj, tn } from "./i18n";
import { setVerifiedOnly, verifiedOnly } from "./recent";
import { ReviewAgain } from "./Questions";

// The filters (SB-2404), as the API names them.
const types: { id: string; name: string }[] = [
  { id: "", name: msg("Any type") },
  { id: "pdf", name: msg("PDF") },
  { id: "doc", name: msg("Word documents") },
  { id: "sheet", name: msg("Spreadsheets") },
  { id: "slides", name: msg("Slides") },
  { id: "web", name: msg("Web pages") },
  { id: "text", name: msg("Text files") },
  { id: "other", name: msg("Everything else") },
];
const ages: { days: number; name: string }[] = [
  { days: 0, name: msg("Any time") },
  { days: 7, name: msg("The last 7 days") },
  { days: 30, name: msg("The last 30 days") },
  { days: 90, name: msg("The last 90 days") },
  { days: 365, name: msg("The last year") },
];

// Documents: search without the model (sb-ask(3), Search; SB-1701). Every result is a
// document the asker can open now, with a quote of the source and its cite.
export function DocumentsView({
  fetcher,
  space,
  query,
  onAsk,
  onAbout,
  owner,
}: {
  fetcher?: Fetch;
  space?: string;
  query?: { text: string; n: number };
  onAsk?: (q: string) => void;
  onAbout?: (d: About) => void;
  owner?: { email: string; n: number };
}) {
  const [ownerOf, setOwnerOf] = useState(""); // Their documents (SB-3306)
  const [q, setQ] = useState("");
  const [type, setType] = useState("");
  const [since, setSince] = useState(0);
  const [verified, setVerified] = useState<boolean>(() => verifiedOnly("documents")); // SB-3602
  const [wide, setWide] = useState(false); // searching the whole company from a space (SB-2406)
  const [ran, setRan] = useState<{ text: string; filtered: boolean; inSpace: boolean }>();
  const typeId = useId();
  const sinceId = useId();
  const [busy, setBusy] = useState(false);
  const [res, setRes] = useState<SearchResponse>();
  const [error, setError] = useState<ApiError>();
  const id = useId();

  useEffect(() => setWide(false), [space]);

  async function run(text: string, filters: Filters = { type, since, owner: ownerOf, verified }, whole = wide) {
    const want = text.trim();
    if ((!want && !filters.owner) || busy) return;
    setBusy(true);
    setError(undefined);
    const where = whole ? undefined : space;
    setRan({ text: want, filtered: !!filters.type || !!filters.since || !!filters.owner, inSpace: !!where });
    try {
      setRes(await search(want, fetcher, where, filters));
    } catch (e) {
      setRes(undefined);
      setError(e instanceof AskError ? e.error : { code: "EUNAVAIL", message: t("Search is unavailable right now."), trace: "", remedy: "" });
    } finally {
      setBusy(false);
    }
  }

  useEffect(() => {
    if (owner?.email) {
      setOwnerOf(owner.email);
      setQ("");
      void run("", { type, since, owner: owner.email, verified });
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [owner?.n]);

  // An ask that failed offers the same text as a search (sb-ask(3), The web UI).
  useEffect(() => {
    if (query?.text) {
      setQ(query.text);
      void run(query.text);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [query?.n]);

  function submit(e: FormEvent) {
    e.preventDefault();
    void run(q);
  }

  return (
    <section aria-label={t("Documents")} className="documents">
      <form onSubmit={submit} className="ask" role="search">
        <label htmlFor={id}>{t("Search documents")}</label>
        <input id={id} type="search" value={q} maxLength={4000} onChange={(e) => setQ(e.target.value)} />
        <label htmlFor={typeId}>{t("File type")}</label>
        <select id={typeId} value={type} onChange={(e) => setType(e.target.value)}>
          {types.map((x) => (
            <option key={x.id} value={x.id}>
              {t(x.name)}
            </option>
          ))}
        </select>
        <label htmlFor={sinceId}>{t("Changed in")}</label>
        <select id={sinceId} value={since} onChange={(e) => setSince(Number(e.target.value))}>
          {ages.map((x) => (
            <option key={x.days} value={x.days}>
              {t(x.name)}
            </option>
          ))}
        </select>
        <label className="short">
          <input
            type="checkbox"
            checked={verified}
            onChange={(e) => {
              setVerified(e.target.checked);
              setVerifiedOnly("documents", e.target.checked);
            }}
          />{" "}
          {t("Verified only")}
        </label>
        <button type="submit" disabled={busy || (q.trim() === "" && !ownerOf)}>
          {busy ? t("Searching…") : t("Search")}
        </button>
      </form>
      {ownerOf && (
        <p className="about" role="status">
          {t("Documents owned by {person}", { person: ownerOf })}{" "}
          <button
            type="button"
            className="link"
            onClick={() => {
              setOwnerOf("");
              setRes(undefined);
            }}
          >
            {t("Clear")}
          </button>
        </p>
      )}
      <div aria-live="polite" aria-busy={busy}>
        {error && (
          <div className="error" role="alert">
            <p>{errorText(error).message}</p>
            {error.trace && <p>{t("Quote trace {trace} to support.", { trace: error.trace })}</p>}
          </div>
        )}
        {res && res.results.length === 0 && (
          <>
            <p className="notice">{t("Nothing you can open matches this.")}</p>
            {ran && (
              <NextSteps
                inSpace={ran.inSpace}
                filtered={ran.filtered}
                onWide={() => {
                  setWide(true);
                  void run(ran.text, { type, since }, true);
                }}
                onClear={() => {
                  setType("");
                  setSince(0);
                  setOwnerOf("");
                  void run(ran.text, {});
                }}
                onAsk={onAsk ? () => onAsk(ran.text) : undefined}
              />
            )}
          </>
        )}
        {res && res.results.length > 0 && (
          <ol className="results">
            {res.results.map((r) => {
              const place = placeOf(r);
              const name = r.title ?? r.cite;
              return (
                <li key={r.cite} className="result">
                  <h2 className="result-title">
                    {r.url ? (
                      <a href={r.url} target="_blank" rel="noopener noreferrer" aria-label={t("{name}, source {cite}, opens in a new tab", { name, cite: r.cite })}>
                        {name}
                      </a>
                    ) : (
                      <span>{name}</span>
                    )}
                  </h2>
                  <blockquote className="snippet" cite={r.url}>
                    {r.snippet}
                  </blockquote>
                  <p className="meta">
                    <span>{r.cite}</span>
                    {place && <span> · {place}</span>}
                    {r.age_days !== undefined && <span> · {tn(r.age_days, "{n} day old", "{n} days old")}</span>}
                    {r.owner && (
                      <span>
                        {" "}
                        · {tj("Owner {person}", { person: <PersonName person={r.owner} /> })}
                      </span>
                    )}
                    {r.flags.includes("injection_suspect") && <span className="flag"> · {t("Flagged")}</span>}
                  </p>
                  {onAbout && <AboutButton cite={r.cite} name={name} onAbout={onAbout} />}
                  {r.mine && <OwnReviewed cite={r.cite} name={name} fetcher={fetcher} />}
                </li>
              );
            })}
          </ol>
        )}
        {res && <TermsView terms={res.terms} />}
        {res && res.refused > 0 && (
          <p className="refused">
            {tn(res.refused, "{n} document was not shown because you can't open it.", "{n} documents were not shown because you can't open them.")}{" "}
            <AccessRequest trace={res.trace} fetcher={fetcher} />
          </p>
        )}
      </div>
    </section>
  );
}

// No dead ends (SB-2406): what to try next, from what the person typed and chose only, never
// from the index, so a document they can't open changes nothing here.
export function NextSteps({ inSpace, filtered, onWide, onClear, onAsk }: { inSpace: boolean; filtered: boolean; onWide: () => void; onClear: () => void; onAsk?: () => void }) {
  return (
    <div className="next-steps">
      <p>{t("You could try:")}</p>
      {inSpace && (
        <button type="button" onClick={onWide}>
          {t("Search the whole company")}
        </button>
      )}
      {filtered && (
        <button type="button" onClick={onClear}>
          {t("Clear filters")}
        </button>
      )}
      {onAsk && (
        <button type="button" onClick={onAsk}>
          {t("Ask it as a question")}
        </button>
      )}
    </div>
  );
}

// Reviewed, still right on the person's own document (SB-3106): the stewards' record.
function OwnReviewed({ cite, name, fetcher }: { cite: string; name: string; fetcher?: Fetch }) {
  const [state, setState] = useState<"idle" | "busy" | "done">("idle");
  const [error, setError] = useState("");
  const [days, setDays] = useState(0);
  async function mark() {
    setState("busy");
    setError("");
    try {
      await markOwnReviewed(cite, fetcher, days);
      setState("done");
    } catch (e) {
      setState("idle");
      setError((e as Error).message || t("The document couldn't be marked."));
    }
  }
  if (state === "done") return <p role="status">{t("Marked reviewed: {name}", { name })}</p>;
  return (
    <>
      <ReviewAgain days={days} onChange={setDays} name={name} />{" "}
      <button type="button" disabled={state === "busy"} aria-label={t("Reviewed, still right: {name}", { name })} onClick={() => void mark()}>
        {t("Reviewed, still right")}
      </button>
      {error && <span role="alert"> {error}</span>}
    </>
  );
}
