import { useEffect, useId, useState } from "react";
import type { FormEvent } from "react";
import type { DefinedTerm, Fetch } from "./api";
import { ReviewList, StewardQuestions } from "./Questions";
import { SpaceAgents } from "./Agents";
import { StewardPolicies } from "./Company";
import { t, tj } from "./i18n";

export interface ReviewItem {
  user: string;
  via: string[];
  last_use: string;
  decision?: string;
  by?: string;
  reason?: string;
  unused?: boolean; // an agent review's (SB-4705)
  tools?: string[];
}

export interface ReviewRequest {
  kind: string;
  user: string;
  group: string;
  proposal?: string;
}

export interface Review {
  id: string;
  space: string;
  quarter: string;
  due: string;
  state: string;
  overdue?: boolean;
  items: ReviewItem[];
  requests: ReviewRequest[] | null;
  kind?: string; // "agents" for an agent review (SB-4705)
  propose_off?: string[];
}

interface Proposed {
  id: string;
  hash: string;
  diff: string[];
  plan: string;
}

// call sends one request and returns its JSON, or throws the cell's own message and the
// manifest errors it lists, if any.
export async function call(
  f: Fetch,
  method: string,
  url: string,
  body?: unknown,
): Promise<Record<string, unknown>> {
  let res: Response;
  try {
    res = await f(url, {
      method,
      credentials: "same-origin",
      headers: { "Content-Type": "application/json" },
      ...(body ? { body: JSON.stringify(body) } : {}),
    });
  } catch {
    throw new Error(t("The cell can't be reached right now."));
  }
  const data = (await res.json().catch(() => null)) as Record<
    string,
    unknown
  > | null;
  if (!res.ok) {
    const e = (data?.error ?? {}) as {
      message?: string;
      errors?: { message?: string; field?: string }[];
    };
    const why = (e.errors ?? [])
      .map((x) => `${x.field ? x.field + ": " : ""}${x.message ?? ""}`)
      .filter(Boolean);
    throw new Error([e.message ?? t("That didn't work."), ...why].join(" "));
  }
  return data ?? {};
}

