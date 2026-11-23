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
  // A steward's review mark at this version, YYYY-MM-DD (SB-2702).
  reviewed?: string;
  // The title of the document a steward retired for this one, when the asker can open it (SB-2801).
  replaces?: string;
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
  terms?: Term[];
  // Most cited documents stale, and connectors behind on syncing (SB-2701, SB-2704).
  notice_stale?: { stale: number; of: number; oldest_days: number };
  notice_sync?: { kind: string; last_success: string | null }[];
}

// A space glossary entry the question used (SB-2203).
export interface Term {
  term: string;
  means: string;
}

export interface ApiError {
  code: string;
  message: string;
  trace: string;
  remedy: string;
  retry_after?: number; // seconds, with ERATE (SB-2305)
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
  outdated_citation?: string;
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
  terms?: Term[];
}

export interface SavedQuestion {
  id: string;
  question: string;
  space: string;
  at: string;
  watch?: boolean;
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

// Filters narrow a search after every check (SB-2404): a type and a number of days.
export interface Filters {
  type?: string;
  since?: number;
}

export function search(q: string, f: Fetch = fetch, space?: string, filters: Filters = {}): Promise<SearchResponse> {
  let path = `/v1/search?q=${encodeURIComponent(q)}`;
  if (filters.type) path += `&type=${encodeURIComponent(filters.type)}`;
  if (filters.since) path += `&since=${filters.since}d`;
  return call<SearchResponse>(inSpace(path, space), {}, f);
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
  // A steward's live claim (SB-2502): who, whether it is the caller's, and until when.
  claim?: { by: string; mine: boolean; until: string };
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

// Stewards' tools (sb-ask(3); SB-2501 to SB-2503).
const sp = (space: string) => `/v1/spaces/${encodeURIComponent(space)}`;

export function markReviewed(space: string, cite: string, f: Fetch = fetch): Promise<{ document: string; version: number }> {
  return call(`${sp(space)}/review/marks`, post({ cite }), f);
}

export async function claimQuestion(space: string, id: string, release: boolean, f: Fetch = fetch): Promise<QueueQuestion["claim"] | null> {
  const d = await call<{ claim?: QueueQuestion["claim"] | null }>(`${sp(space)}/questions/${encodeURIComponent(id)}/claim`, release ? { method: "DELETE" } : post({}), f);
  return d.claim ?? null;
}

export interface ReplyTemplate {
  id: string;
  name: string;
  text: string;
}

export async function replyTemplates(space: string, f: Fetch = fetch): Promise<ReplyTemplate[]> {
  try {
    const d = await call<{ items?: ReplyTemplate[] }>(`${sp(space)}/templates`, {}, f);
    return Array.isArray(d.items) ? d.items.filter((x) => typeof x.name === "string" && typeof x.text === "string") : [];
  } catch {
    return [];
  }
}

export function saveTemplate(space: string, id: string, name: string, text: string, f: Fetch = fetch): Promise<ReplyTemplate> {
  const body = post({ name, text });
  return id === "" ? call(`${sp(space)}/templates`, body, f) : call(`${sp(space)}/templates/${encodeURIComponent(id)}`, { ...body, method: "PUT" }, f);
}

export function deleteTemplate(space: string, id: string, f: Fetch = fetch): Promise<unknown> {
  return call(`${sp(space)}/templates/${encodeURIComponent(id)}`, { method: "DELETE" }, f);
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
  // Reports that it is out of date (SB-2703).
  outdated?: number;
  // A note read wider than its space now (SB-2004): how many readers to remove, and its question.
  wider?: number;
  question?: string;
  // Its owner at the source is no active user (SB-2201); copies of its text you can open (SB-2202).
  owner_gone?: boolean;
  copies?: SearchResult[];
  // Traces of the answers reported as citing it wrongly (SB-2204).
  reports?: string[];
  // A note: how many of its sources have a newer version or are gone (SB-2804).
  sources_moved?: number;
}

// Documents with the same title and different text (SB-2802).
export interface SameNameGroup {
  title: string;
  items: SearchResult[];
  hidden: number;
}

// personal_data is how many documents in the space hold an IBAN or a card number (SB-1904).
export async function reviewList(
  space: string,
  f: Fetch = fetch,
): Promise<{ items: ReviewItem[]; hidden: { owner?: string; owner_gone?: boolean }[]; personal_data?: number; glossary_flagged?: string[]; same_name?: SameNameGroup[] }> {
  try {
    const d = await call<{
      items?: ReviewItem[];
      hidden?: { owner?: string; owner_gone?: boolean }[];
      personal_data?: number;
      glossary_flagged?: string[];
      same_name?: SameNameGroup[];
    }>(
      `/v1/spaces/${encodeURIComponent(space)}/review`,
      {},
      f,
    );
    return {
      items: Array.isArray(d.items) ? d.items : [],
      hidden: Array.isArray(d.hidden) ? d.hidden : [],
      personal_data: typeof d.personal_data === "number" ? d.personal_data : undefined,
      glossary_flagged: Array.isArray(d.glossary_flagged) ? d.glossary_flagged : undefined,
      same_name: Array.isArray(d.same_name) ? d.same_name.filter((g) => Array.isArray(g.items)) : undefined,
    };
  } catch {
    return { items: [], hidden: [] };
  }
}

// Retire from answers (sb-ask(3); SB-2801): only in the steward's own space, undone with restore.
export interface Retired {
  document: string;
  title?: string;
  reason: string;
  replaced_by?: string;
  by: string;
  at: string;
}

export function retireDocument(space: string, cite: string, reason: string, replacedBy: string, f: Fetch = fetch): Promise<Retired> {
  return call(`${sp(space)}/retired`, post({ cite, reason, replaced_by: replacedBy }), f);
}

export async function retiredList(space: string, f: Fetch = fetch): Promise<{ items: Retired[]; hidden: number }> {
  try {
    const d = await call<{ items?: Retired[]; hidden?: number }>(`${sp(space)}/retired`, {}, f);
    const items = Array.isArray(d.items) ? d.items.filter((x) => typeof x.document === "string" && typeof x.reason === "string" && typeof x.at === "string") : [];
    return { items, hidden: typeof d.hidden === "number" ? d.hidden : 0 };
  } catch {
    return { items: [], hidden: 0 };
  }
}

export function restoreDocument(space: string, document: string, f: Fetch = fetch): Promise<{ document: string; restored: boolean }> {
  return call(`${sp(space)}/retired/restore`, post({ document }), f);
}

// Never cited (SB-2803): documents older than 90 days no answer cited in 180 days.
export async function uncitedList(space: string, f: Fetch = fetch): Promise<{ items: SearchResult[]; hidden: { owner?: string }[] }> {
  try {
    const d = await call<{ items?: SearchResult[]; hidden?: { owner?: string }[] }>(`${sp(space)}/review?section=uncited`, {}, f);
    return { items: Array.isArray(d.items) ? d.items : [], hidden: Array.isArray(d.hidden) ? d.hidden : [] };
  } catch {
    return { items: [], hidden: [] };
  }
}

// The handover brief (SB-1707, SB-2002): the person's documents the asker can open now.
export interface HandoverBrief {
  trace: string;
  person: string;
  answer: { claims: Claim[] };
  dropped: number;
  hidden: number;
  locations?: Record<string, string>;
}

export function handover(person: string, f: Fetch = fetch): Promise<HandoverBrief> {
  return call<HandoverBrief>("/v1/handover", post({ person }), f);
}

// Notification settings (sb-ask(3), Notifications; SB-2101): the kinds turned off.
export interface NotifyPrefs {
  off: string[];
  // Until when everything is paused (SB-2506); none is not paused.
  paused_until?: string;
}

function prefsOf(d: { off?: string[]; paused_until?: string }): NotifyPrefs {
  return { off: Array.isArray(d.off) ? d.off : [], ...(typeof d.paused_until === "string" ? { paused_until: d.paused_until } : {}) };
}

export async function notifySettings(f: Fetch = fetch): Promise<NotifyPrefs> {
  return prefsOf(await call<{ off?: string[]; paused_until?: string }>("/v1/me/notifications", {}, f));
}

// setNotifySettings replaces the kinds turned off; pauseDays, when given, pauses for that
// many days from now (0 resumes), and otherwise the pause stays as it is.
// Send me a test (SB-2602): the fixed test text through the cell's real channel.
export function testNotification(f: Fetch = fetch): Promise<{ sent: boolean; via: string }> {
  return call("/v1/me/notifications/test", post({}), f);
}

export async function setNotifySettings(off: string[], f: Fetch = fetch, pauseDays?: number): Promise<NotifyPrefs> {
  const body = pauseDays === undefined ? { off } : { off, pause_days: pauseDays };
  return prefsOf(await call("/v1/me/notifications", { method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify(body) }, f));
}

// Watch a saved question (SB-2104): told weekly when its sources change.
export function watchSaved(id: string, watch: boolean, f: Fetch = fetch): Promise<{ id: string; watch: boolean }> {
  return call(
    `/v1/saved/${encodeURIComponent(id)}/watch`,
    { method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ watch }) },
    f,
  );
}

