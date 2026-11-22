import { useEffect, useState } from "react";
import { startHere } from "./api";
import type { Fetch, SearchResult } from "./api";
import { PersonName } from "./Answer";

// Start here: a space's reading list, each item one the member can open now (SB-1706).
export function StartHere({ space, fetcher }: { space: string; fetcher?: Fetch }) {
  const [list, setList] = useState<{ items: SearchResult[]; hidden: number }>({ items: [], hidden: 0 });
  useEffect(() => {
    let live = true;
    startHere(space, fetcher).then((l) => live && setList(l));
    return () => {
      live = false;
    };
  }, [space, fetcher]);
  if (list.items.length === 0 && list.hidden === 0) return null;
  return (
    <section aria-label="Start here" className="start">
      <h2>Start here</h2>
      <ol className="results">
        {list.items.map((r) => (
          <li key={r.cite}>
            {r.url ? (
              <a href={r.url} target="_blank" rel="noopener noreferrer" aria-label={`${r.title ?? r.cite}, source ${r.cite}, opens in a new tab`}>
                {r.title ?? r.cite}
              </a>
            ) : (
              <span>{r.title ?? r.cite}</span>
            )}
            {r.owner && (
              <span className="meta">
                {" "}
                · Owner <PersonName person={r.owner} />
              </span>
            )}
          </li>
        ))}
      </ol>
      {list.hidden > 0 && <p className="refused">{list.hidden === 1 ? "1 item is" : `${list.hidden} items are`} not shown because you can't open them.</p>}
    </section>
  );
}
