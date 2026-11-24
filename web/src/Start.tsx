import { useEffect, useState } from "react";
import { changes, spacePeople, startHere } from "./api";
import type { Fetch, SearchResult, SpacePeople } from "./api";
import { PersonName } from "./Answer";
import { t, tj, tn } from "./i18n";

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
    <section aria-label={t("Start here")} className="start">
      <h2>{t("Start here")}</h2>
      <ol className="results">
        {list.items.map((r) => (
          <li key={r.cite}>
            {r.url ? (
              <a href={r.url} target="_blank" rel="noopener noreferrer" aria-label={t("{name}, source {cite}, opens in a new tab", { name: r.title ?? r.cite, cite: r.cite })}>
                {r.title ?? r.cite}
              </a>
            ) : (
              <span>{r.title ?? r.cite}</span>
            )}
            {r.owner && (
              <span className="meta">
                {" "}
                · {tj("Owner {person}", { person: <PersonName person={r.owner} /> })}
              </span>
            )}
          </li>
        ))}
      </ol>
      {list.hidden > 0 && <p className="refused">{tn(list.hidden, "{n} item is not shown because you can't open it.", "{n} items are not shown because you can't open them.")}</p>}
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
    <section aria-label={t("What changed this week")} className="start">
      <h2>{t("What changed this week")}</h2>
      <ul className="results">
        {items.map((r) => (
          <li key={r.cite}>
            {r.url ? (
              <a href={r.url} target="_blank" rel="noopener noreferrer" aria-label={t("{name}, source {cite}, opens in a new tab", { name: r.title ?? r.cite, cite: r.cite })}>
                {r.title ?? r.cite}
              </a>
            ) : (
              <span>{r.title ?? r.cite}</span>
            )}
            {r.age_days !== undefined && <span className="meta"> · {r.age_days === 0 ? t("today") : tn(r.age_days, "{n} day ago", "{n} days ago")}</span>}
          </li>
        ))}
      </ul>
    </section>
  );
}

// Who's who in a space (SB-3906): its stewards and the owners of its most-cited documents
// the member can open, and how many members it has, never who.
export function WhosWho({ space, fetcher }: { space: string; fetcher?: Fetch }) {
  const [p, setP] = useState<SpacePeople>();
  useEffect(() => {
    let live = true;
    spacePeople(space, fetcher).then((x) => live && setP(x));
    return () => {
      live = false;
    };
  }, [space, fetcher]);
  if (!p || (p.stewards.length === 0 && p.owners.length === 0)) return null;
  return (
    <section aria-label={t("Who's who")} className="start">
      <h2>{t("Who's who")}</h2>
      <p className="meta">{tn(p.members, "{n} member", "{n} members")}</p>
      {p.stewards.length > 0 && (
        <p>
          {tj("Stewards: {people}", {
            people: (
              <>
                {p.stewards.map((s, i) => (
                  <span key={s.email}>
                    {i > 0 && ", "}
                    <PersonName person={s} />
                  </span>
                ))}
              </>
            ),
          })}
        </p>
      )}
      {p.owners.length > 0 && (
        <ul aria-label={t("Owners of the most-cited documents")}>
          {p.owners.map((o) => (
            <li key={o.email}>
              <PersonName person={o} /> <span className="meta">· {tn(o.cites, "cited {n} time this month", "cited {n} times this month")}</span>
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}
