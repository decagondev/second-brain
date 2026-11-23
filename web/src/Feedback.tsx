import { useId, useState } from "react";
import type { AskResponse, Fetch } from "./api";
import { AskError, sendFeedback } from "./api";
import { errorText, t } from "./i18n";

// Feedback on one answer (sb-ask(3), POST /v1/feedback): a rating, optionally the citation
// that was wrong, and a short comment. Stored against the answer's trace.
export function FeedbackBar({ answer, fetcher }: { answer: AskResponse; fetcher?: Fetch }) {
  const [rating, setRating] = useState<"up" | "down">();
  const [wrong, setWrong] = useState("");
  const [outdated, setOutdated] = useState("");
  const [comment, setComment] = useState("");
  const [status, setStatus] = useState<string>();
  const ids = { wrong: useId(), outdated: useId(), comment: useId() };
  const cites = Array.from(new Set(answer.answer.claims.flatMap((c) => (c.uncited ? [] : (c.cites ?? []).map((x) => x.cite)))));

  async function send(r: "up" | "down") {
    setRating(r);
    try {
      await sendFeedback({ trace: answer.trace, rating: r, ...(wrong ? { wrong_citation: wrong } : {}), ...(outdated ? { outdated_citation: outdated } : {}), ...(comment.trim() ? { comment: comment.trim() } : {}) }, fetcher);
      setStatus(t("Thanks, your feedback was saved."));
    } catch (e) {
      setStatus(e instanceof AskError ? errorText(e.error).message : t("Feedback couldn't be sent."));
    }
  }

  return (
    <div className="feedback" role="group" aria-label={t("Feedback on this answer")}>
      <button type="button" aria-pressed={rating === "up"} onClick={() => send("up")}>
        {t("Helpful")}
      </button>
      <button type="button" aria-pressed={rating === "down"} onClick={() => send("down")}>
        {t("Not helpful")}
      </button>
      {cites.length > 0 && (
        <>
          <label htmlFor={ids.wrong}>{t("Wrong citation")}</label>
          <select id={ids.wrong} value={wrong} onChange={(e) => setWrong(e.target.value)}>
            <option value="">{t("None")}</option>
            {cites.map((c) => (
              <option key={c} value={c}>
                {c}
              </option>
            ))}
          </select>
          <label htmlFor={ids.outdated}>{t("Out of date")}</label>
          <select id={ids.outdated} value={outdated} onChange={(e) => setOutdated(e.target.value)}>
            <option value="">{t("None")}</option>
            {cites.map((c) => (
              <option key={c} value={c}>
                {c}
              </option>
            ))}
          </select>
        </>
      )}
      <label htmlFor={ids.comment}>{t("Comment")}</label>
      <input id={ids.comment} type="text" maxLength={2000} value={comment} onChange={(e) => setComment(e.target.value)} />
      {status && (
        <p className="feedback-status" role="status">
          {status}
        </p>
      )}
    </div>
  );
}
