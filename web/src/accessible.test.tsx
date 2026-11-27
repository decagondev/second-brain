import { act, cleanup, fireEvent, render, screen } from "@testing-library/react";
import axe from "axe-core";
import { afterEach, describe, expect, it } from "vitest";
import { AnswerView } from "./Answer";
import { App } from "./App";
import type { AskResponse, Fetch } from "./api";

// Usable by everyone (sb-ask(3), The web UI, Accessibility; SB-6107): WCAG 2.2 A and AA
// rules axe can check in jsdom over every tab, the ask, cite, review and inbox paths with a
// keyboard alone, and words, not colour alone, for labels, stale cites and notices.

afterEach(cleanup);

async function violations(el: Element): Promise<string[]> {
  const r = await axe.run(el, {
    runOnly: { type: "tag", values: ["wcag2a", "wcag2aa", "wcag21a", "wcag21aa", "wcag22aa"] },
    rules: { "color-contrast": { enabled: false } }, // jsdom computes no colours
  });
  return r.violations.filter((v) => v.impact === "serious" || v.impact === "critical").map((v) => `${v.id}: ${v.nodes.map((n) => n.html).join(" ")}`);
}

const answer: AskResponse = {
  trace: "tr_1",
  release: "rel_x",
  refused: 1,
  notice: null,
  label: "confidential",
  notice_stale: { stale: 2, of: 3, oldest_days: 400 },
  answer: {
    claims: [
      {
        text: "Refunds over 500 need a lead.",
        cites: [{ cite: "docs:refunds@3", title: "Refund policy", span: [0, 30], flags: [], stale: true, age_days: 400, label: "confidential", read_by: "vision", page: 2 }],
      },
    ],
  },
} as unknown as AskResponse;

// every is a fetcher that answers every read with an empty list, and an ask with the answer.
const every: Fetch = async (url) => {
  if (url === "/v1/ask" || url.startsWith("/v1/ask?")) return new Response(JSON.stringify(answer), { status: 200 });
  if (url === "/v1/spaces") return new Response(JSON.stringify({ spaces: [] }), { status: 200 });
  if (url.startsWith("/v1/search")) return new Response(JSON.stringify({ trace: "tr_s", release: "rel_x", results: [], refused: 0 }), { status: 200 });
  return new Response(JSON.stringify({ items: [] }), { status: 200 });
};

describe("every tab", () => {
  it("has no serious or critical violation, and the tabs work from the keyboard", async () => {
    const { container } = render(<App fetcher={every} />);
    await act(async () => {});
    const tabs = screen.getAllByRole("tab");
    expect(tabs.length).toBeGreaterThan(3);
    for (let i = 0; i < tabs.length; i++) {
      await act(async () => {
        fireEvent.click(tabs[i]!);
      });
      expect(await violations(container), `tab ${tabs[i]!.textContent}`).toEqual([]);
    }
    // Arrow keys move between tabs, as a tab list should.
    tabs[0]!.focus();
    await act(async () => {
      fireEvent.click(tabs[0]!);
      fireEvent.keyDown(tabs[0]!, { key: "ArrowRight" });
    });
    expect(document.activeElement).toBe(screen.getAllByRole("tab")[1]);
  });
});

describe("keyboard only", () => {
  it("asks with the keyboard, and reaches the cites and the passage", async () => {
    render(<App fetcher={every} />);
    await act(async () => {});
    const box = screen.getByLabelText("Ask a question");
    box.focus();
    fireEvent.change(box, { target: { value: "refund limit?" } });
    await act(async () => {
      fireEvent.keyDown(box, { key: "Enter", ctrlKey: true });
    });
    const chip = screen.getAllByRole("link").find((a) => (a.getAttribute("aria-label") ?? "").includes("docs:refunds@3"));
    expect(chip, "a cite is a link the keyboard reaches").toBeTruthy();
    expect(chip!.tabIndex).toBeGreaterThanOrEqual(0);
    const show = screen.getByRole("button", { name: "Show passage" });
    expect(show.tabIndex).toBeGreaterThanOrEqual(0);
  });
});

describe("words, not colour", () => {
  it("names the label, a stale cite, a machine-read cite and a notice in words", async () => {
    const { container } = render(<AnswerView answer={answer} />);
    const text = container.textContent ?? "";
    expect(text).toMatch(/Confidential/);
    expect(text).toMatch(/haven't been reviewed in a while/);
    const chip = container.querySelector("[aria-label*='docs:refunds@3']");
    const name = chip?.getAttribute("aria-label") ?? "";
    expect(name).toMatch(/stale|old/i);
    expect(name).toMatch(/Read by a model/);
    for (const n of Array.from(container.querySelectorAll("[role=note]"))) {
      expect((n.textContent ?? "").trim().length, "a notice says something").toBeGreaterThan(10);
    }
    expect(await violations(container)).toEqual([]);
  });
});