// The steward's screen for one space (sb-spaces(7), Stewards in the web UI; SB-1509): a
// proposal of the space's new definition, which only an operator can approve, and the
// space's access reviews, decided one person at a time.
export function StewardView({
  space,
  fetcher = fetch,
}: {
  space: string;
  fetcher?: Fetch;
}) {
  const [definition, setDefinition] = useState(
    '{\n  "members": ["group:..."],\n  "stewards": ["group:..."]\n}',
  );
  const [proposed, setProposed] = useState<Proposed>();
  const [reviews, setReviews] = useState<Review[]>([]);
  const [status, setStatus] = useState<string>();
  const [removing, setRemoving] = useState<{ review: string; user: string }>();
  const [reason, setReason] = useState("");
  const [prefill, setPrefill] = useState<DefinedTerm>();
  const ids = { def: useId(), reason: useId() };
  const base = `/v1/spaces/${encodeURIComponent(space)}`;

  async function load() {
    try {
      const r = await call(fetcher, "GET", `${base}/reviews`);
      setReviews((r.items as Review[]) ?? []);
    } catch (e) {
      setStatus((e as Error).message);
    }
  }
  useEffect(() => {
    void load();
  }, [space]);

  async function propose() {
    let parsed: unknown;
    try {
      parsed = JSON.parse(definition);
    } catch {
      setStatus(
        t("The definition must be JSON, in the fields of {manual}.", {
          manual: "sb-space(5)",
        }),
      );
      return;
    }
    try {
      const p = await call(fetcher, "POST", `${base}/proposals`, {
        space: parsed,
      });
      setProposed({
        id: String(p.id),
        hash: String(p.hash),
        diff: (p.diff as string[]) ?? [],
        plan: String(p.plan ?? ""),
      });
      setStatus(
        t(
          "Proposed. Nothing changes until an operator approves this hash and the manifest is applied.",
        ),
      );
    } catch (e) {
      setProposed(undefined);
      setStatus((e as Error).message);
    }
  }

  async function decide(
    review: string,
    user: string,
    decision: "certify" | "remove",
    why = "",
  ) {
    if (decision === "remove" && !why.trim()) {
      setRemoving({ review, user });
      setStatus(t("A removal needs a reason; it goes into the audit log."));
      return;
    }
    try {
      const next = (await call(
        fetcher,
        "POST",
        `${base}/reviews/${encodeURIComponent(review)}/decide`,
        { user, decision, reason: why.trim() },
      )) as unknown as Review;
      setReviews((rs) => rs.map((r) => (r.id === next.id ? next : r)));
      setRemoving(undefined);
      setReason("");
      setStatus(
        next.state === "complete"
          ? t("The review is complete. Its change requests are listed below.")
          : t(
              decision === "certify"
                ? "Recorded certify for {user}."
                : "Recorded remove for {user}.",
              { user },
            ),
      );
    } catch (e) {
      setStatus((e as Error).message);
    }
  }

  return (
    <main>
      <h1>{t("Space {space}", { space })}</h1>
      <section aria-label={t("Propose a change")}>
        <h2>{t("Propose a change")}</h2>
        <label htmlFor={ids.def}>
          {t("The space's new definition (JSON)")}
        </label>
        <textarea
          id={ids.def}
          rows={8}
          value={definition}
          onChange={(e) => setDefinition(e.target.value)}
        />
        <button type="button" onClick={propose}>
          {t("Propose")}
        </button>
        {proposed && (
          <div>
            <h3>{t("Proposal {id}", { id: proposed.id })}</h3>
            <p>
              {tj("An operator approves hash {hash}.", {
                hash: <code>{proposed.hash}</code>,
              })}
            </p>
            <pre className="payload">{proposed.diff.join("\n")}</pre>
            <pre className="payload">{proposed.plan}</pre>
          </div>
        )}
      </section>
      <SuggestTerm base={base} fetcher={fetcher} prefill={prefill} />
      <SuggestedEditor base={base} fetcher={fetcher} />
      <SpaceAgents base={base} fetcher={fetcher} />
      <StewardPolicies base={base} fetcher={fetcher} />
      <StewardQuestions space={space} fetcher={fetcher} />
      <ReviewList space={space} fetcher={fetcher} onSuggestTerm={setPrefill} />
      <section aria-label={t("Access reviews")}>
        <h2>{t("Access reviews")}</h2>
        {reviews.length === 0 ? (
          <p>{t("No access reviews for this space.")}</p>
        ) : (
          reviews.map((r) => (
            <div key={r.id}>
              <h3>
                {r.kind === "agents" && <>{t("Agents and triggers")} · </>}
                {r.overdue
                  ? t("{quarter}: {state} (overdue), due {due}", {
                      quarter: r.quarter,
                      state: r.state,
                      due: r.due,
                    })
                  : t("{quarter}: {state}, due {due}", {
                      quarter: r.quarter,
                      state: r.state,
                      due: r.due,
                    })}
              </h3>
              {r.kind === "agents" && (r.propose_off?.length ?? 0) > 0 && (
                <>
                  <p>
                    {t(
                      "Unused in the last 90 days, proposed off. Nothing changes until you remove them here and an operator approves the manifest diff.",
                    )}
                  </p>
                  <pre className="payload">{r.propose_off!.join("\n")}</pre>
                </>
              )}
              <table className="diff">
                <caption className="visually-hidden">
                  {r.kind === "agents"
                    ? t("Agents and triggers in {space}", { space: r.space })
                    : t("People with access to {space}", { space: r.space })}
                </caption>
                <thead>
                  <tr>
                    <th scope="col">
                      {r.kind === "agents"
                        ? t("Agent or trigger")
                        : t("Person")}
                    </th>
                    <th scope="col">{t("Via")}</th>
                    <th scope="col">{t("Last use")}</th>
                    <th scope="col">{t("Decision")}</th>
                  </tr>
                </thead>
                <tbody>
                  {r.items.map((it) => (
                    <tr key={it.user}>
                      <th scope="row">{it.user}</th>
                      <td>{it.via.join(", ")}</td>
                      <td>
                        {it.last_use}
                        {it.unused && <> · {t("unused")}</>}
                        {(it.tools?.length ?? 0) > 0 && (
                          <>
                            {" "}
                            ·{" "}
                            {t("tools: {tools}", {
                              tools: it.tools!.join(", "),
                            })}
                          </>
                        )}
                      </td>
                      <td>
                        {it.decision ? (
                          t("{decision} by {by}", {
                            decision: it.decision,
                            by: it.by ?? "?",
                          }) + (it.reason ? ": " + it.reason : "")
                        ) : (
                          <>
                            <button
                              type="button"
                              onClick={() => decide(r.id, it.user, "certify")}
                              aria-label={t("{user} still needs it", {
                                user: it.user,
                              })}
                            >
                              {t("Still needs it")}
                            </button>{" "}
                            <button
                              type="button"
                              onClick={() =>
                                decide(
                                  r.id,
                                  it.user,
                                  "remove",
                                  removing?.user === it.user ? reason : "",
                                )
                              }
                              aria-label={t("Remove {user}", { user: it.user })}
                            >
                              {t("Remove")}
                            </button>
                          </>
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
              {r.requests && r.requests.length > 0 && (
                <ul aria-label={t("Change requests")}>
                  {r.requests.map((q) => (
                    <li key={`${q.user}/${q.group}`}>
                      {r.kind === "agents"
                        ? t("Manifest proposal {proposal}: turn off {item}", {
                            proposal: q.proposal ?? "",
                            item: q.user,
                          })
                        : q.kind === "manifest"
                          ? t(
                              "Manifest proposal {proposal}: take {user} out of {group}",
                              {
                                proposal: q.proposal ?? "",
                                user: q.user,
                                group: q.group,
                              },
                            )
                          : t("IdP change: take {user} out of {group}", {
                              user: q.user,
                              group: q.group,
                            })}
                    </li>
                  ))}
                </ul>
              )}
            </div>
          ))
        )}
        {removing && (
          <div>
            <label htmlFor={ids.reason}>
              {t("Reason for removing {user} (required)", {
                user: removing.user,
              })}
            </label>
            <input
              id={ids.reason}
              type="text"
              value={reason}
              onChange={(e) => setReason(e.target.value)}
            />
            <button
              type="button"
              onClick={() =>
                decide(removing.review, removing.user, "remove", reason)
              }
            >
              {t("Confirm removal of {user}", { user: removing.user })}
            </button>
          </div>
        )}
      </section>
      {status && <p role="status">{status}</p>}
    </main>
  );
}

// Suggest a term (SB-3107): a steward proposal of one glossary term, approved like any other.
function SuggestTerm({
  base,
  fetcher,
  prefill,
}: {
  base: string;
  fetcher: Fetch;
  prefill?: DefinedTerm;
}) {
  const id = useId();
  const [term, setTerm] = useState("");
  const [meaning, setMeaning] = useState("");
  useEffect(() => {
    if (!prefill) return;
    setTerm(prefill.term); // a suggestion from the documents (SB-3407): only fills the form
    setMeaning(prefill.meaning);
  }, [prefill]);
  const [spellings, setSpellings] = useState("");
  const [status, setStatus] = useState("");
  const list = spellings
    .split(",")
    .map((x) => x.trim())
    .filter((x) => x !== "");
  async function send(e: FormEvent) {
    e.preventDefault();
    try {
      const p = await call(fetcher, "POST", `${base}/proposals/term`, {
        term: term.trim(),
        meaning: meaning.trim(),
        spellings: list,
      });
      setStatus(
        t(
          "Proposal {id} made. Nothing changes until an operator approves it and the manifest is applied.",
          { id: String(p.id) },
        ),
      );
      setTerm("");
      setMeaning("");
      setSpellings("");
    } catch (err) {
      setStatus((err as Error).message);
    }
  }
  return (
    <section aria-label={t("Glossary")}>
      <h2>{t("Glossary")}</h2>
      <form onSubmit={send}>
        <label htmlFor={`${id}-t`}>{t("Term")}</label>
        <input
          id={`${id}-t`}
          maxLength={40}
          value={term}
          onChange={(e) => setTerm(e.target.value)}
        />
        <label htmlFor={`${id}-m`}>{t("Meaning")}</label>
        <input
          id={`${id}-m`}
          maxLength={200}
          value={meaning}
          onChange={(e) => setMeaning(e.target.value)}
        />
        <label htmlFor={`${id}-s`}>
          {t("Other spellings, separated by commas (at most 10)")}
        </label>
        <input
          id={`${id}-s`}
          value={spellings}
          onChange={(e) => setSpellings(e.target.value)}
        />
        <button
          type="submit"
          disabled={
            term.trim() === "" || meaning.trim() === "" || list.length > 10
          }
        >
          {t("Suggest a term")}
        </button>
      </form>
      {status && <p role="status">{status}</p>}
    </section>
  );
}

// SuggestedEditor is the space's suggested questions, one per line, at most 8 (SB-3307).
function SuggestedEditor({ base, fetcher }: { base: string; fetcher: Fetch }) {
  const id = useId();
  const [text, setText] = useState("");
  const [status, setStatus] = useState("");
  useEffect(() => {
    let live = true;
    call(fetcher, "GET", `${base}/suggested`).then(
      (r) =>
        live &&
        Array.isArray(r.questions) &&
        setText((r.questions as string[]).join("\n")),
      () => {},
    );
    return () => {
      live = false;
    };
  }, [base, fetcher]);
  const lines = text
    .split("\n")
    .map((x) => x.trim())
    .filter((x) => x !== "");
  async function save(e: FormEvent) {
    e.preventDefault();
    try {
      await call(fetcher, "PUT", `${base}/suggested`, { questions: lines });
      setStatus(t("Saved."));
    } catch (err) {
      setStatus((err as Error).message);
    }
  }
  return (
    <section aria-label={t("Suggested questions")}>
      <h2>{t("Suggested questions")}</h2>
      <form onSubmit={save}>
        <label htmlFor={id}>
          {t(
            "One question per line, at most 8, shown to the space's members when they start",
          )}
        </label>
        <textarea
          id={id}
          rows={4}
          value={text}
          onChange={(e) => setText(e.target.value)}
        />
        <button
          type="submit"
          disabled={lines.length > 8 || lines.some((x) => x.length > 120)}
        >
          {t("Save")}
        </button>
      </form>
      {status && <p role="status">{status}</p>}
    </section>
  );
}