// Ask the owners of sources an answer couldn't show to share them (SB-2205): counts only.
export function requestAccess(trace: string, f: Fetch = fetch): Promise<{ sent: number; already: number; no_owner: number }> {
  return call("/v1/access-requests", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ trace }) }, f);
}

// A wrong-citation report as an eval case (SB-2204): the document the answer should cite.
export function addEvalCase(space: string, report: string, expect: string, f: Fetch = fetch): Promise<{ id: string; question: string }> {
  return call(
    `/v1/spaces/${encodeURIComponent(space)}/evals`,
    { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ report, expect }) },
    f,
  );
}

// People the caller may brief, by name or email (SB-2106).
export interface Briefable {
  id: string;
  name: string;
  email: string;
  title?: string;
}

export async function briefable(q: string, f: Fetch = fetch): Promise<Briefable[]> {
  const data = await call<{ items?: Briefable[] }>(`/v1/handover/people?q=${encodeURIComponent(q)}`, {}, f);
  return Array.isArray(data.items) ? data.items : [];
}

// ---- Read-only API tokens (sb-api(3), Read-only API tokens; SB-2306) ----

export interface ApiToken {
  id: string;
  name: string;
  created: string;
  expires: string;
  token?: string; // once, when it is made
}

export async function listTokens(f: Fetch = fetch): Promise<ApiToken[]> {
  const data = await call<{ items?: ApiToken[] }>("/v1/tokens", {}, f);
  return Array.isArray(data.items) ? data.items : [];
}

export function makeToken(name: string, days: number, f: Fetch = fetch): Promise<ApiToken> {
  return call<ApiToken>("/v1/tokens", post({ name, days }), f);
}

export function revokeToken(id: string, f: Fetch = fetch): Promise<{ revoked: boolean }> {
  return call(`/v1/tokens/${encodeURIComponent(id)}`, { method: "DELETE" }, f);
}
