import { useEffect, useState } from "react";
import { docKey, search, spacePeople } from "./api";
import type { About, Fetch, Space } from "./api";
import { t } from "./i18n";

// @ in the ask box (SB-5207): after @, the spaces the asker is in, the people of the space
// (its stewards and owners) and documents they can open, each from an endpoint that checks
// as them. Picking one asks there, narrows the ask to the document, or lists the person's
// documents, and takes the @word out of the question.

// atWord is the @word the question ends with, without the @, or undefined.
export function atWord(question: string): string | undefined {
  const m = /(?:^|\s)@([^\s@]{1,60})$/.exec(question);
  return m?.[1];
}

// withoutAt is the question without its trailing @word.
export function withoutAt(question: string): string {
  return question.replace(/(?:^|\s)@[^\s@]{1,60}$/, "").trimEnd();
}

export function AtHint({
  question,
  space,
  spaces,
  fetcher = fetch,
  onSpace,
  onDocument,
  onPerson,
}: {
  question: string;
  space?: string;
  spaces: Space[];
  fetcher?: Fetch;
  onSpace: (name: string) => void;
  onDocument: (d: About) => void;
  onPerson: (email: string) => void;
}) {
  const word = atWord(question);
  const [docs, setDocs] = useState<About[]>([]);
  const [people, setPeople] = useState<string[]>([]);
  useEffect(() => {
    if (!word || word.length < 2) {
      setDocs([]);
      setPeople([]);
      return;
    }
    let live = true;
    search(word, fetcher, space || undefined)
      .then((r) => live && setDocs((r.results ?? []).slice(0, 5).map((x) => ({ key: docKey(x.cite), name: x.title ?? x.cite }))))
      .catch(() => live && setDocs([]));
    if (space) {
      spacePeople(space, fetcher).then((p) => {
        if (!live || !p) return;
        const all = [...p.stewards.map((s) => s.email), ...p.owners.map((o) => o.email)];
        setPeople([...new Set(all)].filter((e) => e.toLowerCase().startsWith(word.toLowerCase())).slice(0, 5));
      });
    }
    return () => {
      live = false;
    };
  }, [word, space, fetcher]);
  if (!word) return null;
  const sps = spaces.filter((s) => s.name.toLowerCase().startsWith(word.toLowerCase())).slice(0, 5);
  if (sps.length === 0 && docs.length === 0 && people.length === 0) return null;
  return (
    <ul className="results" aria-label={t("Suggestions for @{word}", { word })}>
      {sps.map((s) => (
        <li key={"s:" + s.name}>
          <button type="button" onClick={() => onSpace(s.name)}>
            {t("Ask in {space}", { space: s.name })}
          </button>
        </li>
      ))}
      {people.map((e) => (
        <li key={"p:" + e}>
          <button type="button" onClick={() => onPerson(e)}>
            {t("Documents of {person}", { person: e })}
          </button>
        </li>
      ))}
      {docs.map((d) => (
        <li key={"d:" + d.key}>
          <button type="button" onClick={() => onDocument(d)}>
            {t("Ask about {title}", { title: d.name })}
          </button>
        </li>
      ))}
    </ul>
  );
}
