import { useState } from "react";
import type { AskResponse, Fetch, HandoffDraft } from "./api";
import { citeName } from "./api";
import { ClaimView } from "./Answer";
import { call } from "./Steward";
import { msg, t, tn } from "./i18n";

const why: Record<string, string> = {
  no_match: msg("the question didn't use its terms"),
  no_agent: msg("it has no department agent"),
  paused: msg("its agent is paused"),
  budget: msg("its budget is spent"),
  over_limit: msg("at most 6 teams are asked"),
};

// Ask each department (SB-4501): a section per consulted team, each its own agent's answer
// as the asker, and which teams weren't asked and why. Teams the asker isn't in are counted.
export function DepartmentsView({ answer, fetcher }: { answer: AskResponse; fetcher?: Fetch }) {
  if (!answer.departments && !answer.not_consulted) return null;
  const others = (answer.not_consulted ?? []).filter((n) => n.space);
  const outside = (answer.not_consulted ?? []).find((n) => n.why === "not_member")?.count ?? 0;
  return (
    <section className="departments" aria-label={t("Each department")}>
      {(answer.departments ?? []).map((d) => (
        <div key={d.space} className="department">
          <h3>{t("{space} ({agent})", { space: d.space, agent: d.agent })}</h3>
          <ul className="claims">
            {d.claims.map((c, i) => (
              <ClaimView key={i} claim={c} trace={answer.trace} fetcher={fetcher} />
            ))}
          </ul>
        </div>
      ))}
      {(answer.departments ?? []).length === 0 && <p className="meta">{t("No team was asked.")}</p>}
      {others.length > 0 && (
        <ul className="meta" aria-label={t("Not asked")}>
          {others.map((n) => (
            <li key={n.space}>{t("{space} wasn't asked: {why}", { space: n.space!, why: why[n.why] ? t(why[n.why]!) : n.why })}</li>
          ))}
        </ul>
      )}
      {outside > 0 && <p className="meta">{tn(outside, "{n} team you aren't in wasn't asked.", "{n} teams you aren't in weren't asked.")}</p>}
    </section>
  );
}

// Hand off (SB-4502): the drafted request to another space, the cites to share ticked by the
// asker. Nothing is sent until they press Send.
export function HandoffView({ answer, question, fetcher = fetch }: { answer: AskResponse; question?: string; fetcher?: Fetch }) {
  const [draft, setDraft] = useState<HandoffDraft>();
  const [text, setText] = useState("");
  const [share, setShare] = useState<string[]>([]);
  const [status, setStatus] = useState<string>();
  const h = answer.handoff;
  if (!h || !question) return null;
  const base = `/v1/spaces/${encodeURIComponent(h.space)}`;
  async function open() {
    try {
      const d = (await call(fetcher, "POST", `${base}/handoff`, { trace: answer.trace, question, category: h!.category })) as unknown as HandoffDraft;
      setDraft(d);
      setText(d.text);
      setShare([]);
    } catch {
      setStatus(t("The request couldn't be drafted."));
    }
  }
  async function send() {
    try {
      await call(fetcher, "POST", `${base}/requests`, { category: draft!.category, text: text.trim(), cites: share });
      setDraft(undefined);
      setStatus(t("Sent to {space}.", { space: h!.space }));
    } catch {
      setStatus(t("The request couldn't be sent."));
    }
  }
  return (
    <section className="handoff" aria-label={t("Hand off")}>
      {!draft && (
        <p className="notice" role="note">
          {t("This looks like a {category} request for {space}.", { category: h.category, space: h.space })}{" "}
          <button type="button" onClick={() => void open()}>
            {t("Hand off")}
          </button>
        </p>
      )}
      {draft && (
        <form
          onSubmit={(e) => {
            e.preventDefault();
            void send();
          }}
        >
          <p className="meta">{t("To {space}, as {category}. Nothing is sent until you press Send.", { space: draft.space, category: draft.category })}</p>
          <label>
            {t("Request")} <textarea value={text} rows={3} maxLength={4000} onChange={(e) => setText(e.target.value)} />
          </label>
          {draft.cites.length > 0 && (
            <fieldset>
              <legend>{t("Share these sources")}</legend>
              {draft.cites.map((c) => (
                <label key={c.cite}>
                  <input
                    type="checkbox"
                    checked={share.includes(c.cite)}
                    onChange={(e) => setShare((s) => (e.target.checked ? [...s, c.cite] : s.filter((x) => x !== c.cite)))}
                  />{" "}
                  {citeName(c)}
                </label>
              ))}
            </fieldset>
          )}
          <button type="submit" disabled={!text.trim()}>
            {t("Send")}
          </button>{" "}
          <button type="button" className="link" onClick={() => setDraft(undefined)}>
            {t("Cancel")}
          </button>
        </form>
      )}
      {status && <p role="status">{status}</p>}
    </section>
  );
}
