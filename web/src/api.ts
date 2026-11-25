// Types and calls for sb-ask(3). Clients ignore fields they don't know (sb-api(3)).

export interface Cite {
  cite: string;
  title?: string;
  // The nearest heading above the cited chunk (SB-3002).
  section?: string;
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
  // Named by a steward for a document whose owner is gone (SB-3405).
  caretaker?: boolean;
}

export interface PersonRef extends Person {
  cites: number;
}

export interface Space {
  name: string;
  label: string;
  parent?: string; // the department above, when the asker is a member of it too (SB-3901)
  teams?: boolean; // the asker is in at least one team under it (SB-3903)
}

// A branch ask (SB-3903) is the space's name with this suffix: the department and the
// teams under it the asker is in.
export const BRANCH = "+teams";

// spaceQuery is the space part of a URL: space=NAME, and branch=true for a branch ask.
export function spaceQuery(space: string): string {
  const branch = space.endsWith(BRANCH);
  const name = branch ? space.slice(0, -BRANCH.length) : space;
  return `space=${encodeURIComponent(name)}${branch ? "&branch=true" : ""}`;
}

// The asker's spaces (GET /v1/spaces). Any failure shows no switcher, so the UI asks
// company-wide; it never guesses a space.
// The admin's notice to everyone while it stands, or null (SB-3207). A failure is no notice.
export type CellNotice = { text: string; until: string };

export async function readNotice(f: Fetch = fetch): Promise<CellNotice | null> {
  try {
    const res = await f("/v1/notice", { credentials: "same-origin" });
    if (!res.ok) return null;
    const d = (await res.json()) as { text?: unknown; until?: unknown };
    if (typeof d?.text !== "string" || d.text === "" || typeof d.until !== "string") return null;
    if (!(Date.parse(d.until) > Date.now())) return null;
    return { text: d.text, until: d.until };
  } catch {
    return null;
  }
}

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
  // The claim's documents say different things (SB-3304).
  disagree?: boolean;
}

// A document the search found that no claim cites (SB-3301): never its text.
export interface SeeAlso {
  cite: string;
  title?: string;
  section?: string;
  span: [number, number];
  age_days?: number;
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
  // Every cite is one document (SB-3005).
  notice_single?: boolean;
  // How many earlier questions of the conversation the model didn't read (SB-2901).
  earlier_dropped?: number;
  see_also?: SeeAlso[];
  // The stewards' verified answer to a question like this one (SB-3601).
  pinned?: Pinned;
  // The cited documents and how many someone has checked (SB-3607).
  verified?: { of: number; marked: number };
  // The one department space whose agent could answer this better (SB-4007).
  suggest?: { space: string; agent: string; terms: string[] };
}

// A verified answer, every cite checked again as the asker (SB-3601).
export interface Pinned {
  id: string;
  question: string;
  text: string;
  cites: string[];
  verified: string;
}

