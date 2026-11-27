import { act, cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import { ContextView, MemoryUsed } from "./Steer";
import { ask, type Fetch } from "./api";
import { conversationMarkdown } from "./Copy";

afterEach(cleanup);

const panel = {
  trace: "tr_1", agent: "ask", release: "r", sources: [{ cite: "docs:dock", title: "Dock hours" }], sources_hidden: 1, departments: [],
  memory: [], documents: [], kept: [], dropped: [], documents_hidden: 0, earlier_kept: 2, earlier_dropped: 0, skipped: 0, held: 0,
};

describe("see and steer the context (SB-5501 to SB-5504)", () => {
  it("shows the context, keeps and drops for the conversation, and excludes for me", async () => {
    const calls: [string, string][] = [];
    const f: Fetch = async (u, init) => {
      calls.push([u, String(init?.body ?? init?.method ?? "")]);
      if (u === "/v1/traces/tr_1/context") return new Response(JSON.stringify(panel), { status: 200 });
      return new Response(JSON.stringify({ items: [], hidden: 0 }), { status: 200 });
    };
    const steered: string[] = [];
    render(<ContextView trace="tr_1" fetcher={f} onSteer={(d, k) => steered.push(k + " " + d)} />);
    fireEvent.click(screen.getByRole("button", { name: "Context" }));
    await act(async () => {});
    expect(screen.getByText(/Dock hours/)).toBeTruthy();
    expect(screen.getByText("1 source you can't open now")).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "Drop" }));
    fireEvent.click(screen.getByRole("button", { name: "Keep" }));
    expect(steered).toEqual(["drop docs:dock", "keep docs:dock"]);
    fireEvent.click(screen.getByRole("button", { name: "Don't use this for me" }));
    await act(async () => {});
    expect(calls[1]).toEqual(["/v1/me/exclusions", JSON.stringify({ document: "docs:dock" })]);
  });

  it("sends Keep and Drop with the ask, and puts them in the download", async () => {
    let body = "";
    const f: Fetch = async (_u, init) => {
      body = String(init?.body ?? "");
      return new Response(JSON.stringify({ trace: "tr_2", release: "r", answer: { claims: [] }, refused: 0, notice: null, conversation: "cv_1" }), { status: 200 });
    };
    await ask("q", "cv_1", f, undefined, undefined, undefined, undefined, undefined, undefined, undefined, undefined, undefined, undefined, undefined, { keep: ["docs:a"], drop: ["docs:b"] });
    expect(JSON.parse(body)).toMatchObject({ keep: ["docs:a"], drop: ["docs:b"] });
    expect(conversationMarkdown([], { keep: ["docs:a"], drop: [] })).toContain("Kept: docs:a");
  });

  it("shows the memory line an answer used, with Forget this", async () => {
    const calls: string[] = [];
    const f: Fetch = async (u, init) => {
      calls.push((init?.method ?? "GET") + " " + u);
      if (u === "/v1/me/memory") return new Response(JSON.stringify({ on: true, lines: [{ id: "mem_1", text: "I run the night shift.", at: "" }, { id: "mem_2", text: "other", at: "" }] }), { status: 200 });
      return new Response(JSON.stringify({ removed: 1 }), { status: 200 });
    };
    render(<MemoryUsed ids={["mem_1"]} fetcher={f} />);
    await act(async () => {});
    expect(screen.getByText(/night shift/)).toBeTruthy();
    expect(screen.queryByText(/other/)).toBeNull();
    fireEvent.click(screen.getByRole("button", { name: "Forget this" }));
    await act(async () => {});
    expect(calls).toContain("DELETE /v1/me/memory/mem_1");
  });
});
