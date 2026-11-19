import { act, cleanup, fireEvent, render, screen } from "@testing-library/react";
import axe from "axe-core";
import { afterEach, describe, expect, it } from "vitest";
import { DraftPanel } from "./Draft";
import type { Fetch } from "./api";

afterEach(cleanup);

const draft = {
  trace: "tr_d",
  release: "rel_x",
  context: "tickets:OPS-7",
  refused: 0,
  notice: null,
  conversation: "",
  answer: {
    claims: [
      { text: "Thanks for flagging this.", cites: [], uncited: true },
      { text: "Refunds up to 500 need no approval.", cites: [{ cite: "docs:refunds@2", span: [0, 10] as [number, number], flags: [] }] },
    ],
  },
};

describe("draft panel", () => {
  it("drafts, copies and records the outcome, and never sends", async () => {
    const calls: { url: string; body: unknown }[] = [];
    const fetcher: Fetch = async (url, init) => {
      calls.push({ url, body: JSON.parse(String(init?.body)) });
      if (url === "/v1/drafts") return new Response(JSON.stringify(draft), { status: 200 });
      return new Response(JSON.stringify({ trace: "tr_d", outcome: "used" }), { status: 200 });
    };
    const copied: string[] = [];
    const { container } = render(<DraftPanel fetcher={fetcher} copy={async (t) => void copied.push(t)} />);
    fireEvent.change(screen.getByLabelText(/Ticket or thread/), { target: { value: "tickets:OPS-7" } });
    fireEvent.change(screen.getByLabelText(/What should the reply say/), { target: { value: "Reply with our refund rule" } });
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Draft" }));
    });
    expect(screen.getByText(/Nothing is sent until you send it yourself/)).toBeTruthy();
    expect(screen.getByRole("link", { name: /docs:refunds@2/ })).toBeTruthy();
    expect(container.textContent).toContain("Uncited");
    expect(await axeClean(container)).toEqual([]);
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Copy draft" }));
    });
    expect(copied).toEqual(["Thanks for flagging this. Refunds up to 500 need no approval."]);
    expect(calls.map((c) => c.url)).toEqual(["/v1/drafts", "/v1/drafts/tr_d/outcome"]);
    expect(calls[1]!.body).toEqual({ outcome: "used" });
    expect(calls.some((c) => c.url.startsWith("/v1/actions"))).toBe(false);
    expect(screen.getByRole("status").textContent).toMatch(/send it there/);
  });

  it("shows a refusal without details", async () => {
    const fetcher: Fetch = async () =>
      new Response(JSON.stringify({ error: { code: "EPERM", message: "You can't see that trace.", trace: "tr_x", remedy: "Ask the owner." } }), { status: 403 });
    render(<DraftPanel fetcher={fetcher} copy={async () => undefined} />);
    fireEvent.change(screen.getByLabelText(/Ticket or thread/), { target: { value: "tickets:FIN-2" } });
    fireEvent.change(screen.getByLabelText(/What should the reply say/), { target: { value: "x" } });
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Draft" }));
    });
    expect(screen.getByRole("status").textContent).toContain("Ask the owner.");
    expect(screen.queryByRole("button", { name: "Copy draft" })).toBeNull();
  });
});

async function axeClean(el: HTMLElement) {
  const r = await axe.run(el, { rules: { "color-contrast": { enabled: false } } });
  return r.violations.map((v) => v.id);
}
