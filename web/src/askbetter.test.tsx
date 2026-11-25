import { act, cleanup, fireEvent, render, screen } from "@testing-library/react";
import axe from "axe-core";
import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { App } from "./App";
import { AnswerView } from "./Answer";
import type { AskResponse, Fetch } from "./api";

beforeEach(() => window.localStorage.clear());
afterEach(cleanup);

async function violations(container: HTMLElement) {
  const r = await axe.run(container, { rules: { "color-contrast": { enabled: false } } });
  return r.violations.map((v) => v.id);
}

const answer: AskResponse = {
  trace: "tr_1",
  release: "r",
  answer: {
    claims: [
      {
        text: "Returns are accepted within 14 or 30 days; raise a PO <b>first</b>.",
        disagree: true,
        cites: [
          { cite: "docs:uk@1", title: "Returns (UK)", span: [0, 40], flags: [] },
          { cite: "docs:eu@1", title: "Returns (EU)", span: [0, 40], flags: [] },
        ],
      },
      { text: "Ask the po box team.", cites: [], uncited: true },
    ],
  },
  refused: 0,
  notice: null,
  conversation: "cv_1",
  terms: [{ term: "PO", means: "purchase order <i>form</i>" }],
  see_also: [{ cite: "docs:exc@2", title: "Refund exceptions", span: [0, 300], age_days: 40 }],
};

describe("ask better (SB-3301 to SB-3304)", () => {
  it("shows See also, the glossary where used, and sources that disagree, as plain text", async () => {
    const { container } = render(<AnswerView answer={answer} onAbout={() => {}} />);
    expect(screen.getByText("Sources disagree:")).toBeTruthy();
    const see = screen.getByRole("region", { name: "See also" });
    expect(see.textContent).toContain("Refund exceptions");
    expect(see.textContent).toContain("40 days old");
    expect(screen.getAllByRole("button", { name: "Show passage" })).toHaveLength(3);
    const tips = screen.getAllByRole("tooltip");
    expect(tips).toHaveLength(1); // "PO" is written with capitals, so "po" isn't it
    expect(tips[0]!.textContent).toBe("purchase order <i>form</i>");
    expect(container.querySelector("i, b")).toBeNull();
    expect(await violations(container)).toEqual([]);
  });

  it("asks for a short answer when the box is ticked, and remembers it", async () => {
    const bodies: string[] = [];
    const fetcher: Fetch = async (u, init) => {
      if (u === "/v1/notice") return new Response("{}", { status: 200 });
      if (u === "/v1/spaces") return new Response(JSON.stringify({ spaces: [] }), { status: 200 });
      if (u.startsWith("/v1/changes")) return new Response(JSON.stringify({ items: [] }), { status: 200 });
      if (u.startsWith("/v1/search")) return new Response(JSON.stringify({ trace: "tr_s", release: "r", results: [], refused: 0 }), { status: 200 });
      bodies.push(String(init?.body));
      return new Response(JSON.stringify(answer), { status: 200 });
    };
    render(<App fetcher={fetcher} />);
    fireEvent.click(screen.getByRole("checkbox", { name: "Short answer" }));
    fireEvent.change(screen.getByLabelText("Ask a question"), { target: { value: "returns?" } });
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Ask" }));
    });
    expect(JSON.parse(bodies[0]!)).toEqual({ question: "returns?", clarify: true, length: "short", language: "en" });
    cleanup();
    render(<App fetcher={fetcher} />);
    expect((screen.getByRole("checkbox", { name: "Short answer" }) as HTMLInputElement).checked).toBe(true);
  });
});
