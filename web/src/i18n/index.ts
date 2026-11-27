import { Fragment, createElement } from "react";
import type { ReactNode } from "react";
import { catalog } from "./catalog";

// The UI's languages (sb-tenant(5) language; SB-2301). Every string the UI shows is a key
// of the catalog, in English; t() gives it in the current language. Answers stay in
// whatever language the model writes.
export type Lang = "en" | "de" | "fr" | "es" | "it" | "nl";
export const langs: { id: Lang; name: string }[] = [
  { id: "en", name: "English" },
  { id: "de", name: "Deutsch" },
  { id: "fr", name: "Français" },
  { id: "es", name: "Español" }, // SB-6106
  { id: "it", name: "Italiano" },
  { id: "nl", name: "Nederlands" },
];

const stored = "sb.lang";
let current: Lang = "en";

export function isLang(s: string | null | undefined): s is Lang {
  return s === "en" || s === "de" || s === "fr" || s === "es" || s === "it" || s === "nl";
}

// The language to start in: the person's own choice, kept in their browser, else the
// manifest's default, which the cell writes into the page's lang attribute.
export function initialLang(): Lang {
  try {
    const own = window.localStorage.getItem(stored);
    if (isLang(own)) return own;
  } catch {
    // storage can be off: the default still applies
  }
  const page = typeof document !== "undefined" ? document.documentElement.lang : "";
  return isLang(page) ? page : "en";
}

export function lang(): Lang {
  return current;
}

// setLang switches the language; keep stores the choice in the browser.
export function setLang(l: Lang, keep = false) {
  current = l;
  if (typeof document !== "undefined") document.documentElement.lang = l;
  if (!keep) return;
  try {
    window.localStorage.setItem(stored, l);
  } catch {
    // not kept: the choice lasts this page
  }
}

function fill(s: string, vars?: Record<string, string | number>): string {
  if (!vars) return s;
  return s.replace(/\{(\w+)\}/g, (m, k: string) => (k in vars ? String(vars[k]) : m));
}

// t is a catalog string in the current language, with {name} filled from vars.
export function t(key: string, vars?: Record<string, string | number>): string {
  const e = catalog[key];
  const s = current === "en" || !e ? key : e[current];
  return fill(s, vars);
}

// tn picks the singular or the plural by n, by the language's rule (French counts 0 as
// singular), and fills {n}.
export function tn(n: number, one: string, other: string, vars?: Record<string, string | number>): string {
  const single = current === "fr" ? n < 2 : n === 1;
  return t(single ? one : other, { n, ...vars });
}

// tj is t with elements for some of its {name}s, such as a link inside a sentence.
export function tj(key: string, parts: Record<string, ReactNode>, vars?: Record<string, string | number>): ReactNode {
  const s = t(key, vars);
  const out: ReactNode[] = [];
  let last = 0;
  for (const m of s.matchAll(/\{(\w+)\}/g)) {
    const k = m[1]!;
    if (!(k in parts)) continue;
    out.push(s.slice(last, m.index));
    out.push(createElement(Fragment, { key: `${k}-${m.index}` }, parts[k]));
    last = m.index! + m[0].length;
  }
  out.push(s.slice(last));
  return out;
}

// Messages for the cell's error codes (sb-errors(7)): in English the cell's own message,
// which says the most; in another language the code's.
const codes: Record<string, [string, string]> = {
  EINVAL: ["The request couldn't be read.", "Check what you entered and try again."],
  EAUTH: ["You're signed out.", "Sign in again."],
  EPERM: ["You can't open this.", "Ask the owner for access."],
  ETENANT: ["This belongs to another company.", "Use your own company's Second Brain."],
  ESPACE: ["You aren't a member of this space.", "Ask its stewards for membership, or ask company-wide."],
  ENOTSUP: ["This isn't supported here.", "Choose something this source supports."],
  ENOTIMPL: ["This isn't available in this release.", "Ask your administrator."],
  EHOLD: ["A legal hold covers this.", "Ask a Legal steward."],
  EKILLED: ["This is switched off for now.", "Wait for the operator to switch it back on."],
  ENOAPPROVAL: ["The approval is missing or out of date.", "Ask for a fresh approval."],
  ENORECEIPT: ["The action didn't run.", "Check the target, then propose it again."],
  EUNAVAIL: ["This is unavailable right now.", "Try again in a minute."],
  EBUDGET: ["The budget or a limit has been reached.", "Wait for it to reset, or ask your administrator."],
  ERATE: ["You've asked a lot in the last minute.", "Wait {seconds} seconds, then ask again."],
};

export function errorText(e: { code: string; message: string; remedy?: string; retry_after?: number }): { message: string; remedy: string } {
  if (current === "en" && e.message) return { message: e.message, remedy: e.remedy ?? "" };
  const c = codes[e.code];
  const vars = { seconds: e.retry_after ?? 60 };
  if (!c) return { message: t("Something went wrong."), remedy: t("Try again.") };
  return { message: t(c[0], vars), remedy: t(c[1], vars) };
}

// errorCodes is every code's message, for the catalog test.
export const errorCodes = codes;

// msg marks a catalog key kept for later, such as a map of notices; t() translates it.
export function msg(key: string): string {
  return key;
}
