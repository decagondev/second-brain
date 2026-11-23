import { act, cleanup, fireEvent, render, screen } from "@testing-library/react";
import axe from "axe-core";
import { afterEach, describe, expect, it } from "vitest";
import { App } from "./App";
import { AnswerView } from "./Answer";
import { answerMarkdown, answerText } from "./Copy";
import type { AskResponse, Claim, Fetch } from "./api";
import { setLang } from "./i18n";
import css from "./styles.css?raw";
import type { Lang } from "./i18n";

afterEach(() => {
  cleanup();
  setLang("en");
});

async function violations(container: HTMLElement) {
  const r = await axe.run(container, { rules: { "color-contrast": { enabled: false } } });
  return r.violations.map((v) => `${v.id}: ${v.nodes.map((n) => n.html).join(" ")}`);
}

const refund = { cite: "docs:4f2a@17", title: "Refund policy", span: [0, 9] as [number, number], url: "https://drive.example/refund?v=17", flags: [], age_days: 40,
  owner: { email: "ann@acme-logistics.example", title: "Support lead" } };
const ticket = { cite: "tickets:LIN-7@2", span: [0, 5] as [number, number], flags: [] };
const claims: Claim[] = [
  { text: "Refunds over 500 need a team lead's approval.", cites: [refund] },
  { text: "Ask finance for exceptions.", cites: [], uncited: true },
  { text: "It changed in March.", cites: [ticket, refund] },
];
const answer: AskResponse = { trace: "tr_secret", release: "rel_x", answer: { claims }, refused: 2, notice: null, conversation: "cv_1" };

// SB-2401
describe("copy an answer", () => {
  it("copies the question, each claim with its cite numbers, and the numbered cites", () => {
    expect(answerText("Who approves refunds?", answer)).toBe(
      [
        "Who approves refunds?",
        "",
        "- Refunds over 500 need a team lead's approval. [1]",
        "- Uncited: Ask finance for exceptions.",
        "- It changed in March. [2] [1]",
        "",
        "Sources",
        "[1] Refund policy, https://drive.example/refund?v=17",
        "[2] tickets:LIN-7@2",
        "",
        "2 sources were not shown because you can't open them.",
        "",
      ].join("\n"),
    );
  });

  it("never drops the uncited mark, and adds nothing the screen doesn't show", () => {
    for (const text of [answerText("q", answer), answerMarkdown("q", answer)]) {
      const lines = text.split("\n").filter((l) => l.includes("Ask finance"));
      expect(lines).toHaveLength(1);
      expect(lines[0]).toMatch(/Uncited/);
      for (const hidden of ["tr_secret", "ann@acme-logistics.example", "rel_x"]) expect(text).not.toContain(hidden);
    }
  });

  it("escapes Markdown so a claim can't become a link, and links only http cites", () => {
    const sneaky: AskResponse = {
      ...answer,
      refused: 0,
      answer: { claims: [{ text: "See [here](javascript:alert(1)) **now**", cites: [{ ...ticket, url: "javascript:alert(1)" }] }] },
    };
    const m = answerMarkdown("q", sneaky);
    expect(m).not.toMatch(/[^\\]\[here\]\(/);
    expect(m).toContain("\\[here\\]\\(javascript:alert\\(1\\)\\) \\*\\*now\\*\\*");
    expect(m).toContain("1. tickets:LIN-7@2");
    expect(answerMarkdown("q", answer)).toContain("1. [Refund policy](https://drive.example/refund?v=17)");
  });

  it("copies with the button and says so", async () => {
    let got = "";
    Object.defineProperty(navigator, "clipboard", { value: { writeText: async (s: string) => void (got = s) }, configurable: true });
    const fetcher: Fetch = async (u) => {
      if (u === "/v1/spaces") return new Response(JSON.stringify({ spaces: [] }), { status: 200 });
      if (u.startsWith("/v1/changes")) return new Response(JSON.stringify({ items: [] }), { status: 200 });
      if (u.startsWith("/v1/search")) return new Response(JSON.stringify({ results: [], hidden: 0 }), { status: 200 });
      return new Response(JSON.stringify(answer), { status: 200 });
    };
    render(<App fetcher={fetcher} />);
    fireEvent.change(screen.getByLabelText("Ask a question"), { target: { value: "Who approves refunds?" } });
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Ask" }));
    });
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Copy as Markdown" }));
    });
    expect(got).toBe(answerMarkdown("Who approves refunds?", answer));
    expect(screen.getByText("Copied.")).toBeTruthy();
  });
});

