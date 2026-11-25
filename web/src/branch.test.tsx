import { act, cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import { App } from "./App";
import type { Fetch } from "./api";

afterEach(cleanup);

const answer = (conversation: string, text: string) => ({
  trace: "tr_1",
  release: "rel_test",
  answer: { claims: [{ text, cites: [] , uncited: true }] },
  refused: 0,
  notice: null,
  conversation,
  agent: "ask",
  dropped: 0,
  earlier_dropped: 0,
  people: [],
});

describe("branch a conversation (SB-5206)", () => {
  it("starts a branch from an earlier question and goes back to the first", async () => {
    const calls: string[] = [];
    let n = 0;
    const f: Fetch = async (u, init) => {
      calls.push(`${init?.method ?? "GET"} ${u} ${init?.body ?? ""}`);
      if (u.startsWith("/v1/conversations/cv_1/branch")) return new Response(JSON.stringify({ conversation: "cv_2", from: "cv_1", at: 1 }), { status: 200 });
      if (u.startsWith("/v1/ask")) {
        n++;
        return new Response(JSON.stringify(answer(n < 3 ? "cv_1" : "cv_2", `answer ${n}`)), { status: 200 });
      }
      return new Response(JSON.stringify({ items: [], results: [], spaces: [] }), { status: 200 });
    };
    render(<App fetcher={f} />);
    await act(async () => {});
    const ask = async (q: string) => {
      fireEvent.change(screen.getByLabelText("Ask a question"), { target: { value: q } });
      fireEvent.click(screen.getByRole("button", { name: "Ask" }));
      await act(async () => {});
    };
    await ask("first question");
    await ask("second question");
    fireEvent.click(screen.getAllByRole("button", { name: "Ask from here" })[0]!);
    await act(async () => {});
    expect(calls.some((c) => c.startsWith('POST /v1/conversations/cv_1/branch {"at":1}'))).toBe(true);
    expect(screen.queryByRole("heading", { name: "second question" })).toBeNull();
    expect(screen.getByRole("region", { name: "Other branches" }).textContent).toContain("Back to: second question");
    await ask("another direction");
    expect(calls.filter((c) => c.startsWith("POST /v1/ask")).pop()).toContain('"conversation":"cv_2"');
    fireEvent.click(screen.getByRole("button", { name: "Back to: second question" }));
    expect(screen.getByRole("heading", { name: "second question" })).toBeTruthy();
  });
});
