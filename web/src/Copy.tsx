import { useState } from "react";
import type { AskResponse, Cite } from "./api";
import { citeName, isCited } from "./api";
import { t, tn } from "./i18n";

// The answer's cites, each once, numbered in the order the answer first uses them: the same
// numbers on the copy (SB-2401) and on the printed footnotes (SB-2402).
export function numberCites(answer: AskResponse): { cite: Cite; n: number }[] {
  const out: { cite: Cite; n: number }[] = [];
  for (const c of answer.answer.claims) {
    if (!isCited(c)) continue;
    for (const ci of c.cites) {
      if (!out.some((x) => x.cite.cite === ci.cite)) out.push({ cite: ci, n: out.length + 1 });
    }
  }
  return out;
}

export function numberOf(nums: { cite: Cite; n: number }[], cite: string): number {
  return nums.find((x) => x.cite.cite === cite)?.n ?? 0;
}

function refusedLine(answer: AskResponse): string {
  return answer.refused > 0
    ? tn(answer.refused, "{n} source was not shown because you can't open it.", "{n} sources were not shown because you can't open them.")
    : "";
}

// answerText is the answer as plain text: the question, each claim with its cite numbers
// (an uncited one marked as on screen), then the numbered cites. Nothing the screen doesn't show.
export function answerText(question: string, answer: AskResponse): string {
  const nums = numberCites(answer);
  const lines = [question, ""];
  for (const c of answer.answer.claims) {
    if (!isCited(c)) lines.push(`- ${t("Uncited")}: ${c.text}`);
    else lines.push(`- ${c.text} ${c.cites.map((ci) => `[${numberOf(nums, ci.cite)}]`).join(" ")}`);
  }
  if (nums.length > 0) {
    lines.push("", t("Sources"));
    for (const { cite, n } of nums) lines.push(`[${n}] ${[citeName(cite), cite.url ?? ""].filter((x) => x !== "").join(", ")}`);
  }
  const r = refusedLine(answer);
  if (r) lines.push("", r);
  return lines.join("\n") + "\n";
}

// md escapes text for Markdown, so a claim can't become a link, an image or markup.
export function md(s: string): string {
  return s.replace(/([\\`*_[\]()<>#!|~])/g, "\\$1").replace(/\s+/g, " ");
}

function mdURL(u: string): string {
  return u.replace(/[()\s<>]/g, (c) => encodeURIComponent(c));
}

// answerMarkdown is the same answer as Markdown, each cite a link where it has one.
export function answerMarkdown(question: string, answer: AskResponse): string {
  const nums = numberCites(answer);
  const lines = [`**${md(question)}**`, ""];
  for (const c of answer.answer.claims) {
    if (!isCited(c)) lines.push(`- **${md(t("Uncited"))}:** ${md(c.text)}`);
    else lines.push(`- ${md(c.text)} ${c.cites.map((ci) => `\\[${numberOf(nums, ci.cite)}\\]`).join(" ")}`);
  }
  if (nums.length > 0) {
    lines.push("", `**${md(t("Sources"))}**`, "");
    for (const { cite, n } of nums) {
      const name = md(citeName(cite));
      lines.push(`${n}. ${cite.url && /^https?:\/\//.test(cite.url) ? `[${name}](${mdURL(cite.url)})` : name}`);
    }
  }
  const r = refusedLine(answer);
  if (r) lines.push("", md(r));
  return lines.join("\n") + "\n";
}

// conversationMarkdown is every answered turn, in order, each as Copy as Markdown makes it
// (SB-3305).
export function conversationMarkdown(turns: { question: string; answer?: AskResponse }[]): string {
  return turns
    .filter((x) => x.answer)
    .map((x) => answerMarkdown(x.question, x.answer!))
    .join("\n---\n\n");
}

// DownloadConversation saves the conversation as one Markdown file, made in this browser and
// sent nowhere (SB-3305).
export function DownloadConversation({ turns }: { turns: { question: string; answer?: AskResponse }[] }) {
  if (!turns.some((x) => x.answer)) return null;
  return (
    <button
      type="button"
      onClick={() => {
        const url = URL.createObjectURL(new Blob([conversationMarkdown(turns)], { type: "text/markdown;charset=utf-8" }));
        const a = document.createElement("a");
        a.href = url;
        a.download = "conversation.md";
        document.body.appendChild(a);
        a.click();
        a.remove();
        setTimeout(() => URL.revokeObjectURL(url), 0);
      }}
    >
      {t("Download conversation")}
    </button>
  );
}

// AnswerTools are Copy, Copy as Markdown and Print under an answer (SB-2401, SB-2402).
export function AnswerTools({ question, answer }: { question: string; answer: AskResponse }) {
  const [state, setState] = useState<"" | "copied" | "failed">("");
  async function copy(text: string) {
    try {
      await navigator.clipboard.writeText(text);
      setState("copied");
    } catch {
      setState("failed");
    }
  }
  return (
    <div className="answer-tools">
      <button type="button" onClick={() => copy(answerText(question, answer))}>
        {t("Copy")}
      </button>
      <button type="button" onClick={() => copy(answerMarkdown(question, answer))}>
        {t("Copy as Markdown")}
      </button>
      <button type="button" onClick={() => window.print()}>
        {t("Print")}
      </button>
      {state !== "" && <span role="status">{state === "copied" ? t("Copied.") : t("This browser didn't allow copying.")}</span>}
    </div>
  );
}
