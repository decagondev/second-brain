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

describe("admin console: proposals and reviews", () => {
  it("approves a proposal only by the hash it shows, rejects with a reason, and lists reviews overdue first", async () => {
    const calls: { method: string; url: string; body?: unknown }[] = [];
    let open = true;
    const hash = "ab12cd34ef56" + "0".repeat(52);
    const fetcher: Fetch = async (url, init) => {
      const method = init?.method ?? "GET";
      const body = init?.body ? JSON.parse(String(init.body)) : undefined;
      calls.push({ method, url, body });
      if (url === "/v1/admin/proposals")
        return new Response(JSON.stringify({ items: open ? [{ id: "prop_1", space: "finance", requester: "usr_ann", hash, diff: ["+ spaces.finance.members: group:audit"] }] : [] }));
      if (url === "/v1/admin/reviews")
        return new Response(JSON.stringify({ items: [{ id: "rev_a", space: "eng", quarter: "2027-Q2", due: "x", state: "open", items: [], requests: [] },
          { id: "rev_b", space: "finance", quarter: "2027-Q2", due: "y", state: "open", overdue: true, items: [], requests: [] }] }));
      if (url === "/v1/admin/proposals/prop_1/approve") {
        open = false;
        return new Response(JSON.stringify({ id: "prop_1", approved: true, manifest: "tenant: acme-logistics\n" }));
      }
      return new Response(JSON.stringify({}));
    };
    const { container } = render(<AdminConsole fetcher={fetcher} />);
    await screen.findByText(/prop_1 by usr_ann/);
    const items = screen.getAllByRole("listitem").map((li) => li.textContent ?? "");
    expect(items.findIndex((t) => t.includes("finance"))).toBeLessThan(items.findIndex((t) => t.includes("eng")));
    expect(container.textContent).toContain("(overdue)");
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Reject prop_1" }));
    });
    expect(screen.getByRole("status").textContent).toMatch(/needs a reason/);
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Approve hash ab12cd34ef56" }));
    });
    expect(calls.find((c) => c.url.endsWith("/approve"))?.body).toEqual({ hash });
    expect(screen.getByLabelText("Manifest to apply").textContent).toContain("tenant: acme-logistics");
    await screen.findByText("No open proposals.");
    const r = await axe.run(container, { rules: { "color-contrast": { enabled: false } } });
    expect(r.violations.map((v) => v.id)).toEqual([]);
  });
});

describe("admin console: health and the unhelpful-answer queue", () => {
  it("shows stale and overdue items, never question text, and marks an answer reviewed", async () => {
    const calls: { method: string; url: string }[] = [];
    let reviewed = false;
    const fetcher: Fetch = async (url, init) => {
      const method = init?.method ?? "GET";
      calls.push({ method, url });
      if (url === "/v1/admin/health")
        return new Response(
          JSON.stringify({
            jobs: [
              { name: "backup", law: "LAW-11", last_finish: null, overdue: true },
              { name: "sync", law: "LAW-13", last_finish: "2027-03-10T11:59:00Z", overdue: false },
            ],
            connectors: [{ kind: "docs", last_success: "2027-03-10T11:00:00Z", last_failure: "2027-03-10T11:58:00Z", failures: 3, stale: true }],
            at: "2027-03-10T12:00:00Z",
            box: {
              facts_at: "2027-03-10T11:30:00Z",
              verdict: "fail",
              checks: [
                { check: "disk /opt/second-brain", verdict: "fail", detail: "93% used", fix: "free space on /opt/second-brain" },
                { check: "readyz", verdict: "pass", detail: "/readyz answered 200" },
              ],
            },
            notify: { last_run: "2027-03-10T11:55:00Z", via: "email", sent: 4, skipped: 1, failed: 2, warning: true, reason: "the last run failed" },
          }),
        );
      if (url === "/v1/admin/feedback?rating=down")
        return new Response(
          JSON.stringify({
            items: reviewed
              ? []
              : [
                  {
                    trace: "tr_7",
                    at: "2027-03-10T10:00:00Z",
                    user: "usr_ann",
                    rating: "down",
                    comment: "it missed the EU limit",
                    claims: 2,
                    uncited: 1,
                    refused: 1,
                    decisions: [{ verdict: "refuse", reason: "source_denied", count: 1 }],
                  },
                ],
          }),
        );
      if (url === "/v1/admin/feedback/tr_7/reviewed") {
        reviewed = true;
        return new Response(JSON.stringify({ trace: "tr_7", reviewed: { by: "usr_olga", at: "2027-03-10T12:01:00Z" } }));
      }
      if (url === "/v1/admin/manifest") return new Response(JSON.stringify({ manifest: "tenant: acme-logistics\n" }));
      return new Response(JSON.stringify({}));
    };
    const { container } = render(<AdminConsole fetcher={fetcher} />);
    await screen.findByText(/3 failed syncs since/);
    const health = screen.getByRole("region", { name: "Cell health" });
    expect(health.textContent).toMatch(/docs: stale/);
    expect(health.textContent).toMatch(/backup \(LAW-11\): overdue, last finished never/);
    // The box's checks, worst first, each with its fix (SB-1802).
    expect(health.textContent).toMatch(/Worst check: fail/);
    expect(health.textContent).toMatch(/disk \/opt\/second-brain: fail, 93% used \(fix: free space on \/opt\/second-brain\)/);
    // The notify job's last run, counts only, with its warning (SB-2607).
    expect(health.textContent).toMatch(/by email: 4 sent, 1 skipped, 2 failed\. Warning: the last run failed\./);
    // Health and the queue come before everything else.
    const order = Array.from(container.querySelectorAll("h2")).map((h) => h.textContent);
    expect(order.slice(0, 2)).toEqual(["Cell health", "Unhelpful answers"]);
    const q = await screen.findByRole("article", { name: "Feedback on tr_7" });
    expect(q.textContent).toMatch(/it missed the EU limit/);
    expect(q.textContent).toMatch(/refuse source_denied ×1/);
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Mark tr_7 reviewed" }));
    });
    expect(calls.some((c) => c.method === "POST" && c.url === "/v1/admin/feedback/tr_7/reviewed")).toBe(true);
    await screen.findByText("No unhelpful answers to review.");
    const r = await axe.run(container, { rules: { "color-contrast": { enabled: false } } });
    expect(r.violations.map((v) => v.id)).toEqual([]);
  });

  it("says when health can't be read, and still shows the rest", async () => {
    const fetcher: Fetch = async (url) => {
      if (url === "/v1/admin/health") return new Response(JSON.stringify({ error: { code: "EUNAVAIL", message: "The cell's health records can't be read right now." } }), { status: 503 });
      if (url === "/v1/admin/manifest") return new Response(JSON.stringify({ manifest: "tenant: acme-logistics\n" }));
      return new Response(JSON.stringify({}));
    };
    render(<AdminConsole fetcher={fetcher} />);
    await screen.findByText(/Health can't be shown: The cell's health records can't be read right now./);
    await screen.findByText(/tenant: acme-logistics/);
  });
});
