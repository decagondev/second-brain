import { useEffect, useId, useRef, useState } from "react";
import type { FormEvent, KeyboardEvent } from "react";
import { ask, AskError, listSpaces, openShare, spaceName } from "./api";
import type { ApiError, AskResponse, Fetch, SharedQuestion, Space } from "./api";
import { AnswerView } from "./Answer";
import { FeedbackBar } from "./Feedback";
import { DraftPanel } from "./Draft";
import { DocumentsView } from "./Documents";
import { SavedView, TurnTools } from "./Saved";
import { AskStewards, QuestionsView } from "./Questions";
import { StartHere, WhatChanged } from "./Start";
import { SignOutEverywhere } from "./Session";
import { HandoverView } from "./Handover";

interface Turn {
  question: string;
  answer?: AskResponse;
  error?: ApiError;
}

type Tab = "answer" | "documents" | "saved" | "questions" | "handover";
const tabs: { id: Tab; name: string }[] = [
  { id: "answer", name: "Answer" },
  { id: "documents", name: "Documents" },
  { id: "saved", name: "Saved" },
  { id: "questions", name: "Questions" },
  { id: "handover", name: "Handover" },
];

// Errors after which the same text can still be searched: search needs no model (SB-1701).
const searchable = new Set(["EUNAVAIL", "EBUDGET"]);

