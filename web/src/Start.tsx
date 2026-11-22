import { useEffect, useState } from "react";
import { changes, startHere } from "./api";
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

// What changed this week (SB-1708): on the empty answer page, documents of the person's
// spaces with a new version, each one they can open now.
export function WhatChanged({ space, fetcher }: { space?: string; fetcher?: Fetch }) {
  const [items, setItems] = useState<SearchResult[]>([]);
  useEffect(() => {
    let live = true;
    changes(7, fetcher, space).then((l) => live && setItems(l));
    return () => {
      live = false;
    };
  }, [space, fetcher]);
  if (items.length === 0) return null;
  return (
    <section aria-label="What changed this week" className="start">
      <h2>What changed this week</h2>
      <ul className="results">
        {items.map((r) => (
          <li key={r.cite}>
            {r.url ? (
              <a href={r.url} target="_blank" rel="noopener noreferrer" aria-label={`${r.title ?? r.cite}, source ${r.cite}, opens in a new tab`}>
                {r.title ?? r.cite}
              </a>
            ) : (
              <span>{r.title ?? r.cite}</span>
            )}
            {r.age_days !== undefined && <span className="meta"> · {r.age_days === 0 ? "today" : r.age_days === 1 ? "1 day ago" : `${r.age_days} days ago`}</span>}
          </li>
        ))}
      </ul>
    </section>
  );
}
