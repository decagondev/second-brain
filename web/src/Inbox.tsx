import { useEffect, useId, useState } from "react";
import type { FormEvent } from "react";
import { AskError, forgetMemory, myInbox, myMemory, remember } from "./api";
import type { Fetch, Inbox, InboxItem, MemoryLine } from "./api";
import { errorText, msg, t } from "./i18n";

const errMsg = (e: unknown, fallback: string) => (e instanceof AskError ? errorText(e.error).message : fallback);

// What each kind of item is, in words (SB-4301).
const kinds: Record<string, string> = {
  draft: msg("Draft to publish"),
  question: msg("Question"),
  request: msg("Request"),
  policy: msg("Policy to read"),
  approval: msg("Approval"),
  action: msg("Action item"),
  fix: msg("Suggested edits"),
  research: msg("Research report"),
  routine: msg("Routine"),
};

function Items({ items, label, empty }: { items: InboxItem[]; label: string; empty: string }) {
  return (
    <section aria-label={label} className="inbox-list">
      <h2>
        {label} ({items.length})
      </h2>
      {items.length === 0 ? (
        <p className="notice">{empty}</p>
      ) : (
        <ul className="saved-list">
          {items.map((x) => (
            <li key={x.kind + x.id}>
              <span className="meta">{t(kinds[x.kind] ?? x.kind)}</span> {x.title}{" "}
              <span className="meta">
                ({x.space || t("company-wide")}
                {x.at ? `, ${x.at.slice(0, 10)}` : ""}
                {x.due ? `, ${t("due {due}", { due: x.due })}` : ""})
              </span>
              {x.overdue && <strong> {t("Overdue")}</strong>}
              {x.why && <span className="refused"> {x.why}</span>}{" "}
              <a href={x.link} aria-label={t("Open: {title}", { title: x.title })}>
                {t("Open")}
              </a>
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}

// The inbox (sb-ask(3), One inbox; SB-4301): what waits on me in every space, built when
// shown, as me. It only lists; each item opens where it's acted on.
export function InboxView({ fetcher }: { fetcher?: Fetch }) {
  const [inbox, setInbox] = useState<Inbox>();
  const [error, setError] = useState<string>();
  useEffect(() => {
    myInbox(fetcher).then(setInbox, (e) => setError(errMsg(e, t("Your inbox is unavailable right now."))));
  }, [fetcher]);
  return (
    <section aria-label={t("Inbox")} className="inbox">
      {error && (
        <p className="error" role="alert">
          {error}
        </p>
      )}
      {inbox && (
        <>
          <Items items={inbox.needs_you} label={t("Needs you")} empty={t("Nothing waits on you.")} />
          {inbox.failed.length > 0 && <Items items={inbox.failed} label={t("Failed")} empty="" />}
          <Items items={inbox.done} label={t("Done")} empty={t("Nothing done in the last 30 days.")} />
        </>
      )}
    </section>
  );
}

// About me (sb-ask(3), Memory I control; SB-4303): lines I wrote about myself, sent with my
// own asks only, never a document's text.
export function AboutMe({ fetcher }: { fetcher?: Fetch }) {
  const [on, setOn] = useState(true);
  const [lines, setLines] = useState<MemoryLine[]>();
  const [text, setText] = useState("");
  const [status, setStatus] = useState<string>();
  const id = useId();
  async function load() {
    try {
      const m = await myMemory(fetcher);
      setOn(m.on);
      setLines(m.lines);
    } catch (e) {
      setStatus(errMsg(e, t("Your memory is unavailable right now.")));
    }
  }
  useEffect(() => {
    void load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);
  async function add(e: FormEvent) {
    e.preventDefault();
    try {
      await remember(text, fetcher);
      setText("");
      setStatus(t("Remembered. Your own questions now carry it."));
      await load();
    } catch (err) {
      setStatus(errMsg(err, t("That couldn't be remembered.")));
    }
  }
  async function remove(line?: string) {
    try {
      await forgetMemory(line, fetcher);
      await load();
    } catch (err) {
      setStatus(errMsg(err, t("That couldn't be removed.")));
    }
  }
  return (
    <section aria-label={t("About me")} className="about-me">
      <h2>{t("About me")}</h2>
      {!on ? (
        <p className="notice">{t("Memory is off for your organisation.")}</p>
      ) : (
        <>
          <p className="meta">{t("A few lines about you, in your own words, sent with your own questions only. Never paste a document here.")}</p>
          {lines && lines.length > 0 && (
            <ul className="saved-list">
              {lines.map((l) => (
                <li key={l.id}>
                  {l.text}{" "}
                  <button type="button" onClick={() => remove(l.id)} aria-label={t("Remove: {question}", { question: l.text })}>
                    {t("Remove")}
                  </button>
                </li>
              ))}
            </ul>
          )}
          <form onSubmit={add}>
            <label htmlFor={id}>{t("Something about you")}</label>{" "}
            <input id={id} type="text" maxLength={200} value={text} onChange={(e) => setText(e.target.value)} />{" "}
            <button type="submit" disabled={!text.trim()}>
              {t("Add")}
            </button>
            {lines && lines.length > 0 && (
              <>
                {" "}
                <button type="button" onClick={() => remove(undefined)}>
                  {t("Remove all")}
                </button>
              </>
            )}
          </form>
        </>
      )}
      {status && (
        <p className="feedback-status" role="status">
          {status}
        </p>
      )}
    </section>
  );
}

// Remember something about me, under an answer (SB-4303): an empty field, never the
// answer's text.
export function RememberBox({ fetcher }: { fetcher?: Fetch }) {
  const [open, setOpen] = useState(false);
  const [text, setText] = useState("");
  const [status, setStatus] = useState<string>();
  const id = useId();
  async function add(e: FormEvent) {
    e.preventDefault();
    try {
      await remember(text, fetcher);
      setText("");
      setOpen(false);
      setStatus(t("Remembered. Your own questions now carry it."));
    } catch (err) {
      setStatus(errMsg(err, t("That couldn't be remembered.")));
    }
  }
  return (
    <div className="remember">
      {!open && (
        <button type="button" onClick={() => setOpen(true)}>
          {t("Remember something about me")}
        </button>
      )}
      {open && (
        <form onSubmit={add}>
          <label htmlFor={id}>{t("Something about you")}</label>{" "}
          <input id={id} type="text" maxLength={200} value={text} onChange={(e) => setText(e.target.value)} />{" "}
          <button type="submit" disabled={!text.trim()}>
            {t("Add")}
          </button>{" "}
          <button type="button" onClick={() => setOpen(false)}>
            {t("Cancel")}
          </button>
        </form>
      )}
      {status && (
        <p className="feedback-status" role="status">
          {status}
        </p>
      )}
    </div>
  );
}
