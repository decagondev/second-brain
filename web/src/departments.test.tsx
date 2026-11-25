import { act, cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import { App } from "./App";
import type { AskResponse, Fetch } from "./api";

afterEach(cleanup);

const json = (v: unknown, status = 200) => new Response(JSON.stringify(v), { status });

const answer: AskResponse = {
  trace: "tr_1", release: "rel_x", refused: 0, notice: null, conversation: "cv_1", answer: { claims: [] },
  departments: [{ space: "returns", agent: "sop-steps", dropped: 0, refused: 0, claims: [{ text: "An RMA takes two days.", cites: [{ cite: "docs:rma@1", title: "RMA procedure", span: [0, 10], flags: [] }] }] }],
  not_consulted: [{ space: "fleet", why: "no_match" }, { why: "not_member", count: 2 }],
  handoff: { space: "it", category: "Access" },
};

// SB-4501, SB-4502
describe("departments work together", () => {
  it("consults each department across a branch, and hands a request off only when sent", async () => {
    const calls: { url: string; body?: unknown }[] = [];
    const fetcher: Fetch = async (u, init) => {
      calls.push({ url: u, body: init?.body ? JSON.parse(String(init.body)) : undefined });
      if (u === "/v1/notice") return new Response("{}", { status: 200 });
      if (u === "/v1/spaces") return json({ spaces: [{ name: "ops", label: "internal", teams: true }, { name: "returns", label: "internal", parent: "ops" }] });
      if (u.startsWith("/v1/search")) return json({ trace: "tr_s", release: "rel_x", refused: 0, results: [] });
      if (u.endsWith("/handoff")) return json({ space: "it", category: "Access", text: "I need access", cites: [{ cite: "docs:rma@1", title: "RMA procedure" }] });
      if (u.endsWith("/requests") && init?.method === "POST") return json({ id: "qs_1", to: "it", category: "Access", due: "2027-01-01" });
      return json(answer);
    };
    render(<App fetcher={fetcher} />);
    await act(async () => undefined);
    expect(screen.queryByLabelText("Ask each department")).toBeNull();
    fireEvent.change(screen.getByLabelText("Ask in"), { target: { value: "ops+teams" } });
    fireEvent.click(screen.getByLabelText("Ask each department"));
    fireEvent.change(screen.getByLabelText("Ask a question"), { target: { value: "I need access to the RMA tool" } });
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Ask" }));
    });
    const asked = calls.find((c) => c.url.startsWith("/v1/ask"))!;
    expect(asked.url).toBe("/v1/ask?space=ops&branch=true");
    expect((asked.body as { consult?: boolean }).consult).toBe(true);
    expect(screen.getByText("returns (sop-steps)")).toBeTruthy();
    expect(screen.getByText("fleet wasn't asked: the question didn't use its terms")).toBeTruthy();
    expect(screen.getByText("2 teams you aren't in weren't asked.")).toBeTruthy();

    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Hand off" }));
    });
    expect(calls.some((c) => c.url.endsWith("/requests") && c.body)).toBe(false);
    fireEvent.click(screen.getByLabelText("RMA procedure"));
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Send" }));
    });
    const sent = calls.find((c) => c.url === "/v1/spaces/it/requests" && c.body);
    expect(sent?.body).toEqual({ category: "Access", text: "I need access", cites: ["docs:rma@1"] });
    expect(screen.getByText("Sent to it.")).toBeTruthy();
  });
});
