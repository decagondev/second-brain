import type { Clarify } from "./api";
import { t } from "./i18n";

// ClarifyView is the one question an ambiguous ask gets (SB-4901): its choices and "just
// answer". Choosing asks the same question again with the choice.
export function ClarifyView({ clarify, busy, onChoose }: { clarify: Clarify; busy: boolean; onChoose: (choice: string) => void }) {
  return (
    <section className="clarify" aria-label={t("Before answering")}>
      <p>{clarify.question}</p>
      <ul className="choices">
        {clarify.choices.map((c) => (
          <li key={c}>
            <button type="button" disabled={busy} onClick={() => onChoose(c)}>
              {c === "just answer" ? t("Just answer") : c}
            </button>
          </li>
        ))}
      </ul>
    </section>
  );
}
