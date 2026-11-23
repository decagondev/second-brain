import { useId, useState } from "react";
import type { ReactNode } from "react";
import { lang, langs, setLang, t } from "./i18n";
import type { Lang } from "./i18n";

// The language picker (SB-2301): the manifest's language is the default; a person's choice
// is kept in their browser. Switching re-renders the whole app in the new language.
export function LanguagePicker({ onChange }: { onChange: (l: Lang) => void }) {
  const id = useId();
  return (
    <header className="language">
      <label htmlFor={id}>{t("Language")}</label>{" "}
      <select
        id={id}
        value={lang()}
        onChange={(e) => {
          setLang(e.target.value as Lang, true);
          onChange(e.target.value as Lang);
        }}
      >
        {langs.map((l) => (
          <option key={l.id} value={l.id} lang={l.id}>
            {l.name}
          </option>
        ))}
      </select>
    </header>
  );
}

// Localised renders a screen under the language picker; switching renders it again in
// place, keeping its state (a new element each time, so React doesn't skip it).
export function Localised({ render }: { render: () => ReactNode }) {
  const [, setShown] = useState(lang());
  return (
    <>
      <LanguagePicker onChange={setShown} />
      {render()}
    </>
  );
}
