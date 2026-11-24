import { useEffect, useId, useRef, useState } from "react";
import type { FormEvent, KeyboardEvent } from "react";
import { ask, AskError, BRANCH, fileToAsk, listSpaces, maxAbout, openShare, search, spaceName, suggestedQuestions } from "./api";
import type { About, ApiError, AskFile, AskResponse, Fetch, SearchResult, SharedQuestion, Space } from "./api";
import { AnswerView } from "./Answer";
import { FeedbackBar } from "./Feedback";
import { DraftPanel } from "./Draft";
import { DocumentsView } from "./Documents";
import { SavedView, TurnTools } from "./Saved";
import { AskStewards, QuestionsView } from "./Questions";
import { StartHere, WhatChanged } from "./Start";
import { MySessions, SignOutEverywhere } from "./Session";
import { HandoverView } from "./Handover";
import { ErrorBox } from "./ErrorBox";
import { SourcesFirst } from "./Sources";
import { ApiTokens } from "./Tokens";
import { NoticeBanner } from "./Notice";
import { AnswerTools, DownloadConversation } from "./Copy";
import { forgetQuestions, recentQuestions, rememberQuestion, setShortAnswers, setVerifiedOnly, shortAnswers, verifiedOnly } from "./recent";
import { errorText, lang, msg, t, tj, tn } from "./i18n";

interface Turn {
  id: number;
  question: string;
  answer?: AskResponse;
  error?: ApiError;
  // While the answer is written: search's allowed documents, shown first (SB-2304).
  pending?: boolean;
  sources?: SearchResult[];
}

type Tab = "answer" | "documents" | "saved" | "questions" | "handover";
const tabs: { id: Tab; name: string }[] = [
  { id: "answer", name: msg("Answer") },
  { id: "documents", name: msg("Documents") },
  { id: "saved", name: msg("Saved") },
  { id: "questions", name: msg("Questions") },
  { id: "handover", name: msg("Handover") },
];

// Errors after which the same text can still be searched: search needs no model (SB-1701).
const searchable = new Set(["EUNAVAIL", "EBUDGET"]);

