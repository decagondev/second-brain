// Types and calls for sb-ask(3). Clients ignore fields they don't know (sb-api(3)).

export interface Cite {
  cite: string;
  title?: string;
  span: [number, number];
  url?: string;
  flags: string[];
}

export interface Claim {
  text: string;
  cites: Cite[];
  uncited?: boolean;
}

export type Notice = "most_uncited" | "no_access" | "no_sources" | null;

export interface AskResponse {
  trace: string;
  release: string;
  answer: { claims: Claim[] };
  refused: number;
  notice: Notice;
  conversation: string;
}

export interface ApiError {
  code: string;
  message: string;
  trace: string;
  remedy: string;
}

export class AskError extends Error {
  constructor(readonly error: ApiError) {
    super(error.message);
  }
}

export type Fetch = (input: string, init?: RequestInit) => Promise<Response>;

export async function ask(question: string, conversation: string | undefined, f: Fetch = fetch): Promise<AskResponse> {
  const body: Record<string, string> = { question };
  if (conversation) body.conversation = conversation;
  let res: Response;
  try {
    res = await f("/v1/ask", {
      method: "POST",
      credentials: "same-origin",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    });
  } catch {
    throw new AskError({ code: "EUNAVAIL", message: "Answers are unavailable right now.", trace: "", remedy: "Try again in a minute." });
  }
  const data: unknown = await res.json().catch(() => null);
  if (!res.ok) {
    const e = (data as { error?: ApiError } | null)?.error;
    throw new AskError(e ?? { code: "EUNAVAIL", message: "Answers are unavailable right now.", trace: "", remedy: "Try again in a minute." });
  }
  return data as AskResponse;
}

// A claim is shown as cited only if it has at least one citation and isn't marked
// uncited; anything else is shown as uncited (LAW-14: never as plain text).
export function isCited(c: Claim): boolean {
  return !c.uncited && Array.isArray(c.cites) && c.cites.length > 0;
}
