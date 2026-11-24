// Recent questions (sb-ask(3), The web UI; SB-2405): the person's last questions, kept in
// this browser only and never sent to the cell for this. Storage can be off; then none.
const key = "sb.recent";
export const MaxRecent = 20;

export function recentQuestions(): string[] {
  try {
    const v = JSON.parse(window.localStorage.getItem(key) ?? "[]") as unknown;
    return Array.isArray(v) ? v.filter((x): x is string => typeof x === "string").slice(0, MaxRecent) : [];
  } catch {
    return [];
  }
}

export function rememberQuestion(q: string): string[] {
  const next = [q, ...recentQuestions().filter((x) => x !== q)].slice(0, MaxRecent);
  try {
    window.localStorage.setItem(key, JSON.stringify(next));
  } catch {
    // storage off: nothing is kept
  }
  return next;
}

export function forgetQuestions(): void {
  try {
    window.localStorage.removeItem(key);
  } catch {
    // nothing to clear
  }
}

// Verified only (SB-3602): the person's choice for asks and for Documents, in this browser.
const verifiedKey = (where: string) => `sb.verified.${where}`;

export function verifiedOnly(where: "ask" | "documents"): boolean {
  try {
    return window.localStorage.getItem(verifiedKey(where)) === "1";
  } catch {
    return false;
  }
}

export function setVerifiedOnly(where: "ask" | "documents", on: boolean): void {
  try {
    if (on) window.localStorage.setItem(verifiedKey(where), "1");
    else window.localStorage.removeItem(verifiedKey(where));
  } catch {
    // storage off: the choice lasts this page only
  }
}

// Short answers (SB-3303): the person's choice, kept in this browser only.
const shortKey = "sb.short";

export function shortAnswers(): boolean {
  try {
    return window.localStorage.getItem(shortKey) === "1";
  } catch {
    return false;
  }
}

export function setShortAnswers(on: boolean): void {
  try {
    if (on) window.localStorage.setItem(shortKey, "1");
    else window.localStorage.removeItem(shortKey);
  } catch {
    // storage off: the choice lasts this page only
  }
}
