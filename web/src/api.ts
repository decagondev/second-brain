// Types and calls for sb-ask(3). Clients ignore fields they don't know (sb-api(3)).

export interface Cite {
  cite: string;
  title?: string;
  span: [number, number];
  url?: string;
  flags: string[];
  // In a cell with spaces (SB-1007): the spaces that hold the source, and its label.
  spaces?: string[];
  label?: string;
}

export interface Space {
  name: string;
  label: string;
}

// The asker's spaces (GET /v1/spaces). Any failure shows no switcher, so the UI asks
// company-wide; it never guesses a space.
export async function listSpaces(f: Fetch = fetch): Promise<Space[]> {
  try {
    const res = await f("/v1/spaces", { credentials: "same-origin" });
    if (!res.ok) return [];
    const data = (await res.json()) as { spaces?: unknown };
    if (!Array.isArray(data.spaces)) return [];
    return data.spaces.filter((s): s is Space => typeof s?.name === "string" && typeof s?.label === "string");
  } catch {
    return [];
  }
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

export async function ask(question: string, conversation: string | undefined, f: Fetch = fetch, space?: string): Promise<AskResponse> {
  const body: Record<string, string> = { question };
  if (conversation) body.conversation = conversation;
  const url = space ? `/v1/ask?space=${encodeURIComponent(space)}` : "/v1/ask";
  let res: Response;
  try {
    res = await f(url, {
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

export interface FeedbackRequest {
  trace: string;
  rating: "up" | "down";
  wrong_citation?: string;
  comment?: string;
}

export async function sendFeedback(req: FeedbackRequest, f: Fetch = fetch): Promise<void> {
  let res: Response;
  try {
    res = await f("/v1/feedback", {
      method: "POST",
      credentials: "same-origin",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(req),
    });
  } catch {
    throw new AskError({ code: "EUNAVAIL", message: "Feedback couldn't be sent.", trace: req.trace, remedy: "Try again in a minute." });
  }
  if (!res.ok) {
    const data = (await res.json().catch(() => null)) as { error?: ApiError } | null;
    throw new AskError(data?.error ?? { code: "EUNAVAIL", message: "Feedback couldn't be sent.", trace: req.trace, remedy: "Try again in a minute." });
  }
}