// tab opens the app on a tab, from a notification's link (?tab=questions; SB-2102).
// q is a search from the address bar (SB-3806): with tab documents, it runs at once.
export function App({ fetcher, share, tab: first, q: fromBar }: { fetcher?: Fetch; share?: string; tab?: string; q?: string }) {
  const [question, setQuestion] = useState("");
  const [turns, setTurns] = useState<Turn[]>([]);
  const [conversation, setConversation] = useState<string>();
  const [about, setAbout] = useState<About[]>([]); // SB-3004
  const [busy, setBusy] = useState(false);
  const [spaces, setSpaces] = useState<Space[]>([]);
  const [space, setSpace] = useState(""); // "" is the whole company
  const [branch, setBranch] = useState(false); // this department and its teams (SB-3903)
  const [tab, setTab] = useState<Tab>(tabs.find((x) => x.id === first)?.id ?? "answer");
  const [searchFor, setSearchFor] = useState<{ text: string; n: number } | undefined>(() =>
    first === "documents" && fromBar?.trim() ? { text: fromBar.trim().slice(0, 4000), n: 1 } : undefined,
  );
  const [ownerFor, setOwnerFor] = useState<{ email: string; n: number }>(); // SB-3306
  const [suggested, setSuggested] = useState<string[]>([]); // SB-3307
  const [shared, setShared] = useState<SharedQuestion>();
  const [sharedError, setSharedError] = useState<string>();
  const inputId = useId();
  const spaceId = useId();
  const tabRefs = useRef<(HTMLButtonElement | null)[]>([]);
  const base = useId();
  const nextTurn = useRef(0);
  const [showTokens, setShowTokens] = useState(false);
  const tokensButton = useRef<HTMLButtonElement>(null);
  const [recent, setRecent] = useState<string[]>(() => recentQuestions());
  function forget() {
    forgetQuestions();
    setRecent([]);
  }
  const askBox = useRef<HTMLTextAreaElement>(null);
  const [short, setShort] = useState<boolean>(() => shortAnswers());
  const [verified, setVerified] = useState<boolean>(() => verifiedOnly("ask"));
  // One file for the next ask only (SB-3802): sent with it and then let go, never kept.
  const [file, setFile] = useState<AskFile>();
  const [fileError, setFileError] = useState<string>();
  const fileId = useId();
  const fileInput = useRef<HTMLInputElement>(null);
  async function pick(f: File | undefined) {
    setFileError(undefined);
    setFile(undefined);
    if (!f) return;
    const a = await fileToAsk(f).catch(() => null);
    if (a === null) {
      setFileError(t("{name} couldn't be read.", { name: f.name }));
      return;
    }
    if (!a) {
      setFileError(t("{name} is larger than 2 MB.", { name: f.name }));
      if (fileInput.current) fileInput.current.value = "";
      return;
    }
    setFile(a);
  }
  function dropFile() {
    setFile(undefined);
    setFileError(undefined);
    if (fileInput.current) fileInput.current.value = "";
  }

  // Keyboard first (SB-2403): "/" goes to the ask box and Esc closes the API tokens panel,
  // never while the person is typing in a field.
  useEffect(() => {
    function onKey(e: globalThis.KeyboardEvent) {
      const el = e.target as HTMLElement | null;
      const typing = !!el && (el.isContentEditable || ["INPUT", "TEXTAREA", "SELECT"].includes(el.tagName));
      if (e.key === "/" && !typing && !e.ctrlKey && !e.metaKey && !e.altKey) {
        e.preventDefault();
        setTab("answer");
        askBox.current?.focus();
      } else if (e.key === "Escape" && showTokens) {
        setShowTokens(false);
        tokensButton.current?.focus();
      }
    }
    document.addEventListener("keydown", onKey);
    return () => document.removeEventListener("keydown", onKey);
  }, [showTokens]);

  useEffect(() => {
    let live = true;
    setSuggested([]);
    if (space) suggestedQuestions(space, fetcher).then((q) => live && setSuggested(q));
    return () => {
      live = false;
    };
  }, [space, fetcher]);

  useEffect(() => {
    let live = true;
    listSpaces(fetcher).then((s) => live && setSpaces(s));
    return () => {
      live = false;
    };
  }, [fetcher]);

  useEffect(() => {
    if (!share) return;
    openShare(share, fetcher).then(setShared, (e) =>
      setSharedError(e instanceof AskError ? errorText(e.error).message : t("That shared question can't be opened.")),
    );
  }, [share, fetcher]);

  // A saved or shared question asks in its space only if the asker is a member of it now;
  // otherwise company-wide (the server decides access either way).
  function spaceFor(recordSpace: string): string {
    const name = spaceName(recordSpace);
    return spaces.some((s) => s.name === name) ? name : "";
  }

  // remember is false for a shared question asked from its link (SB-2405).
  async function run(q: string, inSpace: string, conv: string | undefined, remember = true) {
    setBusy(true);
    if (remember) setRecent(rememberQuestion(q));
    const id = nextTurn.current++;
    const settle = (x: Turn) => setTurns((ts) => ts.map((y) => (y.id === id ? x : y)));
    setTurns((ts) => [...ts, { id, question: q, pending: true }]);
    // Sources first (SB-2304): search needs no model, so the documents it allows show while
    // the answer is written. A failed search changes nothing; a late one is ignored.
    search(q, fetcher, inSpace || undefined).then(
      (s) => setTurns((ts) => ts.map((y) => (y.id === id && y.pending ? { ...y, sources: s.results } : y))),
      () => undefined,
    );
    try {
      const a = await ask(q, conv, fetcher, inSpace || undefined, lang(), about.map((d) => d.key), short, verified, file);
      setConversation(a.conversation);
      dropFile();
      settle({ id, question: q, answer: a });
      setQuestion("");
    } catch (err) {
      const error = err instanceof AskError ? err.error : { code: "EUNAVAIL", message: t("Something went wrong."), trace: "", remedy: t("Try again.") };
      if (error.code === "EAUTH") forget(); // signed out: this browser forgets their questions
      settle({ id, question: q, error });
    } finally {
      setBusy(false);
    }
  }

  async function submit(e: FormEvent) {
    e.preventDefault();
    const q = question.trim();
    if (!q || busy) return;
    await run(q, branch && space ? space + BRANCH : space, conversation);
  }

  // Asking a saved or shared question again starts a new conversation, live.
  async function askAgain(q: string, recordSpace: string, remember = true) {
    const sp = spaceFor(recordSpace);
    setSpace(sp);
    setBranch(false);
    setConversation(undefined);
    setTurns([]);
    setTab("answer");
    await run(q, sp, undefined, remember);
  }

  // No dead ends (SB-2406): the same question, company-wide, in a new conversation.
  async function askWholeCompany(q: string) {
    setSpace("");
    setConversation(undefined);
    setTurns([]);
    await run(q, "", undefined);
  }

  // From Documents: the text goes into the ask box, to be asked when the person chooses.
  function askInstead(q: string) {
    setQuestion(q);
    setTab("answer");
    setTimeout(() => askBox.current?.focus(), 0);
  }

  // Ask about this document (SB-3004): the next questions search only the chosen ones.
  function addAbout(d: About) {
    setAbout((xs) => (xs.some((x) => x.key === d.key) || xs.length >= maxAbout ? xs : [...xs, d]));
    setTab("answer");
    setTimeout(() => askBox.current?.focus(), 0);
  }

  function theirDocuments(email: string) {
    setOwnerFor((o) => ({ email, n: (o?.n ?? 0) + 1 }));
    setTab("documents");
  }

  function searchInstead(q: string) {
    setSearchFor((s) => ({ text: q, n: (s?.n ?? 0) + 1 }));
    setTab("documents");
  }

  function onTabKey(e: KeyboardEvent, i: number) {
    const step = e.key === "ArrowRight" ? 1 : e.key === "ArrowLeft" ? -1 : 0;
    if (!step) return;
    e.preventDefault();
    const next = (i + step + tabs.length) % tabs.length;
    setTab(tabs[next]!.id);
    tabRefs.current[next]?.focus();
  }

  return (
    <main>
      <a className="skip" href="#conversation" onClick={() => setTab("answer")}>
        {t("Skip to the conversation")}
      </a>
      <h1>{"Second Brain"}</h1>
      <NoticeBanner fetcher={fetcher} />
      {(shared || sharedError) && (
        <section className="shared" aria-label={t("Shared question")}>
          {shared ? (
            <>
              <p>{t("A colleague shared this question. Asking it shows only what you can open.")}</p>
              <p className="question">{shared.question}</p>
              <button type="button" disabled={busy} onClick={() => askAgain(shared.question, shared.space, false)}>
                {t("Ask this question")}
              </button>
            </>
          ) : (
            <p className="error" role="alert">
              {sharedError}
            </p>
          )}
        </section>
      )}
      <div role="tablist" aria-label={t("Views")} className="tabs">
        {tabs.map((x, i) => (
          <button
            key={x.id}
            ref={(el) => {
              tabRefs.current[i] = el;
            }}
            type="button"
            role="tab"
            id={`${base}-tab-${x.id}`}
            aria-selected={tab === x.id}
            aria-controls={`${base}-panel-${x.id}`}
            tabIndex={tab === x.id ? 0 : -1}
            onClick={() => setTab(x.id)}
            onKeyDown={(e) => onTabKey(e, i)}
          >
            {t(x.name)}
          </button>
        ))}
      </div>
      <div role="tabpanel" id={`${base}-panel-answer`} aria-labelledby={`${base}-tab-answer`} hidden={tab !== "answer"}>
        {space && turns.length === 0 && <StartHere space={space} fetcher={fetcher} />}
        {turns.length === 0 && <WhatChanged space={space || undefined} fetcher={fetcher} />}
        <section id="conversation" tabIndex={-1} aria-live="polite" aria-busy={busy} aria-label={t("Conversation")} className="turns">
          {turns.map((x) => (
            <div className="turn" key={x.id}>
              <h2 className="question">{x.question}</h2>
              {x.pending && <SourcesFirst sources={x.sources} />}
              {x.answer && <AnswerView answer={x.answer} fetcher={fetcher} onAbout={addAbout} onOwner={theirDocuments} question={x.question} space={space || undefined} />}
              {x.answer && <AnswerTools question={x.question} answer={x.answer} />}
              {x.answer && <FeedbackBar answer={x.answer} fetcher={fetcher} question={x.question} space={space || undefined} />}
              {x.answer && <TurnTools question={x.question} fetcher={fetcher} space={space || undefined} />}
              {x.answer && space && (x.answer.notice === "no_sources" || x.answer.notice === "no_access") && (
                <button type="button" disabled={busy} onClick={() => askWholeCompany(x.question)}>
                  {t("Ask the whole company")}
                </button>
              )}
              {x.answer && <AskStewards question={x.question} answer={x.answer} space={space || undefined} fetcher={fetcher} />}
              {x.answer && (x.answer.earlier_dropped ?? 0) > 0 && x.id === turns[turns.length - 1]?.id && (
                <p className="meta">
                  {tn(
                    x.answer.earlier_dropped!,
                    "This conversation is long: the earliest question wasn't used for this answer.",
                    "This conversation is long: the earliest {n} questions weren't used for this answer.",
                  )}{" "}
                  <button
                    type="button"
                    onClick={() => {
                      setConversation(undefined);
                      setTurns([]);
                      setAbout([]);
                    }}
                  >
                    {t("Start fresh")}
                  </button>
                </p>
              )}
              {x.error && x.error.code === "EAUTH" && (
                <div className="error" role="alert">
                  <p>{tj("You're signed out. {link}, then ask again.", { link: <a href="/login">{t("Sign in")}</a> })}</p>
                </div>
              )}
              {x.error && x.error.code !== "EAUTH" && <ErrorBox error={x.error} />}
              {x.error && searchable.has(x.error.code) && (
                <button type="button" onClick={() => searchInstead(x.question)}>
                  {t("Search documents instead")}
                </button>
              )}
            </div>
          ))}
        </section>
        <form onSubmit={submit} className="ask">
          {spaces.length > 0 && (
            <>
              <label htmlFor={spaceId}>{t("Ask in")}</label>
              <select
                id={spaceId}
                value={branch && space ? space + BRANCH : space}
                onChange={(e) => {
                  const v = e.target.value;
                  setBranch(v.endsWith(BRANCH));
                  setSpace(v.endsWith(BRANCH) ? v.slice(0, -BRANCH.length) : v);
                  setConversation(undefined); // a new space starts a new conversation
                  setTurns([]);
                }}
              >
                <option value="">{t("Whole company")}</option>
                {spaces.map((s) => [
                  <option key={s.name} value={s.name}>
                    {s.name.charAt(0).toUpperCase() + s.name.slice(1)} ({s.label})
                  </option>,
                  s.teams && (
                    <option key={s.name + BRANCH} value={s.name + BRANCH}>
                      {t("{space} and its teams", { space: s.name.charAt(0).toUpperCase() + s.name.slice(1) })}
                    </option>
                  ),
                ])}
              </select>
            </>
          )}
          {about.length > 0 && (
            <p className="about" role="status">
              {t("Asking about: {titles}", { titles: about.map((d) => d.name).join(", ") })}{" "}
              <button type="button" className="link" onClick={() => setAbout([])}>
                {t("Clear")}
              </button>
            </p>
          )}
          <label htmlFor={inputId}>{t("Ask a question")}</label>
          <textarea
            id={inputId}
            ref={askBox}
            value={question}
            maxLength={4000}
            rows={3}
            aria-describedby={`${inputId}-keys`}
            onChange={(e) => setQuestion(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Enter" && (e.ctrlKey || e.metaKey)) {
                e.preventDefault();
                e.currentTarget.form?.requestSubmit();
              }
            }}
          />
          {space && question === "" && suggested.length > 0 && (
            <div className="suggested" role="group" aria-label={t("Suggested questions")}>
              {suggested.map((x) => (
                <button
                  key={x}
                  type="button"
                  className="link"
                  onClick={() => {
                    setQuestion(x);
                    askBox.current?.focus();
                  }}
                >
                  {x}
                </button>
              ))}
            </div>
          )}
          <p className="keys" id={`${inputId}-keys`}>
            {t("Keyboard: / goes to the ask box, Ctrl+Enter (Cmd+Enter on a Mac) asks, Esc closes a panel.")}
          </p>
          {recent.length > 0 && (
            <details className="recent">
              <summary>{t("Recent questions")}</summary>
              <ul>
                {recent.map((r) => (
                  <li key={r}>
                    <button
                      type="button"
                      className="link"
                      onClick={() => {
                        setQuestion(r);
                        askBox.current?.focus();
                      }}
                    >
                      {r}
                    </button>
                  </li>
                ))}
              </ul>
              <button type="button" onClick={forget}>
                {t("Clear the list")}
              </button>
            </details>
          )}
          <label className="short">
            <input
              type="checkbox"
              checked={short}
              onChange={(e) => {
                setShort(e.target.checked);
                setShortAnswers(e.target.checked);
              }}
            />{" "}
            {t("Short answer")}
          </label>
          <label className="short">
            <input
              type="checkbox"
              checked={verified}
              onChange={(e) => {
                setVerified(e.target.checked);
                setVerifiedOnly("ask", e.target.checked);
              }}
            />{" "}
            {t("Verified only")}
          </label>
          <span className="attach">
            <label htmlFor={fileId}>{t("Attach a file")}</label>{" "}
            <input id={fileId} ref={fileInput} type="file" onChange={(e) => void pick(e.target.files?.[0])} />
            {file && (
              <>
                {" "}
                <span>{t("Asking about {name}", { name: file.name })}</span>{" "}
                <button type="button" onClick={dropFile}>
                  {t("Remove the file")}
                </button>
              </>
            )}
            {fileError && <span role="alert">{fileError}</span>}
          </span>
          <button type="submit" disabled={busy || question.trim() === ""}>
            {busy ? t("Asking…") : t("Ask")}
          </button>
          <DownloadConversation turns={turns} />
          {conversation && (
            <button
              type="button"
              onClick={() => {
                setConversation(undefined);
                setTurns([]);
                setAbout([]);
              }}
            >
              {t("New conversation")}
            </button>
          )}
        </form>
        <DraftPanel fetcher={fetcher} />
      </div>
      <div role="tabpanel" id={`${base}-panel-documents`} aria-labelledby={`${base}-tab-documents`} hidden={tab !== "documents"}>
        <DocumentsView fetcher={fetcher} space={(branch && space ? space + BRANCH : space) || undefined} query={searchFor} onAsk={askInstead} onAbout={addAbout} owner={ownerFor} />
      </div>
      <div role="tabpanel" id={`${base}-panel-saved`} aria-labelledby={`${base}-tab-saved`} hidden={tab !== "saved"}>
        {tab === "saved" && <SavedView fetcher={fetcher} onAsk={askAgain} />}
      </div>
      <div role="tabpanel" id={`${base}-panel-questions`} aria-labelledby={`${base}-tab-questions`} hidden={tab !== "questions"}>
        {tab === "questions" && <QuestionsView fetcher={fetcher} />}
      </div>
      <div role="tabpanel" id={`${base}-panel-handover`} aria-labelledby={`${base}-tab-handover`} hidden={tab !== "handover"}>
        {tab === "handover" && <HandoverView fetcher={fetcher} />}
      </div>
      <button type="button" className="link" ref={tokensButton} aria-expanded={showTokens} onClick={() => setShowTokens((x) => !x)}>
        {t("API tokens")}
      </button>
      {showTokens && <ApiTokens fetcher={fetcher} />}
      <MySessions fetcher={fetcher} onSignedOut={forget} />
      <SignOutEverywhere fetcher={fetcher} onSignedOut={forget} />
    </main>
  );
}
