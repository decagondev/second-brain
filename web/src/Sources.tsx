import type { SearchResult } from "./api";
import { PersonName } from "./Answer";
import { t, tj, tn } from "./i18n";

// Sources first (sb-ask(3), The web UI; SB-2304): while the answer is written, the
// documents search allowed the asker, by title, owner and age. Never their text, so
// nothing here reads as a claim; the cited answer replaces them.
export function SourcesFirst({ sources }: { sources?: SearchResult[] }) {
  return (
    <div className="sources-first">
      <p className="notice">{t("Writing the answer…")}</p>
      {sources && sources.length > 0 && (
        <>
          <p>{t("Sources you can open, while the answer is written:")}</p>
          <ul className="sources">
            {sources.map((r) => {
              const name = r.title ?? r.cite;
              return (
                <li key={r.cite}>
                  {r.url ? (
                    <a href={r.url} target="_blank" rel="noopener noreferrer" aria-label={t("{name}, source {cite}, opens in a new tab", { name, cite: r.cite })}>
                      {name}
                    </a>
                  ) : (
                    <span>{name}</span>
                  )}
                  {r.age_days !== undefined && <span className="meta"> · {tn(r.age_days, "{n} day old", "{n} days old")}</span>}
                  {r.owner && (
                    <span className="meta">
                      {" "}
                      · {tj("Owner {person}", { person: <PersonName person={r.owner} /> })}
                    </span>
                  )}
                </li>
              );
            })}
          </ul>
        </>
      )}
    </div>
  );
}
