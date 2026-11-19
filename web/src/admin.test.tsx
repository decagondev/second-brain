import { act, cleanup, fireEvent, render, screen } from "@testing-library/react";
import axe from "axe-core";
import { afterEach, describe, expect, it } from "vitest";
import { AdminConsole } from "./Admin";
import type { Fetch } from "./api";

afterEach(cleanup);

function server() {
  const calls: { method: string; url: string; body?: unknown }[] = [];
  let switches: unknown[] = [];
  const fetcher: Fetch = async (url, init) => {
    const method = init?.method ?? "GET";
    const body = init?.body ? JSON.parse(String(init.body)) : undefined;
    calls.push({ method, url, body });
    if (url === "/v1/admin/manifest") return new Response(JSON.stringify({ manifest: "tenant: acme-logistics\nagents: [ask, draft-reply]\n", commit: "abc" }));
    if (url === "/v1/admin/kill" && method === "GET") return new Response(JSON.stringify({ switches }));
    if (url === "/v1/admin/kill") {
      switches = [{ scope: `${body.scope}:${body.name}`, reason: body.reason, actor: "usr_olga" }];
      return new Response(JSON.stringify({ scope: "x", killed: true }));
    }
    if (url === "/v1/admin/manifest/diff") return new Response(JSON.stringify({ diff: ["- agents: [ask, draft-reply]", "+ agents: [ask]"] }));
    return new Response(JSON.stringify({}));
  };
  return { fetcher, calls };
}

describe("admin console", () => {
  it("kills with a required reason, then offers make permanent as a manifest diff", async () => {
    const s = server();
    const { container } = render(<AdminConsole fetcher={s.fetcher} />);
    await screen.findByText(/No kill switches are set/);
    fireEvent.change(screen.getByLabelText("Name"), { target: { value: "draft-reply" } });
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Kill now" }));
    });
    expect(screen.getByRole("status").textContent).toMatch(/reason is required/);
    expect(s.calls.some((c) => c.method === "POST")).toBe(false);
    fireEvent.change(screen.getByLabelText("Reason (required)"), { target: { value: "runaway drafts" } });
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Kill now" }));
    });
    expect(s.calls.find((c) => c.method === "POST")?.body).toEqual({ scope: "agent", name: "draft-reply", reason: "runaway drafts" });
    await screen.findByText("agent:draft-reply");
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Make permanent" }));
    });
    expect(s.calls.find((c) => c.url === "/v1/admin/manifest/diff")?.body).toEqual([{ op: "remove", path: "/agents/draft-reply" }]);
    expect(container.textContent).toContain("+ agents: [ask]");
    expect(screen.getByRole("status").textContent).toMatch(/sb tenant apply/);
    const r = await axe.run(container, { rules: { "color-contrast": { enabled: false } } });
    expect(r.violations.map((v) => v.id)).toEqual([]);
  });
});
