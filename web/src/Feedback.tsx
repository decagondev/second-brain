import { useId, useState } from "react";
import type { AskResponse, Fetch } from "./api";
import { AskError, reportAnswer, sendFeedback } from "./api";
import { errorText, t } from "./i18n";

// Feedback on one answer (sb-ask(3), POST /v1/feedback): a rating, optionally the citation
// that was wrong, and a short comment. Stored against the answer's trace.
export function FeedbackBar({ answer, fetcher, question, space }: { answer: AskResponse; fetcher?: Fetch; question?: string; space?: string }) {
  const [rating, setRating] = useState<"up" | "down">();
  const [wrong, setWrong] = useState("");
  const [outdated, setOutdated] = useState("");
  const [comment, setComment] = useState("");
  const [status, setStatus] = useState<string>();
  const ids = { wrong: useId(), outdated: useId(), comment: useId(), note: useId() };
  const [note, setNote] = useState("");
  const [reported, setReported] = useState<string>();
  const [sent, setSent] = useState(false);
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

  // Report this answer (SB-3606): sent only by this click, and it names the asker.
  async function report() {
    if (!question) return;
    try {
      const r = await reportAnswer(question, answer.trace, note, fetcher, space);
      setSent(true);
      setReported(t("Sent to the stewards of {space}.", { space: r.to }));
    } catch (e) {
      setReported(e instanceof AskError ? errorText(e.error).message : t("The report couldn't be sent."));
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
      {rating === "down" && question && (
        <div className="report-answer">
          <p className="meta">{t("The stewards will see your name, the question and this answer.")}</p>
          <label htmlFor={ids.note}>{t("Note for the stewards (optional)")}</label>
          <input id={ids.note} type="text" maxLength={500} value={note} onChange={(e) => setNote(e.target.value)} disabled={sent} />
          <button type="button" onClick={report} disabled={sent}>
            {t("Send to the stewards")}
          </button>
          {reported && <p role="status">{reported}</p>}
        </div>
      )}
    </div>
  );
}
