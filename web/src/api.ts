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
  // Who to ask about it (SB-1702): only for one active owner, only on an allowed cite.
  owner?: Person;
  // Its age, and whether that is past the review threshold of a space holding it (SB-1709).
  age_days?: number;
  stale?: boolean;
}

export interface Person {
  email: string;
  title?: string;
}

export interface PersonRef extends Person {
  cites: number;
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
  people?: PersonRef[];
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

// ---- Search, saved and shared questions (sb-ask(3); SB-1701, SB-1703) ----

export interface SearchResult {
  cite: string;
  title?: string;
  url?: string;
  owner?: Person;
  age_days?: number;
  snippet: string;
  flags: string[];
  spaces?: string[];
  label?: string;
}

export interface SearchResponse {
  trace: string;
  release: string;
  results: SearchResult[];
  refused: number;
}

export interface SavedQuestion {
  id: string;
  question: string;
  space: string;
  at: string;
}

export interface SharedQuestion {
  id: string;
  question: string;
  space: string;
  expires: string;
}

const unavailable: ApiError = { code: "EUNAVAIL", message: "This is unavailable right now.", trace: "", remedy: "Try again in a minute." };

// call is one JSON request to the cell's API; any failure is an AskError with the API's error.
async function call<T>(url: string, init: RequestInit, f: Fetch): Promise<T> {
  let res: Response;
  try {
    res = await f(url, { credentials: "same-origin", ...init });
  } catch {
    throw new AskError(unavailable);
  }
  const data: unknown = await res.json().catch(() => null);
  if (!res.ok) throw new AskError((data as { error?: ApiError } | null)?.error ?? unavailable);
  return data as T;
}

function post(body: unknown): RequestInit {
  return { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(body) };
}

function inSpace(path: string, space?: string): string {
  if (!space) return path;
  return `${path}${path.includes("?") ? "&" : "?"}space=${encodeURIComponent(space)}`;
}

export function search(q: string, f: Fetch = fetch, space?: string): Promise<SearchResponse> {
  return call<SearchResponse>(inSpace(`/v1/search?q=${encodeURIComponent(q)}`, space), {}, f);
}

export function saveQuestion(question: string, f: Fetch = fetch, space?: string): Promise<{ id: string }> {
  return call(inSpace("/v1/saved", space), post({ question }), f);
}

export async function listSaved(f: Fetch = fetch): Promise<SavedQuestion[]> {
  const data = await call<{ items?: SavedQuestion[] }>("/v1/saved", {}, f);
  return Array.isArray(data.items) ? data.items : [];
}

export function removeSaved(id: string, f: Fetch = fetch): Promise<unknown> {
  return call(`/v1/saved/${encodeURIComponent(id)}`, { method: "DELETE" }, f);
}

export function shareQuestion(question: string, f: Fetch = fetch, space?: string): Promise<{ id: string; url: string; expires: string }> {
  return call(inSpace("/v1/shares", space), post({ question }), f);
}

export function openShare(id: string, f: Fetch = fetch): Promise<SharedQuestion> {
  return call(`/v1/shares/${encodeURIComponent(id)}`, {}, f);
}

// The space name of a record's space ("spc_finance" -> "finance"; company-wide -> "").
export function spaceName(space: string): string {
  return space.startsWith("spc_") ? space.slice(4) : "";
}

// ---- Questions to the stewards and notes (sb-ask(3); SB-1704, SB-1705) ----

export interface ReplyView {
  text: string;
  at: string;
  cites: SearchResult[];
  dropped: number;
  note: boolean;
}

export interface MyQuestion {
  id: string;
  question: string;
  space: string;
  at: string;
  state: "open" | "answered";
  reply?: ReplyView;
}

export interface QueueQuestion {
  id: string;
  question: string;
  at: string;
  state: "open" | "answered";
  reply?: { by: string; text: string; cites: string[]; at: string; note?: string };
}

export interface NoteResult {
  path: string;
  readers: number;
  narrower: number;
  action?: string;
  state?: string;
  receipt?: string;
}

export function sendToStewards(question: string, trace: string, f: Fetch = fetch, space?: string): Promise<{ id: string; to: string }> {
  return call(inSpace("/v1/questions", space), post({ question, trace }), f);
}

export async function listQuestions(f: Fetch = fetch): Promise<MyQuestion[]> {
  const data = await call<{ items?: MyQuestion[] }>("/v1/questions", {}, f);
  return Array.isArray(data.items) ? data.items : [];
}

export async function stewardQueue(space: string, f: Fetch = fetch): Promise<QueueQuestion[]> {
  const data = await call<{ items?: QueueQuestion[] }>(`/v1/spaces/${encodeURIComponent(space)}/questions`, {}, f);
  return Array.isArray(data.items) ? data.items : [];
}

export function replyToQuestion(space: string, id: string, text: string, cites: string[], f: Fetch = fetch): Promise<unknown> {
  return call(`/v1/spaces/${encodeURIComponent(space)}/questions/${encodeURIComponent(id)}/reply`, post({ text, cites }), f);
}

export function publishNote(space: string, id: string, title: string, confirm: number | undefined, f: Fetch = fetch): Promise<NoteResult> {
  return call(`/v1/spaces/${encodeURIComponent(space)}/questions/${encodeURIComponent(id)}/note`, post(confirm === undefined ? { title } : { title, confirm }), f);
}

// A space's start list for a member (sb-spaces(7), Start here; SB-1706). Any failure shows none.
export async function startHere(space: string, f: Fetch = fetch): Promise<{ items: SearchResult[]; hidden: number }> {
  try {
    const d = await call<{ items?: SearchResult[]; hidden?: number }>(`/v1/start?space=${encodeURIComponent(space)}`, {}, f);
    return { items: Array.isArray(d.items) ? d.items : [], hidden: typeof d.hidden === "number" ? d.hidden : 0 };
  } catch {
    return { items: [], hidden: 0 };
  }
}

// What changed (SB-1708) and documents to review (SB-1709). Any failure shows none.
export async function changes(days: number, f: Fetch = fetch, space?: string): Promise<SearchResult[]> {
  try {
    const d = await call<{ items?: SearchResult[] }>(inSpace(`/v1/changes?since=${days}d`, space), {}, f);
    return Array.isArray(d.items) ? d.items : [];
  } catch {
    return [];
  }
}

export interface ReviewItem extends SearchResult {
  cited: number;
  flagged: number;
}

export async function reviewList(space: string, f: Fetch = fetch): Promise<{ items: ReviewItem[]; hidden: { owner?: string }[] }> {
  try {
    const d = await call<{ items?: ReviewItem[]; hidden?: { owner?: string }[] }>(`/v1/spaces/${encodeURIComponent(space)}/review`, {}, f);
    return { items: Array.isArray(d.items) ? d.items : [], hidden: Array.isArray(d.hidden) ? d.hidden : [] };
  } catch {
    return { items: [], hidden: [] };
  }
}