// SB-2402
describe("print an answer", () => {
  it("prints the cites as numbered footnotes matching the claims", () => {
    const { container } = render(<AnswerView answer={answer} />);
    const notes = [...container.querySelectorAll(".footnotes li")].map((li) => `${li.getAttribute("value")} ${li.textContent}`);
    expect(notes).toEqual(["1 Refund policy, https://drive.example/refund?v=17, 40 days old", "2 tickets:LIN-7@2"]);
    const marks = [...container.querySelectorAll("li.claim .print-only")].map((s) => s.textContent?.trim());
    expect(marks).toEqual(["[1]", "[2] [1]"]);
    expect(container.querySelector("li.claim.uncited")?.textContent).toContain("Uncited");
  });

  it("hides every control when printed, and shows the footnotes", () => {
    const print = css.slice(css.indexOf("@media print"));
    for (const sel of [".tabs", "form.ask", "button", ".answer-tools", ".feedback", ".turn-tools"]) expect(print).toContain(sel);
    expect(print).toMatch(/\.print-only \{ display: inline; \}/);
    expect(css).toMatch(/^\.print-only \{ display: none; \}/m);
  });
});

// SB-2403
describe("keyboard first", () => {
  function app() {
    const asked: string[] = [];
    const fetcher: Fetch = async (u, init) => {
      if (u === "/v1/spaces") return new Response(JSON.stringify({ spaces: [] }), { status: 200 });
      if (u.startsWith("/v1/changes")) return new Response(JSON.stringify({ items: [] }), { status: 200 });
      if (u.startsWith("/v1/search")) return new Response(JSON.stringify({ results: [], hidden: 0 }), { status: 200 });
      if (u === "/v1/tokens") return new Response(JSON.stringify({ items: [] }), { status: 200 });
      asked.push(String(init?.body));
      return new Response(JSON.stringify(answer), { status: 200 });
    };
    return { asked, ...render(<App fetcher={fetcher} />) };
  }

  it("goes to the ask box with /, asks with Ctrl+Enter, and closes a panel with Esc", async () => {
    const { asked } = app();
    const box = screen.getByLabelText("Ask a question") as HTMLTextAreaElement;
    fireEvent.click(screen.getByRole("tab", { name: "Saved" }));
    fireEvent.keyDown(document.body, { key: "/" });
    expect(document.activeElement).toBe(box);
    expect(screen.getByRole("tab", { name: "Answer" }).getAttribute("aria-selected")).toBe("true");
    // "/" typed in a field is just a character
    const slash = fireEvent.keyDown(box, { key: "/" });
    expect(slash).toBe(true);
    fireEvent.change(box, { target: { value: "Who approves refunds?" } });
    await act(async () => {
      fireEvent.keyDown(box, { key: "Enter", ctrlKey: true });
    });
    expect(asked.some((b) => b.includes("Who approves refunds?"))).toBe(true);
    const tokens = screen.getByRole("button", { name: "API tokens" });
    await act(async () => {
      fireEvent.click(tokens);
    });
    expect(tokens.getAttribute("aria-expanded")).toBe("true");
    (screen.getByLabelText("Token name") as HTMLInputElement).focus();
    fireEvent.keyDown(document.activeElement!, { key: "Escape" });
    expect(tokens.getAttribute("aria-expanded")).toBe("false");
    expect(document.activeElement).toBe(tokens);
  });

  it("starts with a skip link to the conversation, and passes axe in each language", async () => {
    for (const l of ["en", "de", "fr"] as Lang[]) {
      setLang(l);
      const { container } = app();
      const first = container.querySelector("main a");
      expect(first?.getAttribute("href")).toBe("#conversation");
      expect(container.querySelector("#conversation")).toBeTruthy();
      expect(await violations(container)).toEqual([]);
      cleanup();
    }
  });
});