// A pin on the steward screen.
export interface PinItem {
  id: string;
  question: string;
  text: string;
  cites: string[];
  at: string;
  due?: boolean;
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

// citeName is a cite as people read it: "Title › Section", the title alone, or the cite (SB-3002).
export function citeName(c: { cite: string; title?: string; section?: string }): string {
  const title = c.title ?? c.cite;
  return c.section ? `${title} › ${c.section}` : title;
}

// Passage is one cite's own text, checked live as the asker (SB-3001).
export interface Passage {
  cite: string;
  text?: string;
  cut?: boolean;
  gone?: boolean;
}

export async function passage(trace: string, cite: string, start: number, f: Fetch = fetch): Promise<Passage> {
  const q = `cite=${encodeURIComponent(cite)}&start=${encodeURIComponent(String(start))}`;
  let res: Response;
  try {
    res = await f(`/v1/traces/${encodeURIComponent(trace)}/passage?${q}`, { credentials: "same-origin" });
  } catch {
    throw new AskError({ code: "EUNAVAIL", message: "Answers are unavailable right now.", trace, remedy: "Try again in a minute." });
  }
  const data: unknown = await res.json().catch(() => null);
  if (!res.ok) {
    const e = (data as { error?: ApiError } | null)?.error;
    throw new AskError(e ?? { code: "EUNAVAIL", message: "Answers are unavailable right now.", trace, remedy: "Try again in a minute." });
  }
  return data as Passage;
}

// docKey is a cite's document, KIND:ID, without its version (SB-3004).
export function docKey(cite: string): string {
  const at = cite.lastIndexOf("@");
  return at > 0 ? cite.slice(0, at) : cite;
}

// About is a document the next questions are asked about (SB-3004).
export interface About {
  key: string;
  name: string;
}

export const maxAbout = 5;

// A file asked about (sb-ask(3), file; SB-3802): its bytes in base64, read for one ask only.
export interface AskFile {
  name: string;
  data: string;
}

export const maxFileBytes = 2 << 20;

// fileToAsk reads a picked file as an AskFile, or undefined when it is over 2 MB.
export function fileToAsk(f: Blob & { name: string }): Promise<AskFile | undefined> {
  if (f.size > maxFileBytes) return Promise.resolve(undefined);
  return new Promise((resolve, reject) => {
    const r = new FileReader();
    r.onerror = () => reject(r.error ?? new Error("unreadable"));
    r.onload = () => {
      const url = String(r.result ?? "");
      resolve({ name: f.name, data: url.slice(url.indexOf(",") + 1) });
    };
    r.readAsDataURL(f);
  });
}

export async function ask(
  question: string,
  conversation: string | undefined,
  f: Fetch = fetch,
  space?: string,
  language?: string,
  documents?: string[],
  short?: boolean,
  verified?: boolean,
  file?: AskFile,
  agent?: string,
  collection?: string,
): Promise<AskResponse> {
  const body: Record<string, string | string[] | boolean | AskFile> = { question };
  if (collection) body.collection = collection; // SB-4304
  if (agent) body.agent = agent; // a department agent (SB-4007)
  if (file) body.file = file; // SB-3802
  if (short) body.length = "short"; // SB-3303
  if (verified) body.verified = true; // SB-3602
  if (documents && documents.length > 0) body.documents = documents;
  if (conversation) body.conversation = conversation;
  if (language) body.language = language;
  const url = space ? `/v1/ask?${spaceQuery(space)}` : "/v1/ask";
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
  // The asker is the document's current owner at the source (SB-3106).
  mine?: boolean;
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
  routine?: Routine;
}

// A saved question asked on a schedule (SB-4302).
export interface Routine {
  every: "daily" | "weekly";
  day?: string;
  last?: string;
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
  return `${path}${path.includes("?") ? "&" : "?"}${spaceQuery(space)}`;
}

// Filters narrow a search after every check (SB-2404): a type and a number of days.
export interface Filters {
  type?: string;
  since?: number;
  owner?: string; // an owner's documents (SB-3306)
  verified?: boolean; // verified documents only (SB-3602)
}

export function search(q: string, f: Fetch = fetch, space?: string, filters: Filters = {}): Promise<SearchResponse> {
  let path = `/v1/search?q=${encodeURIComponent(q)}`;
  if (filters.type) path += `&type=${encodeURIComponent(filters.type)}`;
  if (filters.since) path += `&since=${filters.since}d`;
  if (filters.owner) path += `&owner=${encodeURIComponent(filters.owner)}`;
  if (filters.verified) path += "&verified=true";
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
  // A request to a department (SB-4104): its category and when it is due.
  request?: RequestMeta;
}

export interface RequestMeta {
  category: string;
  due_days: number;
  due: string;
}

export interface QueueQuestion {
  id: string;
  question: string;
  at: string;
  state: "open" | "answered";
  reply?: { by: string; text: string; cites: string[]; at: string; note?: string };
  // A steward's live claim (SB-2502): who, whether it is the caller's, and until when.
  claim?: { by: string; mine: boolean; until: string };
  // Rerouted from a space with no active steward (SB-3102), or one that left it open
  // `waited` working days (SB-3404).
  from_space?: string;
  waited?: number;
  // The question of the queue it looks like (SB-3402), and open 5 working days (SB-3404).
  like?: string;
  overdue?: boolean;
  // A request to the department (SB-4104); overdue past its due date.
  request?: RequestMeta;
  // A reported answer (SB-3606): who sent it, and only what the steward can open.
  report?: { by: string; note?: string; claims: string[]; documents: SearchResult[]; hidden: number };
  // A chat thread saved to the stewards (SB-3703): who saved it, a link and its messages.
  // agent, kind and hidden: a draft an agent wrote on a schedule or a trigger (SB-4005, SB-4006).
  thread?: { by: string; link: string; text: string; agent?: string; kind?: string; hidden?: boolean; actions?: DraftAction[] };
  // A note a steward started from a found-nothing term (SB-3705).
  draft?: { title: string; comment: string };
}

// Draft a note from a found-nothing term (SB-3705).
export function draftNote(space: string, term: string, count: number, f: Fetch = fetch): Promise<{ id: string }> {
  return call(`/v1/spaces/${encodeURIComponent(space)}/drafts`, post({ term, count }), f);
}

// Ask the owner (SB-3706): the question to one person the answer names in Who to ask.
export function askOwner(question: string, trace: string, to: string, f: Fetch = fetch, space?: string): Promise<{ id: string; to: string }> {
  return call(inSpace("/v1/questions", space), post({ question, trace, to }), f);
}

// An action item as a decisions draft reads it from the transcript (SB-4203).
export interface DraftAction {
  text: string;
  owner: string;
  email?: string;
  due?: string;
}

// A confirmed action item, on its owner's list.
export interface ActionItem extends DraftAction {
  id: string;
  space: string;
  done_at?: string;
  overdue?: boolean;
}

// Confirm a decisions draft's action items, as edited (SB-4203): only then are owners told.
export async function confirmActions(space: string, draft: string, items: DraftAction[], f: Fetch = fetch): Promise<ActionItem[]> {
  const d = await call<{ actions?: ActionItem[] }>(`/v1/spaces/${encodeURIComponent(space)}/actions`, post({ draft, items }), f);
  return d.actions ?? [];
}

// The signed-in person's own action items (SB-4203).
export async function myActions(f: Fetch = fetch): Promise<ActionItem[]> {
  const d = await call<{ actions?: ActionItem[] }>("/v1/me/actions", {}, f);
  return d.actions ?? [];
}

export function actionDone(id: string, f: Fetch = fetch): Promise<ActionItem> {
  return call(`/v1/me/actions/${encodeURIComponent(id)}/done`, post({}), f);
}

// A research report or a meeting brief (SB-4201, SB-4202), checked as the reader when shown.
export interface ResearchSection {
  question: string;
  trace: string;
  claims: Claim[];
  decisions?: Cite[];
  questions?: number;
}

export interface Research {
  id: string;
  kind: string; // research or prep
  question: string;
  space: string;
  state: string; // planned, queued, running, done, stopped, failed
  why?: string;
  steps: string[];
  at: string;
  done_at?: string;
  cost_usd: number;
  stopped?: string;
  summary: Claim[];
  sections: ResearchSection[];
  attendees?: { email: string; known: boolean; owns?: Cite[] }[];
  dropped: number;
  changed?: number; // a routine's sources new since its last run (SB-4302)
}

export interface ResearchItem {
  id: string;
  kind: string;
  question: string;
  space: string;
  state: string;
  steps: number;
  at: string;
  done_at?: string;
}

// Draft a fix (SB-4206): a suggested edit of a document, kept for its owner and told to them.
export function draftFix(space: string, doc: string, f: Fetch = fetch): Promise<{ id?: string; owner: string; changes: number; told: boolean }> {
  return call(`/v1/spaces/${encodeURIComponent(space)}/review/fix`, post({ doc }), f);
}

// Plan a research report: nothing runs until runResearch.
export function planResearch(question: string, f: Fetch = fetch, space?: string): Promise<Research> {
  return call(inSpace("/v1/research", space), post({ question }), f);
}

export function runResearch(id: string, f: Fetch = fetch, space?: string): Promise<Research> {
  return call(inSpace(`/v1/research/${encodeURIComponent(id)}/run`, space), post({}), f);
}

export async function listResearch(f: Fetch = fetch): Promise<ResearchItem[]> {
  const d = await call<{ items?: ResearchItem[] }>("/v1/research", {}, f);
  return d.items ?? [];
}

export function openResearch(id: string, f: Fetch = fetch): Promise<Research> {
  return call(`/v1/research/${encodeURIComponent(id)}`, {}, f);
}

// Prepare a brief from a pasted agenda (SB-4202).
export function prepareBrief(agenda: string, title: string, attendees: string[], f: Fetch = fetch, space?: string): Promise<Research> {
  return call(inSpace("/v1/prepare", space), post({ agenda, title, attendees }), f);
}

// Discard a saved chat thread (SB-3703).
export function discardThread(space: string, id: string, f: Fetch = fetch): Promise<unknown> {
  return call(`/v1/spaces/${encodeURIComponent(space)}/questions/${encodeURIComponent(id)}`, { method: "DELETE" }, f);
}

export interface NoteResult {
  path: string;
  readers: number;
  narrower: number;
  action?: string;
  state?: string;
  receipt?: string;
  // Documents of the space that read the same, checked as the steward (SB-3707).
  similar?: SearchResult[];
}

// Report this answer (SB-3606): only by the asker's click, and it names them.
export function reportAnswer(question: string, trace: string, note: string, f: Fetch = fetch, space?: string): Promise<{ id: string; to: string }> {
  return call(inSpace("/v1/questions", space), post({ question, trace, report: true, ...(note.trim() ? { note: note.trim() } : {}) }), f);
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

export function replyToQuestion(space: string, id: string, text: string, cites: string[], f: Fetch = fetch, both = false): Promise<unknown> {
  return call(`/v1/spaces/${encodeURIComponent(space)}/questions/${encodeURIComponent(id)}/reply`, post(both ? { text, cites, both } : { text, cites }), f);
}

export function publishNote(space: string, id: string, title: string, confirm: number | undefined, f: Fetch = fetch, keepQuestion = false): Promise<NoteResult> {
  const keep = keepQuestion ? { keep_question: true } : {};
  return call(`/v1/spaces/${encodeURIComponent(space)}/questions/${encodeURIComponent(id)}/note`, post(confirm === undefined ? { title, ...keep } : { title, confirm, ...keep }), f);
}

// Stewards' tools (sb-ask(3); SB-2501 to SB-2503).
const sp = (space: string) => `/v1/spaces/${encodeURIComponent(space)}`;

// markReviewed is Reviewed, still right; days is Review again in (SB-3603), 0 the space's default.
export function markReviewed(space: string, cite: string, f: Fetch = fetch, days = 0): Promise<{ document: string; version: number }> {
  return call(`${sp(space)}/review/marks`, post(days ? { cite, days } : { cite }), f);
}

// The space's verified answers (SB-3601).
export async function listPins(space: string, f: Fetch = fetch): Promise<PinItem[]> {
  const d = await call<{ items?: PinItem[] }>(`${sp(space)}/pins`, {}, f);
  return Array.isArray(d.items) ? d.items : [];
}

export function pinReply(space: string, questionId: string, question: string, f: Fetch = fetch): Promise<PinItem> {
  return call(`${sp(space)}/pins`, post({ question_id: questionId, question }), f);
}

export function unpin(space: string, id: string, f: Fetch = fetch): Promise<unknown> {
  return call(`${sp(space)}/pins/${encodeURIComponent(id)}`, { method: "DELETE" }, f);
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
  // Not helpful ratings since its review (SB-3604).
  unhelpful?: number;
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

// Of the answers that cite anything, how many rested mostly on stale cites (SB-2807).
export interface StaleCount {
  answers: number;
  stale: number;
}
export interface Freshness {
  this_month: StaleCount;
  last_month: StaleCount;
}

// Questions that found nothing (SB-3101) and the space's feedback (SB-3104): counts only.
export interface FoundNothing {
  this_week: number;
  last_week: number;
  terms: string[];
  counts?: Record<string, number>;
}
export interface Ratings {
  helpful: number;
  not_helpful: number;
}
export interface StewardFeedback {
  this_month: Ratings;
  last_month: Ratings;
  documents: SearchResult[];
  documents_hidden: number;
}

// Each document's outcome in a bulk mark or retirement (SB-3103).
export interface BulkItem {
  cite: string;
  ok?: boolean;
  error?: { code: string; message: string };
}

export async function markReviewedMany(space: string, cites: string[], f: Fetch = fetch): Promise<BulkItem[]> {
  const d = await call<{ items?: BulkItem[] }>(`${sp(space)}/review/marks`, post({ cites }), f);
  return Array.isArray(d.items) ? d.items : [];
}

export async function retireMany(space: string, cites: string[], reason: string, f: Fetch = fetch): Promise<BulkItem[]> {
  const d = await call<{ items?: BulkItem[] }>(`${sp(space)}/retired`, post({ cites, reason }), f);
  return Array.isArray(d.items) ? d.items : [];
}

// A pair of documents disagreeing claims cited this month (SB-3605).
export interface Conflict {
  a: SearchResult;
  b: SearchResult;
  answers: number;
}

export interface ReviewData {
  conflicts?: Conflict[];
  conflicts_hidden?: number;
  found_nothing?: FoundNothing;
  feedback?: StewardFeedback;
  items: ReviewItem[];
  hidden: { owner?: string; owner_gone?: boolean }[];
  personal_data?: number;
  glossary_flagged?: string[];
  same_name?: SameNameGroup[];
  freshness?: Freshness;
  // Eval cases whose expected document changed (SB-2904), and start_here items gone stale (SB-2907).
  evals_to_check?: { id: string; question: string; check: string }[];
  start_here?: StartItem[];
  start_here_hidden?: number;
  // Terms the space's documents define that its glossary lacks (SB-3407).
  terms?: DefinedTerm[];
  // The space's own usage over the last 7 days (SB-3506).
  usage?: SpaceUsage;
  // A department's teams, counts only (SB-3905).
  teams?: TeamRollup[];
}

// One team's counts on its department's list (SB-3905): never a title or a question.
export interface TeamRollup {
  space: string;
  to_review: number;
  overdue: number;
  open_questions: number;
  found_nothing?: number;
  asks?: number;
  fewer_than_5?: boolean;
}

// Who's who in a space (SB-3906): its stewards, the owners of its most-cited documents the
// member can open, and how many members it has.
export interface SpacePeople {
  space: string;
  parent?: string;
  stewards: Person[];
  owners: { email: string; title?: string; cites: number; caretaker?: boolean }[];
  members: number;
}

// A space's people for a member. Any failure shows none.
export async function spacePeople(space: string, f: Fetch = fetch): Promise<SpacePeople | undefined> {
  try {
    const d = await call<Partial<SpacePeople>>(`${sp(space)}/people`, {}, f);
    if (!Array.isArray(d.stewards) || !Array.isArray(d.owners) || typeof d.members !== "number") return undefined;
    return { space, parent: d.parent, stewards: d.stewards, owners: d.owners, members: d.members };
  } catch {
    return undefined;
  }
}

// One space's usage row (SB-3506): no shares with fewer than 5 asks.
export interface SpaceUsage {
  space: string;
  asks?: number;
  fewer_than_5?: boolean;
  found_nothing_share?: number;
  helpful_share?: number;
}

export interface DefinedTerm {
  term: string;
  meaning: string;
  docs: number;
}

// A document's caretaker (SB-3405): named by a steward, or removed.
export function nameCaretaker(space: string, doc: string, email: string, f: Fetch = fetch): Promise<{ doc: string; email: string }> {
  return call(`/v1/spaces/${encodeURIComponent(space)}/caretakers`, { method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ doc, email }) }, f);
}

export function removeCaretaker(space: string, doc: string, f: Fetch = fetch): Promise<unknown> {
  return call(`/v1/spaces/${encodeURIComponent(space)}/caretakers?doc=${encodeURIComponent(doc)}`, { method: "DELETE" }, f);
}

export interface StartItem {
  item: string;
  result?: SearchResult;
  why: string[];
  members_without: number;
  members: number;
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
): Promise<ReviewData> {
  try {
    const d = await call<{
      items?: ReviewItem[];
      hidden?: { owner?: string; owner_gone?: boolean }[];
      personal_data?: number;
      glossary_flagged?: string[];
      same_name?: SameNameGroup[];
      freshness?: Freshness;
      evals_to_check?: { id: string; question: string; check: string }[];
      start_here?: StartItem[];
      start_here_hidden?: number;
      found_nothing?: FoundNothing;
      feedback?: StewardFeedback;
      conflicts?: Conflict[];
      conflicts_hidden?: number;
      terms?: DefinedTerm[];
      usage?: SpaceUsage;
      teams?: TeamRollup[];
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
      evals_to_check: Array.isArray(d.evals_to_check) ? d.evals_to_check : undefined,
      start_here: Array.isArray(d.start_here) ? d.start_here.filter((x) => typeof x.item === "string" && Array.isArray(x.why)) : undefined,
      start_here_hidden: typeof d.start_here_hidden === "number" ? d.start_here_hidden : undefined,
      freshness: d.freshness && typeof d.freshness.this_month?.answers === "number" && typeof d.freshness.last_month?.answers === "number" ? d.freshness : undefined,
      found_nothing: d.found_nothing && typeof d.found_nothing.this_week === "number" && Array.isArray(d.found_nothing.terms) ? d.found_nothing : undefined,
      feedback: d.feedback && typeof d.feedback.this_month?.helpful === "number" && Array.isArray(d.feedback.documents) ? d.feedback : undefined,
      terms: Array.isArray(d.terms) ? d.terms.filter((x) => typeof x.term === "string" && typeof x.meaning === "string") : undefined,
      usage: d.usage && typeof d.usage.space === "string" ? d.usage : undefined,
      teams: Array.isArray(d.teams) ? d.teams.filter((x) => typeof x.space === "string" && typeof x.to_review === "number") : undefined,
      conflicts: Array.isArray(d.conflicts) ? d.conflicts.filter((c) => c.a && c.b && typeof c.answers === "number") : undefined,
      conflicts_hidden: typeof d.conflicts_hidden === "number" ? d.conflicts_hidden : undefined,
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
// unused_terms are the space's glossary terms no question used in 180 days (SB-2805).
export async function uncitedList(space: string, f: Fetch = fetch): Promise<{ items: SearchResult[]; hidden: { owner?: string }[]; unused_terms?: string[] }> {
  try {
    const d = await call<{ items?: SearchResult[]; hidden?: { owner?: string }[]; unused_terms?: string[] }>(`${sp(space)}/review?section=uncited`, {}, f);
    return {
      items: Array.isArray(d.items) ? d.items : [],
      hidden: Array.isArray(d.hidden) ? d.hidden : [],
      unused_terms: Array.isArray(d.unused_terms) ? d.unused_terms : undefined,
    };
  } catch {
    return { items: [], hidden: [] };
  }
}

// Coming due (SB-3105): relied-on documents that turn stale in the next 14 days.
export interface DueItem extends SearchResult {
  due_in_days: number;
}

export async function dueList(space: string, f: Fetch = fetch): Promise<{ items: DueItem[]; hidden: { owner?: string }[] }> {
  const d = await call<{ items?: DueItem[]; hidden?: { owner?: string }[] }>(`${sp(space)}/review?section=due`, {}, f);
  return { items: Array.isArray(d.items) ? d.items : [], hidden: Array.isArray(d.hidden) ? d.hidden : [] };
}

// An owner's Reviewed, still right on their own document (SB-3106).
export function markOwnReviewed(cite: string, f: Fetch = fetch, days = 0): Promise<{ document: string; version: number; spaces: string[] }> {
  return call("/v1/documents/reviewed", post(days ? { cite, days } : { cite }), f);
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
  // "daily" for the daily brief (SB-3805); none is weekly.
  digest?: string;
  // A brief before each meeting with an agenda (SB-4202).
  prep?: boolean;
}

function prefsOf(d: { off?: string[]; paused_until?: string; digest?: string; prep?: boolean }): NotifyPrefs {
  return {
    off: Array.isArray(d.off) ? d.off : [],
    ...(typeof d.paused_until === "string" ? { paused_until: d.paused_until } : {}),
    ...(d.digest === "daily" ? { digest: "daily" } : {}),
    ...(d.prep === true ? { prep: true } : {}),
  };
}

export async function notifySettings(f: Fetch = fetch): Promise<NotifyPrefs> {
  return prefsOf(await call<{ off?: string[]; paused_until?: string; digest?: string; prep?: boolean }>("/v1/me/notifications", {}, f));
}

// setNotifySettings replaces the kinds turned off; pauseDays, when given, pauses for that
// many days from now (0 resumes), and otherwise the pause stays as it is.
// Send me a test (SB-2602): the fixed test text through the cell's real channel.
export function testNotification(f: Fetch = fetch): Promise<{ sent: boolean; via: string }> {
  return call("/v1/me/notifications/test", post({}), f);
}

export async function setNotifySettings(off: string[], f: Fetch = fetch, pauseDays?: number, digest?: "daily" | "weekly", prep?: boolean): Promise<NotifyPrefs> {
  const body: Record<string, unknown> = pauseDays === undefined ? { off } : { off, pause_days: pauseDays };
  if (digest) body.digest = digest;
  if (prep !== undefined) body.prep = prep;
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

// A space's suggested questions, for its members (SB-3307). A failure is none.
export async function suggestedQuestions(space: string, f: Fetch = fetch): Promise<string[]> {
  try {
    const d = await call<{ questions?: unknown }>(`/v1/spaces/${encodeURIComponent(space)}/suggested`, {}, f);
    return Array.isArray(d.questions) ? d.questions.filter((q): q is string => typeof q === "string") : [];
  } catch {
    return [];
  }
}

export function setSuggestedQuestions(space: string, questions: string[], f: Fetch = fetch): Promise<{ questions: string[] }> {
  return call(`/v1/spaces/${encodeURIComponent(space)}/suggested`, { method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ questions }) }, f);
}

// Make a saved question a routine, or stop it (SB-4302): every is daily, weekly or off.
export function setRoutine(id: string, every: string, day: string | undefined, f: Fetch = fetch): Promise<SavedQuestion> {
  return call(
    `/v1/saved/${encodeURIComponent(id)}/routine`,
    { method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify(day ? { every, day } : { every }) },
    f,
  );
}

// One inbox (SB-4301): what waits on me in every space, checked as me when listed.
export interface InboxItem {
  kind: string; // draft, question, request, policy, approval, action, fix, research, routine
  id: string;
  space?: string;
  title: string;
  at?: string;
  due?: string;
  overdue?: boolean;
  why?: string;
  link: string;
}

export interface Inbox {
  needs_you: InboxItem[];
  failed: InboxItem[];
  done: InboxItem[];
  counts: { needs_you: number; failed: number; done: number };
}

export async function myInbox(f: Fetch = fetch): Promise<Inbox> {
  const d = await call<Partial<Inbox>>("/v1/me/inbox", {}, f);
  const needs_you = d.needs_you ?? [];
  const failed = d.failed ?? [];
  const done = d.done ?? [];
  return { needs_you, failed, done, counts: { needs_you: needs_you.length, failed: failed.length, done: done.length } };
}

// Memory I control (SB-4303): lines I wrote about myself, sent with my own asks only.
export interface MemoryLine {
  id: string;
  text: string;
  at: string;
}

export async function myMemory(f: Fetch = fetch): Promise<{ on: boolean; lines: MemoryLine[] }> {
  const d = await call<{ on?: boolean; lines?: MemoryLine[] }>("/v1/me/memory", {}, f);
  return { on: d.on !== false, lines: d.lines ?? [] };
}

export function remember(text: string, f: Fetch = fetch): Promise<MemoryLine> {
  return call("/v1/me/memory", post({ text }), f);
}

export function forgetMemory(id: string | undefined, f: Fetch = fetch): Promise<{ removed: number }> {
  return call(id ? `/v1/me/memory/${encodeURIComponent(id)}` : "/v1/me/memory", { method: "DELETE" }, f);
}

// What agents did as me (SB-4305): runs made as me, what they read checked as me now.
export interface AgentRun {
  kind: string;
  agent: string;
  space: string;
  at: string;
  title: string;
  read: { cite: string; title: string }[];
  hidden: number;
  drafted: string;
  link: string;
}

export async function agentRuns(f: Fetch = fetch): Promise<AgentRun[]> {
  const d = await call<{ items?: AgentRun[] }>("/v1/me/agent-runs", {}, f);
  return (d.items ?? []).map((r) => ({ ...r, read: r.read ?? [], hidden: r.hidden ?? 0 }));
}

// Collections (SB-4304): ids only; each reader sees what they can open, the rest counted.
export interface CollectionItem {
  item: string;
  title?: string;
  question?: string;
}

export interface Collection {
  id: string;
  name: string;
  owner: string;
  mine: boolean;
  items: CollectionItem[];
  hidden: number;
  shared?: string[];
  at: string;
  stale?: boolean;
}

export async function collections(f: Fetch = fetch): Promise<Collection[]> {
  const d = await call<{ items?: Collection[] }>("/v1/collections", {}, f);
  return (d.items ?? []).map((c) => ({ ...c, items: c.items ?? [], hidden: c.hidden ?? 0 }));
}

export function makeCollection(name: string, items: string[], shared: string[], f: Fetch = fetch): Promise<Collection> {
  return call("/v1/collections", post({ name, items, shared }), f);
}

export function removeCollection(id: string, f: Fetch = fetch): Promise<{ deleted: boolean }> {
  return call(`/v1/collections/${encodeURIComponent(id)}`, { method: "DELETE" }, f);
}
