import { act, cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import axe from "axe-core";
import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { App } from "./App";
import { AdminTokens } from "./AdminTokens";
import type { AskResponse, Fetch } from "./api";

beforeEach(() => window.localStorage.clear());
afterEach(cleanup);

async function violations(container: HTMLElement) {
  const r = await axe.run(container, { rules: { "color-contrast": { enabled: false } } });
  return r.violations.map((v) => v.id);
}

const json = (x: unknown, status = 200) => new Response(JSON.stringify(x), { status });

const answer: AskResponse = {
  trace: "tr_1",
  release: "r",
  answer: { claims: [{ text: "The rate is 4%.", cites: [{ cite: "file:quote.txt", title: "Your file: quote.txt", span: [0, 10], flags: [] }] }] },
  refused: 0,
  notice: null,
  conversation: "cv_1",
};

// SB-3802
describe("ask about a file", () => {
  it("sends one file with the next ask only, and refuses one over 2 MB", async () => {
    const bodies: string[] = [];
    const fetcher: Fetch = async (u, init) => {
      if (u === "/v1/notice") return json({});
      if (u === "/v1/spaces") return json({ spaces: [] });
      if (u.startsWith("/v1/changes")) return json({ items: [] });
      if (u.startsWith("/v1/search")) return json({ trace: "tr_s", release: "r", results: [], refused: 0 });
      bodies.push(String(init?.body));
      return json(answer);
    };
    const { container } = render(<App fetcher={fetcher} />);
    const input = screen.getByLabelText("Attach a file") as HTMLInputElement;
    const big = new File([new Uint8Array((2 << 20) + 1)], "big.txt", { type: "text/plain" });
    await act(async () => {
      fireEvent.change(input, { target: { files: [big] } });
    });
    expect(screen.getByRole("alert").textContent).toBe("big.txt is larger than 2 MB.");
    const small = new File(["the rate is 4%"], "quote.txt", { type: "text/plain" });
    await act(async () => {
      fireEvent.change(input, { target: { files: [small] } });
    });
    await waitFor(() => expect(screen.getByText("Asking about quote.txt")).toBeTruthy());
    expect(await violations(container)).toEqual([]);
    fireEvent.change(screen.getByLabelText("Ask a question"), { target: { value: "what rate?" } });
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Ask" }));
    });
    const sent = JSON.parse(bodies[0]!);
    expect(sent.file).toEqual({ name: "quote.txt", data: btoa("the rate is 4%") });
    expect(screen.queryByText("Asking about quote.txt")).toBeNull();
    fireEvent.change(screen.getByLabelText("Ask a question"), { target: { value: "and again?" } });
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Ask" }));
    });
    expect(JSON.parse(bodies[1]!).file).toBeUndefined();
  });
});

// SB-3803
describe("every token in one place", () => {
  it("lists the tenant's tokens and revokes one with a reason", async () => {
    const calls: string[] = [];
    let items = [
      { id: "tok_a", owner: "usr_ann", name: "nightly", created: "2027-01-01T00:00:00Z", expires: "2027-03-01T00:00:00Z" },
      { id: "tok_b", owner: "usr_bob", name: "report", created: "2026-12-01T00:00:00Z", last_used: "2027-01-10", expires: "2027-02-01T00:00:00Z" },
    ];
    const fetcher: Fetch = async (u, init) => {
      calls.push(`${init?.method ?? "GET"} ${u} ${init?.body ?? ""}`.trim());
      if (u === "/v1/admin/tokens") return json({ items });
      items = items.filter((x) => x.id !== "tok_a");
      return json({ id: "tok_a", revoked: true });
    };
    const { container } = render(<AdminTokens fetcher={fetcher} />);
    await waitFor(() => expect(screen.getByText("nightly")).toBeTruthy());
    expect(container.textContent).toContain("never used");
    expect(container.textContent).toContain("last used 2027-01-10");
    expect(await violations(container)).toEqual([]);
    fireEvent.click(screen.getByRole("button", { name: "Revoke nightly" }));
    await waitFor(() => expect(screen.getByRole("status").textContent).toBe("A reason is required; it goes into the audit log."));
    fireEvent.change(screen.getByLabelText("Reason (required)"), { target: { value: "left the team" } });
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Revoke nightly" }));
    });
    await waitFor(() => expect(screen.getByRole("status").textContent).toBe("Revoked nightly."));
    expect(calls).toContain('DELETE /v1/admin/tokens/tok_a {"reason":"left the team"}');
    expect(screen.queryByText("nightly")).toBeNull();
  });
});
