import { useCallback, useEffect, useId, useState } from "react";
import type { FormEvent } from "react";
import {
  AskError,
  actionDone,
  citeName,
  listResearch,
  myActions,
  openResearch,
  planResearch,
  prepareBrief,
  runResearch,
} from "./api";
import type {
  ActionItem,
  ApiError,
  Fetch,
  Research,
  ResearchItem,
} from "./api";
import { ClaimView } from "./Answer";
import { ErrorBox } from "./ErrorBox";
import { AccountBriefForm, AccountFactsView, HowThisWasMade } from "./Further";
import { MarkLine } from "./Marked";
import { msg, t, tn } from "./i18n";

// The legwork (sb-ask(3), Research reports, Meeting prep and Action items; SB-4201 to
// SB-4203): every report and brief is the person's own, and is checked again as them each
// time it is shown (LAW-26).

const states: Record<string, string> = {
  planned: msg("planned"),
  queued: msg("queued"),
  running: msg("running"),
  done: msg("done"),
  stopped: msg("stopped"),
  failed: msg("failed"),
  suggested: msg("suggested"),
  applied: msg("applied"),
};

const kindName = (k: string) =>
  k === "prep"
    ? t("Brief")
    : k === "fix"
      ? t("Suggested edits")
      : k === "account"
        ? t("Account brief")
        : t("Report");

const unavailable = (): ApiError => ({
  code: "EUNAVAIL",
  message: t("Something went wrong."),
  trace: "",
  remedy: t("Try again."),
});

export function LegworkView({
  fetcher,
  space,
}: {
  fetcher?: Fetch;
  space?: string;
}) {
  const [list, setList] = useState<ResearchItem[]>([]);
  const [shown, setShown] = useState<Research>();
  const [error, setError] = useState<ApiError>();
  const load = useCallback(() => {
    listResearch(fetcher).then(setList, () => setList([]));
  }, [fetcher]);
  useEffect(load, [load]);
  const fail = (err: unknown) =>
    setError(err instanceof AskError ? err.error : unavailable());
  const show = async (id: string) => {
    setError(undefined);
    try {
      setShown(await openResearch(id, fetcher));
    } catch (err) {
      fail(err);
    }
  };
  return (
    <section aria-label={t("Legwork")} className="legwork">
      <h2>{t("Legwork")}</h2>
      <ResearchForm
        fetcher={fetcher}
        space={space}
        onPlanned={(r) => (setShown(r), load())}
        onError={fail}
      />
      <PrepareForm
        fetcher={fetcher}
        space={space}
        onMade={(r) => (setShown(r), load())}
        onError={fail}
      />
      <AccountBriefForm
        fetcher={fetcher}
        space={space}
        onMade={(r) => (setShown(r), load())}
      />
      {error && <ErrorBox error={error} />}
      {shown && (
        <ResearchShown
          r={shown}
          fetcher={fetcher}
          space={space}
          onChange={(r) => (setShown(r), load())}
          onError={fail}
        />
      )}
      {list.length > 0 && (
        <section aria-label={t("Your reports and briefs")}>
          <h3>{t("Your reports and briefs")}</h3>
          <ul className="saved-list">
            {list.map((r) => (
              <li key={r.id}>
                <button type="button" onClick={() => show(r.id)}>
                  {r.question}
                </button>{" "}
                <span className="meta">
                  {kindName(r.kind)} · {t(states[r.state] ?? r.state)}
                </span>
              </li>
            ))}
          </ul>
        </section>
      )}
      <MyActions fetcher={fetcher} />
    </section>
  );
}

function ResearchForm({
  fetcher,
  space,
  onPlanned,
  onError,
}: {
  fetcher?: Fetch;
  space?: string;
  onPlanned: (r: Research) => void;
  onError: (e: unknown) => void;
}) {
  const id = useId();
  const [q, setQ] = useState("");
  const [busy, setBusy] = useState(false);
  const submit = async (e: FormEvent) => {
    e.preventDefault();
    if (!q.trim() || busy) return;
    setBusy(true);
    try {
      onPlanned(await planResearch(q.trim(), fetcher, space));
      setQ("");
    } catch (err) {
      onError(err);
    } finally {
      setBusy(false);
    }
  };
  return (
    <form onSubmit={submit} aria-label={t("Research")}>
      <h3>{t("Research")}</h3>
      <label htmlFor={id}>
        {t("A question that needs more than one search")}
      </label>
      <input
        id={id}
        value={q}
        maxLength={500}
        onChange={(e) => setQ(e.target.value)}
      />
      <button type="submit" disabled={busy || q.trim() === ""}>
        {busy ? t("Planning…") : t("Plan")}
      </button>
      <p className="meta">
        {t("You see the plan first. Nothing runs until you press Run.")}
      </p>
    </form>
  );
}

