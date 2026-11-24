import { act, cleanup, fireEvent, render, screen } from "@testing-library/react";
import axe from "axe-core";
import { afterEach, describe, expect, it } from "vitest";
import { App } from "./App";
import type { AskResponse, Fetch } from "./api";
import { conversationMarkdown } from "./Copy";
import { StewardView } from "./Steward";

afterEach(cleanup);

const json = (v: unknown, status = 200) => new Response(JSON.stringify(v), { status });

const answer: AskResponse = {
  trace: "tr_1",
  release: "r",
  answer: {
    claims: [
      { text: "Refunds over 500 need approval.", cites: [{ cite: "docs:r@2", title: "Refunds", span: [0, 9], flags: [], owner: { email: "ann@acme.example" } }] },
      { text: "Ask finance.", cites: [], uncited: true },
    ],
  },
  refused: 1,
  notice: null,
  conversation: "cv_1",
  people: [{ email: "ann@acme.example", cites: 1 }],
};

function cell(calls: string[]): Fetch {
  return async (u, init) => {
    calls.push(`${init?.method ?? "GET"} ${u} ${init?.body ?? ""}`);
    if (u === "/v1/notice") return json({});
    if (u === "/v1/spaces") return json({ spaces: [{ name: "ops", label: "internal" }] });
    if (u.startsWith("/v1/changes") || u.startsWith("/v1/start")) return json({ items: [] });
    if (u.startsWith("/v1/spaces/ops/suggested")) return json({ questions: ["When is the zebra cutoff?"] });
    if (u.startsWith("/v1/search?q=&owner=")) return json({ trace: "tr_s", release: "r", results: [{ cite: "docs:r@2", title: "Refunds", snippet: "x", flags: [] }], refused: 0 });
    if (u.startsWith("/v1/search")) return json({ trace: "tr_s", release: "r", results: [], refused: 0 });
    return json(answer);
  };
}

describe("know where to look (SB-3305 to SB-3307)", () => {
  it("makes the whole conversation as Markdown, by the Copy rules", () => {
    const md = conversationMarkdown([
      { question: "one?", answer },
      { question: "pending?" },
      { question: "two?", answer },
    ]);
    expect(md.match(/\*\*one\?\*\*|\*\*two\?\*\*/g)).toHaveLength(2);
    expect(md).not.toContain("pending");
    expect(md).toContain("**Uncited:** Ask finance.");
    expect(md).toContain("1 source was not shown");
  });

  it("opens an owner's documents from Who to ask", async () => {
    const calls: string[] = [];
    render(<App fetcher={cell(calls)} />);
    fireEvent.change(screen.getByLabelText("Ask a question"), { target: { value: "refunds?" } });
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Ask" }));
    });
    expect(screen.getByRole("button", { name: "Download conversation" })).toBeTruthy();
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Documents of ann@acme.example" }));
    });
    expect(calls.some((c) => c.startsWith("GET /v1/search?q=&owner=ann%40acme.example"))).toBe(true);
    expect(screen.getByText("Documents owned by ann@acme.example")).toBeTruthy();
    expect(screen.getAllByText("Refunds").length).toBeGreaterThan(0);
  });

  it("shows a space's suggested questions in an empty ask box, and the stewards' editor", async () => {
    const calls: string[] = [];
    const { container } = render(<App fetcher={cell(calls)} />);
    await act(async () => {});
    expect(screen.queryByRole("group", { name: "Suggested questions" })).toBeNull(); // company-wide: none
    await act(async () => {
      fireEvent.change(screen.getByLabelText("Ask in"), { target: { value: "ops" } });
    });
    fireEvent.click(screen.getByRole("button", { name: "When is the zebra cutoff?" }));
    expect((screen.getByLabelText("Ask a question") as HTMLTextAreaElement).value).toBe("When is the zebra cutoff?");
    expect(screen.queryByRole("group", { name: "Suggested questions" })).toBeNull();
    const r = await axe.run(container, { rules: { "color-contrast": { enabled: false } } });
    expect(r.violations.map((v) => v.id)).toEqual([]);
    cleanup();
    render(<StewardView space="ops" fetcher={cell(calls)} />);
    await act(async () => {});
    const box = screen.getByLabelText(/One question per line/) as HTMLTextAreaElement;
    expect(box.value).toBe("When is the zebra cutoff?");
    fireEvent.change(box, { target: { value: "A?\nB?" } });
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Save" }));
    });
    expect(calls).toContain('PUT /v1/spaces/ops/suggested {"questions":["A?","B?"]}');
  });
});
