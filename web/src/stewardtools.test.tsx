import { act, cleanup, fireEvent, render, screen } from "@testing-library/react";
import axe from "axe-core";
import { afterEach, describe, expect, it } from "vitest";
import { ReviewList, StewardQuestions } from "./Questions";
import type { Fetch } from "./api";

afterEach(cleanup);

async function violations(container: HTMLElement) {
  const r = await axe.run(container, { rules: { "color-contrast": { enabled: false } } });
  return r.violations.map((v) => `${v.id}: ${v.nodes.map((n) => n.html).join(" ")}`);
}

const json = (v: unknown, status = 200) => new Response(JSON.stringify(v), { status });

// SB-2501
describe("reviewed, still right", () => {
  it("marks the document at its version and takes it off the list", async () => {
    const calls: string[] = [];
    const fetcher: Fetch = async (u, init) => {
      calls.push(`${init?.method ?? "GET"} ${u} ${init?.body ?? ""}`);
      if (u.endsWith("/review/marks")) return json({ document: "docs:returns", version: 4 });
      return json({ items: [{ cite: "docs:returns@4", title: "Returns policy", snippet: "", flags: [], cited: 3, flagged: 1 }], hidden: [] });
    };
    const { container } = render(<ReviewList space="support" fetcher={fetcher} />);
    await act(async () => {});
    expect(await violations(container)).toEqual([]);
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Reviewed, still right: Returns policy" }));
    });
    expect(calls).toContain('POST /v1/spaces/support/review/marks {"cite":"docs:returns@4"}');
    expect(screen.queryByText("Returns policy")).toBeNull();
  });
});

// SB-2502, SB-2503
describe("claims and templates", () => {
  function cell(claim?: { by: string; mine: boolean; until: string }) {
    const calls: string[] = [];
    let templates = [{ id: "rt_1", name: "Not ours", text: "Ask finance; they own payroll." }];
    const fetcher: Fetch = async (u, init) => {
      const m = init?.method ?? "GET";
      calls.push(`${m} ${u} ${init?.body ?? ""}`);
      if (u.endsWith("/claim")) return json({ claim: m === "DELETE" ? null : (claim ?? { by: "me@acme-logistics.example", mine: true, until: "2026-11-23T13:00:00Z" }) });
      if (u.includes("/templates")) {
        if (m === "DELETE") templates = templates.filter((x) => !u.endsWith(x.id));
        if (m === "POST") templates = [...templates, { id: "rt_2", ...JSON.parse(String(init?.body)) }];
        return json(m === "GET" ? { items: templates } : {});
      }
      return json({ items: [{ id: "qs_1", question: "payroll?", at: "2026-11-22T00:00:00Z", state: "open", ...(claim ? { claim } : {}) }] });
    };
    return { calls, fetcher };
  }

  it("claims and lets go, and fills the reply from a template", async () => {
    const { calls, fetcher } = cell();
    const { container } = render(<StewardQuestions space="support" fetcher={fetcher} />);
    await act(async () => {});
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "I'll answer this" }));
    });
    expect(screen.getByText(/You said you'd answer this/)).toBeTruthy();
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Let it go" }));
    });
    expect(calls.filter((c) => c.includes("/claim")).map((c) => c.split(" ")[0])).toEqual(["POST", "DELETE"]);
    fireEvent.change(screen.getByLabelText("Use a template"), { target: { value: "rt_1" } });
    expect((screen.getByLabelText("Reply to: payroll?") as HTMLTextAreaElement).value).toBe("Ask finance; they own payroll.");
    expect(await violations(container)).toEqual([]);
  });

  it("says who else is answering, and never blocks the reply", async () => {
    const { fetcher } = cell({ by: "kim@acme-logistics.example", mine: false, until: "2026-11-23T13:00:00Z" });
    render(<StewardQuestions space="support" fetcher={fetcher} />);
    await act(async () => {});
    expect(screen.getByText(/kim@acme-logistics.example is answering this/)).toBeTruthy();
    expect(screen.queryByRole("button", { name: "I'll answer this" })).toBeNull();
    expect(screen.getByRole("button", { name: "Send reply" })).toBeTruthy();
  });

  it("adds and deletes a template", async () => {
    const { calls, fetcher } = cell();
    render(<StewardQuestions space="support" fetcher={fetcher} />);
    await act(async () => {});
    fireEvent.click(screen.getByText("Reply templates"));
    fireEvent.change(screen.getByLabelText("Template name"), { target: { value: "Thanks" } });
    fireEvent.change(screen.getByLabelText("Template text"), { target: { value: "Thanks for asking." } });
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Add template" }));
    });
    expect(calls).toContain('POST /v1/spaces/support/templates {"name":"Thanks","text":"Thanks for asking."}');
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Delete Not ours" }));
    });
    expect(calls.some((c) => c.startsWith("DELETE /v1/spaces/support/templates/rt_1"))).toBe(true);
    expect(screen.queryByRole("button", { name: "Delete Not ours" })).toBeNull();
    expect(screen.getByRole("button", { name: "Delete Thanks" })).toBeTruthy();
  });
});
