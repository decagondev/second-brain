import { act, cleanup, fireEvent, render, screen } from "@testing-library/react";
import axe from "axe-core";
import { afterEach, describe, expect, it } from "vitest";
import { App } from "./App";
import type { Fetch } from "./api";

afterEach(cleanup);

async function violations(container: HTMLElement) {
  const r = await axe.run(container, { rules: { "color-contrast": { enabled: false } } });
  return r.violations.map((v) => `${v.id}: ${v.nodes.map((n) => n.html).join(" ")}`);
}

// SB-2306: a person makes a token, sees it once, and revokes it.
describe("API tokens", () => {
  it("makes, shows once and revokes a token", async () => {
    const calls: string[] = [];
    const fetcher: Fetch = async (u, init) => {
      if (u === "/v1/spaces") return new Response(JSON.stringify({ spaces: [] }), { status: 200 });
      if (u.startsWith("/v1/changes")) return new Response(JSON.stringify({ items: [] }), { status: 200 });
      calls.push(`${init?.method ?? "GET"} ${u} ${init?.body ?? ""}`.trim());
      if (u === "/v1/tokens" && init?.method === "POST")
        return new Response(
          JSON.stringify({ id: "tok_1", name: "nightly", created: "2026-11-23T09:00:00Z", expires: "2026-12-23T09:00:00Z", token: "sbt_" + "a".repeat(43) }),
          { status: 200 },
        );
      if (u === "/v1/tokens") return new Response(JSON.stringify({ items: [] }), { status: 200 });
      return new Response(JSON.stringify({ id: "tok_1", revoked: true }), { status: 200 });
    };
    const { container } = render(<App fetcher={fetcher} />);
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "API tokens" }));
    });
    fireEvent.change(screen.getByLabelText("Token name"), { target: { value: "nightly" } });
    fireEvent.change(screen.getByLabelText("Valid for"), { target: { value: "90" } });
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Make a token" }));
    });
    expect(calls).toContain(`POST /v1/tokens {"name":"nightly","days":90}`);
    expect(screen.getByRole("status").textContent).toContain("sbt_");
    expect(await violations(container)).toEqual([]);
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Revoke nightly" }));
    });
    expect(calls).toContain("DELETE /v1/tokens/tok_1");
    expect(container.textContent).not.toContain("sbt_");
  });
});
