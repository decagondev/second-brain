import { act, cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import axe from "axe-core";
import { afterEach, describe, expect, it } from "vitest";
import { ApproverView } from "./Approver";
import type { ActionView } from "./actions";
import type { Fetch } from "./api";

afterEach(cleanup);

const shown: ActionView = {
  id: "tc_1",
  state: "PendingApproval",
  workflow: "ticket-update",
  target: "tickets:OPS-1182",
  payload: { comment: "Root cause: DNS" },
  payload_sha256: "5d1f0c".padEnd(64, "0"),
  requester: "usr_dana",
  approvers: "group:grp_it_leads",
  expires_at: "2027-03-03T10:41:07Z",
  policy_reason: "propose: permitted by base.cedar#10",
  diff: [{ field: "comment", before: "", after: "Root cause: DNS" }],
};

function server(after?: Partial<ActionView>, fail?: { status: number; code: string }) {
  const posts: { url: string; body: unknown }[] = [];
  let current = shown;
  const fetcher: Fetch = async (url, init) => {
    if (init?.method === "POST") {
      posts.push({ url, body: JSON.parse(String(init.body)) });
      if (fail) return new Response(JSON.stringify({ error: { code: fail.code, message: "Can't approve as shown.", trace: "tr_9", remedy: "Reload it." } }), { status: fail.status });
      current = { ...current, ...after };
      return new Response(JSON.stringify(current), { status: 200 });
    }
    return new Response(JSON.stringify(current), { status: 200 });
  };
  return { fetcher, posts, change: (a: Partial<ActionView>) => (current = { ...current, ...a }) };
}

async function violations(el: HTMLElement) {
  const r = await axe.run(el, { rules: { "color-contrast": { enabled: false } } });
  return r.violations.map((v) => `${v.id}: ${v.nodes.map((n) => n.html).join(" ")}`);
}

describe("approver view", () => {
  it("shows target, requester, policy reason, expiry, payload and diff", async () => {
    const { fetcher } = server();
    const { container } = render(<ApproverView id="tc_1" fetcher={fetcher} />);
    await screen.findByText("Approve ticket-update");
    for (const text of ["tickets", "OPS-1182", "usr_dana", "propose: permitted by base.cedar#10", "2027-03-03T10:41:07Z"]) {
      expect(screen.getAllByText(text).length).toBeGreaterThan(0);
    }
    expect(container.querySelector("pre.payload")?.textContent).toContain("Root cause: DNS");
    expect(container.querySelector("table.diff ins")?.textContent).toBe("Root cause: DNS");
    expect(await violations(container)).toEqual([]);
  });

  it("approves only on an explicit click, echoing the hash it showed", async () => {
    const s = server({ state: "Ran", receipt: "tickets:comment/9", approved_by: "usr_erin" });
    render(<ApproverView id="tc_1" fetcher={s.fetcher} />);
    await screen.findByText("Approve ticket-update");
    s.change({ payload_sha256: "f".repeat(64) }); // the payload changed on the server after the page loaded
    fireEvent.keyDown(screen.getByLabelText("Note (optional)"), { key: "Enter" });
    expect(s.posts).toEqual([]);
    fireEvent.change(screen.getByLabelText("Note (optional)"), { target: { value: "ok" } });
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Approve this payload" }));
    });
    expect(s.posts).toEqual([{ url: "/v1/actions/tc_1/approve", body: { payload_sha256: shown.payload_sha256, note: "ok" } }]);
    await waitFor(() => expect(screen.getByRole("status").textContent).toContain("tickets:comment/9"));
    expect(screen.queryByRole("button", { name: "Approve this payload" })).toBeNull();
  });

  it("shows a refused approval with its remedy", async () => {
    const s = server(undefined, { status: 409, code: "ENOAPPROVAL" });
    render(<ApproverView id="tc_1" fetcher={s.fetcher} />);
    await screen.findByText("Approve ticket-update");
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Reject" }));
    });
    expect(s.posts[0]).toEqual({ url: "/v1/actions/tc_1/reject", body: { note: "" } });
    expect(screen.getByRole("alert").textContent).toContain("Reload it.");
  });

  it("says when there is no diff", async () => {
    const { fetcher } = server();
    const f: Fetch = async (u, i) => {
      const r = await fetcher(u, i);
      const body = { ...(await r.json()), diff: null };
      return new Response(JSON.stringify(body), { status: 200 });
    };
    render(<ApproverView id="tc_1" fetcher={f} />);
    expect(await screen.findByText(/no diff/)).toBeTruthy();
  });
});
