import type { Fetch } from "./api";
import { t, tn } from "./i18n";

// Branch a conversation (SB-5206): Ask from here starts a new conversation with the
// questions up to an earlier one, and the conversation it left is kept in this browser so
// the asker can go back to it. Each branch is its own conversation, with its own download.

export interface Branch<T> {
  conversation?: string;
  turns: T[];
}

// branchFrom asks the cell for a new conversation with the first at questions of conversation.
export async function branchFrom(fetcher: Fetch, conversation: string, at: number): Promise<string> {
  const r = await fetcher(`/v1/conversations/${encodeURIComponent(conversation)}/branch`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ at }),
  });
  const body = (await r.json().catch(() => ({}))) as { conversation?: string; error?: { message?: string } };
  if (!r.ok || !body.conversation) throw new Error(body.error?.message ?? t("The conversation couldn't be branched."));
  return body.conversation;
}

// AskFromHere is the button on an earlier answer.
export function AskFromHere({ disabled, onBranch }: { disabled?: boolean; onBranch: () => void }) {
  return (
    <button type="button" disabled={disabled} onClick={onBranch}>
      {t("Ask from here")}
    </button>
  );
}

// OtherBranches lists the branches this browser kept, to go back to one.
export function OtherBranches<T extends { question: string }>({ branches, onOpen }: { branches: Branch<T>[]; onOpen: (i: number) => void }) {
  if (branches.length === 0) return null;
  return (
    <section aria-label={t("Other branches")}>
      <p className="meta">{tn(branches.length, "{n} other branch of this conversation", "{n} other branches of this conversation")}</p>
      <ul>
        {branches.map((b, i) => (
          <li key={i}>
            <button type="button" onClick={() => onOpen(i)}>
              {t("Back to: {question}", { question: b.turns[b.turns.length - 1]?.question ?? "" })}
            </button>
          </li>
        ))}
      </ul>
    </section>
  );
}