function PrepareForm({
  fetcher,
  space,
  onMade,
  onError,
}: {
  fetcher?: Fetch;
  space?: string;
  onMade: (r: Research) => void;
  onError: (e: unknown) => void;
}) {
  const base = useId();
  const [title, setTitle] = useState("");
  const [agenda, setAgenda] = useState("");
  const [attendees, setAttendees] = useState("");
  const [busy, setBusy] = useState(false);
  const submit = async (e: FormEvent) => {
    e.preventDefault();
    if (!agenda.trim() || busy) return;
    setBusy(true);
    try {
      const who = attendees.split(/[\s,;]+/).filter(Boolean);
      onMade(await prepareBrief(agenda, title.trim(), who, fetcher, space));
    } catch (err) {
      onError(err);
    } finally {
      setBusy(false);
    }
  };
  return (
    <form onSubmit={submit} aria-label={t("Prepare a meeting")}>
      <h3>{t("Prepare a meeting")}</h3>
      <label htmlFor={`${base}-title`}>{t("Meeting")}</label>
      <input
        id={`${base}-title`}
        value={title}
        maxLength={200}
        onChange={(e) => setTitle(e.target.value)}
      />
      <label htmlFor={`${base}-agenda`}>{t("Agenda, one item a line")}</label>
      <textarea
        id={`${base}-agenda`}
        value={agenda}
        maxLength={4000}
        rows={5}
        onChange={(e) => setAgenda(e.target.value)}
      />
      <label htmlFor={`${base}-who`}>{t("Attendees' emails")}</label>
      <input
        id={`${base}-who`}
        value={attendees}
        maxLength={2000}
        onChange={(e) => setAttendees(e.target.value)}
      />
      <button type="submit" disabled={busy || agenda.trim() === ""}>
        {busy ? t("Preparing…") : t("Prepare")}
      </button>
    </form>
  );
}

function ResearchShown({
  r,
  fetcher,
  space,
  onChange,
  onError,
}: {
  r: Research;
  fetcher?: Fetch;
  space?: string;
  onChange: (r: Research) => void;
  onError: (e: unknown) => void;
}) {
  const run = async () => {
    try {
      onChange(await runResearch(r.id, fetcher, space));
    } catch (err) {
      onError(err);
    }
  };
  return (
    <article className="answer" aria-label={r.question} aria-live="polite">
      <h3>{r.question}</h3>
      <p className="meta">
        {kindName(r.kind)} · {t(states[r.state] ?? r.state)}
      </p>
      {r.state === "planned" && (
        <>
          <ol>
            {r.steps.map((s, i) => (
              <li key={i}>{s}</li>
            ))}
          </ol>
          <button type="button" onClick={run}>
            {t("Run")}
          </button>
        </>
      )}
      {r.account && <AccountFactsView account={r.account} />}
      {r.state === "queued" && (
        <p>{t("Queued. You're told when it's ready.")}</p>
      )}
      {r.stopped && (
        <p className="notice">
          {t("It stopped early: {why}", { why: r.stopped })}
        </p>
      )}
      {r.sections.map((s, i) => (
        <section key={i} aria-label={s.question}>
          <h4>{s.question}</h4>
          <ul className="claims">
            {s.claims.map((c, j) => (
              <ClaimView key={j} claim={c} trace={s.trace} fetcher={fetcher} />
            ))}
          </ul>
          {s.decisions && s.decisions.length > 0 && (
            <p className="meta">
              {t("Decisions behind it")}:{" "}
              {s.decisions.map((d) => citeName(d)).join(", ")}
            </p>
          )}
          {s.questions !== undefined && s.questions > 0 && (
            <p className="meta">
              {tn(
                s.questions,
                "{n} open question in this space",
                "{n} open questions in this space",
              )}
            </p>
          )}
        </section>
      ))}
      {r.attendees && r.attendees.length > 0 && (
        <section aria-label={t("Attendees")}>
          <h4>{t("Attendees")}</h4>
          <ul>
            {r.attendees.map((a) => (
              <li key={a.email}>
                {a.email}
                {!a.known && ` (${t("not someone we know")})`}
                {a.owns &&
                  a.owns.length > 0 &&
                  `: ${t("owns")} ${a.owns.map((d) => citeName(d)).join(", ")}`}
              </li>
            ))}
          </ul>
        </section>
      )}
      {r.dropped > 0 && (
        <p className="meta">
          {tn(
            r.dropped,
            "{n} line was left out because you can't open its source now.",
            "{n} lines were left out because you can't open their sources now.",
          )}
        </p>
      )}
      {r.sections.length > 0 && r.mark && (
        <MarkLine agent={r.mark.agent} version={r.mark.version} />
      )}
      {r.sections.length > 0 && (
        <HowThisWasMade report={r.id} space={space} fetcher={fetcher} />
      )}
    </article>
  );
}

function MyActions({ fetcher }: { fetcher?: Fetch }) {
  const [items, setItems] = useState<ActionItem[]>([]);
  useEffect(() => {
    myActions(fetcher).then(setItems, () => setItems([]));
  }, [fetcher]);
  const done = async (id: string) => {
    try {
      const a = await actionDone(id, fetcher);
      setItems((xs) => xs.map((x) => (x.id === id ? a : x)));
    } catch {
      // stays as it was; the next load shows the truth
    }
  };
  if (items.length === 0) return null;
  return (
    <section aria-label={t("Your action items")}>
      <h3>{t("Your action items")}</h3>
      <ul>
        {items.map((a) => (
          <li key={a.id} className={a.overdue ? "overdue" : undefined}>
            {a.text}
            {a.due && (
              <span className="meta">
                {" "}
                · {t("due {date}", { date: a.due })}
              </span>
            )}
            {a.overdue && <span className="meta"> · {t("Overdue")}</span>}{" "}
            {a.done_at ? (
              <span className="meta">{t("Done")}</span>
            ) : (
              <button type="button" onClick={() => done(a.id)}>
                {t("Mark done")}
              </button>
            )}
          </li>
        ))}
      </ul>
    </section>
  );
}
