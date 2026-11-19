// sb-actions(3): read, approve and reject one action.
import type { ApiError, Fetch } from "./api";
import { AskError } from "./api";

export interface Change {
  field: string;
  before: unknown;
  after: unknown;
}

export interface ActionView {
  id: string;
  state: string;
  workflow: string;
  target: string;
  payload: unknown;
  payload_sha256: string;
  requester: string;
  approvers: string;
  expires_at?: string;
  policy_reason?: string;
  approval?: string;
  approved_by?: string;
  receipt?: string;
  reason?: string;
  diff: Change[] | null;
}

const unavailable = (trace = ""): ApiError => ({ code: "EUNAVAIL", message: "Actions are unavailable right now.", trace, remedy: "Try again in a minute." });

async function call<T>(f: Fetch, url: string, init?: RequestInit): Promise<T> {
  let res: Response;
  try {
    res = await f(url, { credentials: "same-origin", ...init, headers: { "Content-Type": "application/json", ...(init?.headers ?? {}) } });
  } catch {
    throw new AskError(unavailable());
  }
  const data: unknown = await res.json().catch(() => null);
  if (!res.ok) {
    throw new AskError((data as { error?: ApiError } | null)?.error ?? unavailable());
  }
  return data as T;
}

export const getAction = (id: string, f: Fetch = fetch) => call<ActionView>(f, `/v1/actions/${encodeURIComponent(id)}`);

// approve echoes the hash the approver was shown, never a freshly fetched one (LAW-15).
export const approveAction = (id: string, shownHash: string, note: string, f: Fetch = fetch) =>
  call<ActionView>(f, `/v1/actions/${encodeURIComponent(id)}/approve`, { method: "POST", body: JSON.stringify({ payload_sha256: shownHash, note }) });

export const rejectAction = (id: string, note: string, f: Fetch = fetch) =>
  call<ActionView>(f, `/v1/actions/${encodeURIComponent(id)}/reject`, { method: "POST", body: JSON.stringify({ note }) });
