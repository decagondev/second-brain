import { useEffect, useState } from "react";
import type { Claim, Fetch } from "./api";
import { t, tn } from "./i18n";

// Your conversations (sb-ask(3), Your conversations; SB-5802 to SB-5804): the ones you
// started in a chat app, and with Keep my conversations on every device on, the ones you
// started on the web, each answer checked again as you now. Continue here asks the next
// question in the same conversation. Nothing is fetched until the list is opened.

export interface ConversationItem {
  id: string;
  client: string;
  first: string;
  turns: number;
  at: string;
  synced?: boolean;
}

export interface ConversationTurn {
  question?: string;
  trace: string;
  at: string;
  claims?: Claim[];
  label?: string;
  unavailable?: boolean;
}

export interface ConversationView {
  id?: string;
  turns: ConversationTurn[];
  left?: number;
  label?: string;
}

async function getJSON<T>(
  fetcher: Fetch,
  url: string,
  init?: RequestInit,
): Promise<T> {
  const res = await fetcher(url, { credentials: "same-origin", ...init });
  if (!res.ok) throw new Error(String(res.status));
  return (await res.json()) as T;
}

// Turns shows a conversation's answers: an answer you can no longer open shows as such.
export function Turns({ view }: { view: ConversationView }) {
  return (
    <ol className="conversation-turns">
      {view.turns.map((x) => (
        <li key={x.trace}>
          {x.question && <p className="question">{x.question}</p>}
          {x.unavailable ? (
            <p className="unavailable">{t("No longer available to you")}</p>
          ) : (
            <ul>
              {(x.claims ?? []).map((c, i) => (
                <li
                  key={i}
                  className={c.uncited ? "claim uncited" : "claim cited"}
                >
                  {c.uncited && <strong>{t("Uncited")}: </strong>}
                  {c.text} {c.cites.map((ci) => ci.title || ci.cite).join(", ")}
                </li>
              ))}
            </ul>
          )}
        </li>
      ))}
    </ol>
  );
}

export function MyConversations({
  fetcher = fetch,
  onContinue,
}: {
  fetcher?: Fetch;
  onContinue: (id: string) => void;
}) {
  const [items, setItems] = useState<ConversationItem[]>();
  const [open, setOpen] = useState<ConversationView>();
  const [status, setStatus] = useState("");
  async function load() {
    try {
      const d = await getJSON<{ conversations: ConversationItem[] }>(
        fetcher,
        "/v1/me/conversations",
      );
      setItems(d.conversations);
    } catch {
      setStatus(t("Your conversations can't be read right now."));
    }
  }
  async function cont(id: string) {
    try {
      const v = await getJSON<ConversationView>(
        fetcher,
        "/v1/me/conversations/" + encodeURIComponent(id),
      );
      setOpen(v);
      onContinue(id);
    } catch {
      setStatus(t("Your conversations can't be read right now."));
    }
  }
  async function share(id: string) {
    try {
      const d = await getJSON<{ url: string }>(
        fetcher,
        "/v1/me/conversations/" + encodeURIComponent(id) + "/share",
        { method: "POST" },
      );
      setStatus(
        t("Link to share: {url}").replace(
          "{url}",
          window.location.origin + d.url,
        ),
      );
    } catch {
      setStatus(t("The conversation couldn't be shared."));
    }
  }
  return (
    <details
      className="recent"
      onToggle={(e) =>
        (e.currentTarget as HTMLDetailsElement).open &&
        items === undefined &&
        void load()
      }
    >
      <summary>{t("Your conversations")}</summary>
      {items && items.length === 0 && (
        <p>{t("No conversations from chat or your other devices yet.")}</p>
      )}
      <ul>
        {(items ?? []).map((c) => (
          <li key={c.id}>
            {c.first}{" "}
            <small>
              (
              {c.client === "chat"
                ? t("started in chat")
                : t("on another device")}
              )
            </small>{" "}
            <button type="button" onClick={() => void cont(c.id)}>
              {t("Continue here")}
            </button>{" "}
            <button type="button" onClick={() => void share(c.id)}>
              {t("Share conversation")}
            </button>
          </li>
        ))}
      </ul>
      {open && (
        <section aria-label={t("Conversation")}>
          <Turns view={open} />
          <p>{t("Ask below to continue this conversation.")}</p>
        </section>
      )}
      {status && <p role="status">{status}</p>}
    </details>
  );
}