// tab opens the app on a tab, from a notification's link (?tab=questions; SB-2102).
export function App({ fetcher, share, tab: first }: { fetcher?: Fetch; share?: string; tab?: string }) {
  const [question, setQuestion] = useState("");
  const [turns, setTurns] = useState<Turn[]>([]);
  const [conversation, setConversation] = useState<string>();
  const [busy, setBusy] = useState(false);
  const [spaces, setSpaces] = useState<Space[]>([]);
  const [space, setSpace] = useState(""); // "" is the whole company
  const [tab, setTab] = useState<Tab>(tabs.find((t) => t.id === first)?.id ?? "answer");
  const [searchFor, setSearchFor] = useState<{ text: string; n: number }>();
  const [shared, setShared] = useState<SharedQuestion>();
  const [sharedError, setSharedError] = useState<string>();
  const inputId = useId();
  const spaceId = useId();
  const tabRefs = useRef<(HTMLButtonElement | null)[]>([]);
  const base = useId();

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
      setSharedError(e instanceof AskError ? e.error.message : "That shared question can't be opened."),
    );
  }, [share, fetcher]);

  // A saved or shared question asks in its space only if the asker is a member of it now;
  // otherwise company-wide (the server decides access either way).
  function spaceFor(recordSpace: string): string {
    const name = spaceName(recordSpace);
    return spaces.some((s) => s.name === name) ? name : "";
  }

  async function run(q: string, inSpace: string, conv: string | undefined) {
    setBusy(true);
    try {
      const a = await ask(q, conv, fetcher, inSpace || undefined);
      setConversation(a.conversation);
      setTurns((t) => [...t, { question: q, answer: a }]);
      setQuestion("");
    } catch (err) {
      const error = err instanceof AskError ? err.error : { code: "EUNAVAIL", message: "Something went wrong.", trace: "", remedy: "Try again." };
      setTurns((t) => [...t, { question: q, error }]);
    } finally {
      setBusy(false);
    }
  }

  async function submit(e: FormEvent) {
    e.preventDefault();
    const q = question.trim();
    if (!q || busy) return;
    await run(q, space, conversation);
  }

  // Asking a saved or shared question again starts a new conversation, live.
  async function askAgain(q: string, recordSpace: string) {
    const sp = spaceFor(recordSpace);
    setSpace(sp);
    setConversation(undefined);
    setTurns([]);
    setTab("answer");
    await run(q, sp, undefined);
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
      <h1>Second Brain</h1>
      {(shared || sharedError) && (
        <section className="shared" aria-label="Shared question">
          {shared ? (
            <>
              <p>A colleague shared this question. Asking it shows only what you can open.</p>
              <p className="question">{shared.question}</p>
              <button type="button" disabled={busy} onClick={() => askAgain(shared.question, shared.space)}>
                Ask this question
              </button>
            </>
          ) : (
            <p className="error" role="alert">
              {sharedError}
            </p>
          )}
        </section>
      )}
      <div role="tablist" aria-label="Views" className="tabs">
        {tabs.map((t, i) => (
          <button
            key={t.id}
            ref={(el) => {
              tabRefs.current[i] = el;
            }}
            type="button"
            role="tab"
            id={`${base}-tab-${t.id}`}
            aria-selected={tab === t.id}
            aria-controls={`${base}-panel-${t.id}`}
            tabIndex={tab === t.id ? 0 : -1}
            onClick={() => setTab(t.id)}
            onKeyDown={(e) => onTabKey(e, i)}
          >
            {t.name}
          </button>
        ))}
      </div>
      <div role="tabpanel" id={`${base}-panel-answer`} aria-labelledby={`${base}-tab-answer`} hidden={tab !== "answer"}>
        {space && turns.length === 0 && <StartHere space={space} fetcher={fetcher} />}
        {turns.length === 0 && <WhatChanged space={space || undefined} fetcher={fetcher} />}
        <section aria-live="polite" aria-busy={busy} aria-label="Conversation" className="turns">
          {turns.map((t, i) => (
            <div className="turn" key={i}>
              <h2 className="question">{t.question}</h2>
              {t.answer && <AnswerView answer={t.answer} fetcher={fetcher} />}
              {t.answer && <FeedbackBar answer={t.answer} fetcher={fetcher} />}
              {t.answer && <TurnTools question={t.question} fetcher={fetcher} space={space || undefined} />}
              {t.answer && <AskStewards question={t.question} answer={t.answer} space={space || undefined} fetcher={fetcher} />}
              {t.error && (
                <div className="error" role="alert">
                  {t.error.code === "EAUTH" ? (
                    <p>
                      You're signed out. <a href="/login">Sign in</a>, then ask again.
                    </p>
                  ) : (
                    <>
                      <p>{t.error.message}</p>
                      {t.error.remedy && t.error.remedy.toLowerCase() !== t.error.message.toLowerCase() && <p>{t.error.remedy}</p>}
                    </>
                  )}
                  {t.error.trace && <p>Quote trace {t.error.trace} to support.</p>}
                  {searchable.has(t.error.code) && (
                    <button type="button" onClick={() => searchInstead(t.question)}>
                      Search documents instead
                    </button>
                  )}
                </div>
              )}
            </div>
          ))}
        </section>
        <form onSubmit={submit} className="ask">
          {spaces.length > 0 && (
            <>
              <label htmlFor={spaceId}>Ask in</label>
              <select
                id={spaceId}
                value={space}
                onChange={(e) => {
                  setSpace(e.target.value);
                  setConversation(undefined); // a new space starts a new conversation
                  setTurns([]);
                }}
              >
                <option value="">Whole company</option>
                {spaces.map((s) => (
                  <option key={s.name} value={s.name}>
                    {s.name.charAt(0).toUpperCase() + s.name.slice(1)} ({s.label})
                  </option>
                ))}
              </select>
            </>
          )}
          <label htmlFor={inputId}>Ask a question</label>
          <textarea id={inputId} value={question} maxLength={4000} rows={3} onChange={(e) => setQuestion(e.target.value)} />
          <button type="submit" disabled={busy || question.trim() === ""}>
            {busy ? "Asking…" : "Ask"}
          </button>
          {conversation && (
            <button
              type="button"
              onClick={() => {
                setConversation(undefined);
                setTurns([]);
              }}
            >
              New conversation
            </button>
          )}
        </form>
        <DraftPanel fetcher={fetcher} />
      </div>
      <div role="tabpanel" id={`${base}-panel-documents`} aria-labelledby={`${base}-tab-documents`} hidden={tab !== "documents"}>
        <DocumentsView fetcher={fetcher} space={space || undefined} query={searchFor} />
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
      <SignOutEverywhere fetcher={fetcher} />
    </main>
  );
}
