import { act, cleanup, fireEvent, render, screen } from "@testing-library/react";
import axe from "axe-core";
import { afterEach, describe, expect, it } from "vitest";
import type { Fetch } from "./api";
import { StewardView } from "./Steward";

afterEach(cleanup);

function server(opts: { refuse?: boolean } = {}) {
  const calls: { method: string; url: string; body?: unknown }[] = [];
  let review = {
    id: "rev_1",
    space: "finance",
    quarter: "2027-Q2",
    due: "2027-04-15",
    state: "open",
    items: [
      { user: "usr_ann", via: ["group:finance"], last_use: "2027-03-30T10:00:00Z" },
      { user: "usr_bo", via: ["group:finance"], last_use: "never" },
    ],
    requests: null as unknown,
  };
  const fetcher: Fetch = async (url, init) => {
    const method = init?.method ?? "GET";
    const body = init?.body ? JSON.parse(String(init.body)) : undefined;
    calls.push({ method, url, body });
    if (url === "/v1/spaces/finance/reviews") return new Response(JSON.stringify({ items: [review] }));
    if (url === "/v1/spaces/finance/proposals") {
      if (opts.refuse) {
        return new Response(JSON.stringify({ error: { code: "EMANIFEST", message: "1 error", errors: [{ field: "spaces.finance.label", message: "can't be lowered" }] } }), { status: 422 });
      }
      return new Response(JSON.stringify({ id: "prop_7", space: "finance", base: "aaa", hash: "f00d" + "0".repeat(60), diff: ["+ spaces.finance.members: group:audit"], plan: "~ space finance" }));
    }
    if (url === "/v1/spaces/finance/reviews/rev_1/decide") {
      if (body.user === "usr_ann" && body.decision === "certify") {
        return new Response(JSON.stringify({ error: { code: "EPERM", message: "You can't decide on your own access." } }), { status: 403 });
      }
      review = {
        ...review,
        state: "complete",
        items: review.items.map((i) => (i.user === body.user ? { ...i, decision: body.decision, by: "usr_cy", reason: body.reason } : { ...i, decision: "certify", by: "usr_cy" })),
        requests: [{ kind: "idp", user: "usr_bo", group: "group:finance" }],
      };
      return new Response(JSON.stringify(review));
    }
    return new Response(JSON.stringify({}));
  };
  return { fetcher, calls };
}

describe("steward screen", () => {
  it("proposes a change and shows the hash the operator approves", async () => {
    const s = server();
    const { container } = render(<StewardView space="finance" fetcher={s.fetcher} />);
    await screen.findByText(/2027-Q2/);
    fireEvent.change(screen.getByLabelText("The space's new definition (JSON)"), { target: { value: "not json" } });
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Propose" }));
    });
    expect(screen.getByRole("status").textContent).toMatch(/must be JSON/);
    expect(s.calls.some((c) => c.method === "POST")).toBe(false);
    fireEvent.change(screen.getByLabelText("The space's new definition (JSON)"), { target: { value: '{"members": ["group:finance", "group:audit"]}' } });
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Propose" }));
    });
    expect(s.calls.find((c) => c.method === "POST")?.body).toEqual({ space: { members: ["group:finance", "group:audit"] } });
    expect(container.textContent).toContain("f00d");
    expect(container.textContent).toContain("+ spaces.finance.members: group:audit");
    expect(screen.getByRole("status").textContent).toMatch(/Nothing changes until an operator approves/);
    const r = await axe.run(container, { rules: { "color-contrast": { enabled: false } } });
    expect(r.violations.map((v) => v.id)).toEqual([]);
  });

  it("shows why a proposal was refused", async () => {
    const s = server({ refuse: true });
    render(<StewardView space="finance" fetcher={s.fetcher} />);
    await screen.findByText(/2027-Q2/);
    fireEvent.change(screen.getByLabelText("The space's new definition (JSON)"), { target: { value: '{"label": "public"}' } });
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Propose" }));
    });
    expect(screen.getByRole("status").textContent).toMatch(/spaces.finance.label: can't be lowered/);
  });

  it("decides a review: a removal needs a reason, your own access is refused, completion lists the requests", async () => {
    const s = server();
    const { container } = render(<StewardView space="finance" fetcher={s.fetcher} />);
    await screen.findByText("usr_bo");
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "usr_ann still needs it" }));
    });
    expect(screen.getByRole("status").textContent).toMatch(/own access/);
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Remove usr_bo" }));
    });
    expect(screen.getByRole("status").textContent).toMatch(/needs a reason/);
    expect(s.calls.filter((c) => c.url.endsWith("/decide")).length).toBe(1);
    fireEvent.change(screen.getByLabelText("Reason for removing usr_bo (required)"), { target: { value: "left the team" } });
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Confirm removal of usr_bo" }));
    });
    expect(s.calls.filter((c) => c.url.endsWith("/decide")).pop()?.body).toEqual({ user: "usr_bo", decision: "remove", reason: "left the team" });
    expect(container.textContent).toContain("IdP change: take usr_bo out of group:finance");
    expect(screen.getByRole("status").textContent).toMatch(/complete/);
    const r = await axe.run(container, { rules: { "color-contrast": { enabled: false } } });
    expect(r.violations.map((v) => v.id)).toEqual([]);
  });
});