// KeepOnEveryDevice is the person's choice on My data (SB-5803), off by default.
export function KeepOnEveryDevice({ fetcher = fetch }: { fetcher?: Fetch }) {
  const [on, setOn] = useState<boolean>();
  const [status, setStatus] = useState("");
  async function set(next: boolean) {
    try {
      const d = await getJSON<{ sync: boolean }>(
        fetcher,
        "/v1/me/conversations/sync",
        {
          method: "PUT",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ on: next }),
        },
      );
      setOn(d.sync);
      setStatus(
        d.sync
          ? t(
              "Your web conversations are kept for your other devices for 30 days.",
            )
          : t(
              "Your conversations are kept in this browser only; the copy for other devices is deleted.",
            ),
      );
    } catch {
      setStatus(t("Your choice couldn't be saved. Try again."));
    }
  }
  return (
    <p>
      <label>
        <input
          type="checkbox"
          checked={on ?? false}
          onChange={(e) => void set(e.target.checked)}
        />{" "}
        {t("Keep my conversations on every device")}
      </label>
      {status && <span role="status"> {status}</span>}
    </p>
  );
}

// SharedConversation is a colleague's conversation, at /c/sh_… (SB-5804): each answer
// checked as you, the ones you can't open every cite of left out and counted.
export function SharedConversation({
  fetcher = fetch,
  id,
}: {
  fetcher?: Fetch;
  id: string;
}) {
  const [view, setView] = useState<ConversationView>();
  const [status, setStatus] = useState("");
  useEffect(() => {
    getJSON<ConversationView>(
      fetcher,
      "/v1/conversation-shares/" + encodeURIComponent(id),
    )
      .then(setView)
      .catch(() =>
        setStatus(
          t("This conversation isn't shared with you, or the link has ended."),
        ),
      );
  }, [fetcher, id]);
  return (
    <section aria-label={t("Shared conversation")}>
      <h2>{t("Shared conversation")}</h2>
      {view?.label === "confidential" || view?.label === "restricted" ? (
        <p>
          <strong>
            {t(
              "Confidential: this answer cites confidential documents. Don't forward it.",
            )}
          </strong>
        </p>
      ) : null}
      {view && <Turns view={view} />}
      {view?.left ? (
        <p>
          {tn(
            view.left,
            "{n} answer not shared with you.",
            "{n} answers not shared with you.",
          )}
        </p>
      ) : null}
      {status && <p role="status">{status}</p>}
    </section>
  );
}

// Where you left off (SB-5805): up to 3 of your own open things, shown when a new
// conversation starts; Not now hides them for today in this browser.
interface LeftOffItem {
  kind: string;
  id: string;
  title: string;
  link: string;
}

const hiddenKey = "sb.leftoff.hidden";
const today = () => new Date().toISOString().slice(0, 10);

function hiddenToday(): boolean {
  try {
    return window.localStorage.getItem(hiddenKey) === today();
  } catch {
    return false;
  }
}

export function LeftOff({
  fetcher = fetch,
  start,
}: {
  fetcher?: Fetch;
  start: boolean;
}) {
  const [items, setItems] = useState<LeftOffItem[]>([]);
  const [hidden, setHidden] = useState(hiddenToday);
  useEffect(() => {
    if (!start || hidden) return;
    getJSON<{ items: LeftOffItem[] }>(fetcher, "/v1/me/left-off")
      .then((d) => setItems(d.items ?? []))
      .catch(() => setItems([]));
  }, [fetcher, start, hidden]);
  if (hidden || items.length === 0) return null;
  return (
    <aside className="left-off" aria-label={t("Where you left off")}>
      <h3>{t("Where you left off")}</h3>
      <ul>
        {items.map((x) => (
          <li key={x.kind + x.id}>
            <a href={x.link}>{x.title}</a>
          </li>
        ))}
      </ul>
      <button
        type="button"
        onClick={() => {
          try {
            window.localStorage.setItem(hiddenKey, today());
          } catch {
            // storage off: hidden for this page only
          }
          setHidden(true);
        }}
      >
        {t("Not now")}
      </button>
    </aside>
  );
}
