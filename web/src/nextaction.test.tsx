import { act, cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import { NextAction } from "./NextAction";
import type { Fetch } from "./api";

afterEach(cleanup);

describe("draft the next action (SB-5304)", () => {
  it("drafts a message, sends nothing until Send, then posts it in the channel", async () => {
    const calls: [string, string][] = [];
    const f: Fetch = async (u, init) => {
      calls.push([u, String(init?.body ?? "")]);
      if (u === "/v1/actions/draft") return new Response(JSON.stringify({ kind: "message", text: "- The dock opens at seven.", cites: ["docs:dock@1"] }), { status: 200 });
      return new Response(JSON.stringify({ channel: "#ops", id: "p1" }), { status: 200 });
    };
    render(<NextAction trace="tr_1" space="ops" fetcher={f} />);
    fireEvent.click(screen.getByRole("button", { name: "Draft a message" }));
    await act(async () => {});
    expect(calls).toHaveLength(1);
    expect((screen.getByLabelText("Text") as HTMLTextAreaElement).value).toBe("- The dock opens at seven.");
    fireEvent.change(screen.getByLabelText("Channel"), { target: { value: "#ops" } });
    fireEvent.click(screen.getByRole("button", { name: "Send" }));
    await act(async () => {});
    expect(calls[1]![0]).toBe("/v1/spaces/ops/messages");
    expect(JSON.parse(calls[1]![1])).toEqual({ channel: "#ops", text: "- The dock opens at seven.", cites: ["docs:dock@1"] });
    expect(screen.getByRole("status").textContent).toContain("marked as sent by you");
  });

  it("sends a ticket comment through the gated workflow", async () => {
    const calls: [string, string][] = [];
    const f: Fetch = async (u, init) => {
      calls.push([u, String(init?.body ?? "")]);
      if (u === "/v1/actions/draft") return new Response(JSON.stringify({ kind: "ticket", text: "Root cause", cites: [] }), { status: 200 });
      return new Response(JSON.stringify({ id: "tc_1", state: "PendingApproval" }), { status: 200 });
    };
    render(<NextAction trace="tr_1" fetcher={f} />);
    fireEvent.click(screen.getByRole("button", { name: "Draft a ticket comment" }));
    await act(async () => {});
    fireEvent.change(screen.getByLabelText("Ticket"), { target: { value: "OPS-1182" } });
    fireEvent.click(screen.getByRole("button", { name: "Send" }));
    await act(async () => {});
    expect(JSON.parse(calls[1]![1])).toEqual({ workflow: "ticket-update", target: "tickets:OPS-1182", payload: { comment: "Root cause" } });
    expect(screen.getByRole("status").textContent).toContain("PendingApproval");
  });
});
