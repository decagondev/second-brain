import { useEffect, useId, useState } from "react";
import type { FormEvent } from "react";
import { AskError, search } from "./api";
import type { ApiError, Fetch, SearchResponse } from "./api";
import { placeOf, PersonName } from "./Answer";

// Documents: search without the model (sb-ask(3), Search; SB-1701). Every result is a
// document the asker can open now, with a quote of the source and its cite.
export function DocumentsView({ fetcher, space, query }: { fetcher?: Fetch; space?: string; query?: { text: string; n: number } }) {
  const [q, setQ] = useState("");
  const [busy, setBusy] = useState(false);
  const [res, setRes] = useState<SearchResponse>();
  const [error, setError] = useState<ApiError>();
  const id = useId();

  async function run(text: string) {
    const t = text.trim();
    if (!t || busy) return;
    setBusy(true);
    setError(undefined);
    try {
      setRes(await search(t, fetcher, space));
    } catch (e) {
      setRes(undefined);
      setError(e instanceof AskError ? e.error : { code: "EUNAVAIL", message: "Search is unavailable right now.", trace: "", remedy: "" });
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
    <section aria-label="Documents" className="documents">
      <form onSubmit={submit} className="ask" role="search">
        <label htmlFor={id}>Search documents</label>
        <input id={id} type="search" value={q} maxLength={4000} onChange={(e) => setQ(e.target.value)} />
        <button type="submit" disabled={busy || q.trim() === ""}>
          {busy ? "Searching…" : "Search"}
        </button>
      </form>
      <div aria-live="polite" aria-busy={busy}>
        {error && (
          <div className="error" role="alert">
            <p>{error.message}</p>
            {error.trace && <p>Quote trace {error.trace} to support.</p>}
          </div>
        )}
        {res && res.results.length === 0 && <p className="notice">Nothing you can open matches this.</p>}
        {res && res.results.length > 0 && (
          <ol className="results">
            {res.results.map((r) => {
              const place = placeOf(r);
              const name = r.title ?? r.cite;
              return (
                <li key={r.cite} className="result">
                  <h2 className="result-title">
                    {r.url ? (
                      <a href={r.url} target="_blank" rel="noopener noreferrer" aria-label={`${name}, source ${r.cite}, opens in a new tab`}>
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
                    {r.age_days !== undefined && <span> · {r.age_days === 1 ? "1 day old" : `${r.age_days} days old`}</span>}
                    {r.owner && (
                      <span>
                        {" "}
                        · Owner <PersonName person={r.owner} />
                      </span>
                    )}
                    {r.flags.includes("injection_suspect") && <span className="flag"> · Flagged</span>}
                  </p>
                </li>
              );
            })}
          </ol>
        )}
        {res && res.refused > 0 && (
          <p className="refused">{res.refused === 1 ? "1 document was" : `${res.refused} documents were`} not shown because you can't open them.</p>
        )}
      </div>
    </section>
  );
}
