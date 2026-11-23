import { useEffect, useId, useState } from "react";
import type { FormEvent } from "react";
import { AskError, search } from "./api";
import type { ApiError, Fetch, SearchResponse } from "./api";
import { AccessRequest, placeOf, PersonName, TermsView } from "./Answer";
import { errorText, t, tj, tn } from "./i18n";

// Documents: search without the model (sb-ask(3), Search; SB-1701). Every result is a
// document the asker can open now, with a quote of the source and its cite.
export function DocumentsView({ fetcher, space, query }: { fetcher?: Fetch; space?: string; query?: { text: string; n: number } }) {
  const [q, setQ] = useState("");
  const [busy, setBusy] = useState(false);
  const [res, setRes] = useState<SearchResponse>();
  const [error, setError] = useState<ApiError>();
  const id = useId();

  async function run(text: string) {
    const want = text.trim();
    if (!want || busy) return;
    setBusy(true);
    setError(undefined);
    try {
      setRes(await search(want, fetcher, space));
    } catch (e) {
      setRes(undefined);
      setError(e instanceof AskError ? e.error : { code: "EUNAVAIL", message: t("Search is unavailable right now."), trace: "", remedy: "" });
    } finally {
      setBusy(false);
    }
  }

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
        <button type="submit" disabled={busy || q.trim() === ""}>
          {busy ? t("Searching…") : t("Search")}
        </button>
      </form>
      <div aria-live="polite" aria-busy={busy}>
        {error && (
          <div className="error" role="alert">
            <p>{errorText(error).message}</p>
            {error.trace && <p>{t("Quote trace {trace} to support.", { trace: error.trace })}</p>}
          </div>
        )}
        {res && res.results.length === 0 && <p className="notice">{t("Nothing you can open matches this.")}</p>}
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
