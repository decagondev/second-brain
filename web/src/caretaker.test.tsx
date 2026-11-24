import { act, cleanup, fireEvent, render, screen } from "@testing-library/react";
import axe from "axe-core";
import { afterEach, describe, expect, it } from "vitest";
import { StewardView } from "./Steward";
import type { Fetch } from "./api";

afterEach(cleanup);

async function violations(container: HTMLElement) {
  const r = await axe.run(container, { rules: { "color-contrast": { enabled: false } } });
  return r.violations.map((v) => `${v.id}: ${v.nodes.map((n) => n.html).join(" ")}`);
}

const json = (v: unknown, status = 200) => new Response(JSON.stringify(v), { status });

// SB-3405, SB-3407
describe("keep someone on every document", () => {
  it("names a caretaker and fills the glossary form from a defined term", async () => {
    const calls: string[] = [];
    let named = false;
    const fetcher: Fetch = async (u, init) => {
      const m = init?.method ?? "GET";
      calls.push(`${m} ${u} ${init?.body ?? ""}`);
      if (u.includes("/caretakers")) {
        named = m === "PUT";
        return json({ doc: "docs:Ops/policy.md", email: "lee@acme.example" });
      }
      if (u.endsWith("/review")) {
        return json({
          items: [{ cite: "docs:Ops/policy.md@3", title: "Policy", snippet: "", flags: [], cited: 0, flagged: 0, owner_gone: true, ...(named ? { owner: { email: "lee@acme.example", caretaker: true } } : {}) }],
          hidden: [],
          terms: [{ term: "PO", meaning: "purchase order", docs: 2 }],
        });
      }
      return json({ items: [] });
    };
    const { container } = render(<StewardView space="ops" fetcher={fetcher} />);
    await act(async () => {});
    expect(await violations(container)).toEqual([]);
    fireEvent.change(screen.getByLabelText("Caretaker's email for Policy"), { target: { value: "lee@acme.example" } });
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Name a caretaker" }));
    });
    expect(calls).toContain('PUT /v1/spaces/ops/caretakers {"doc":"docs:Ops/policy.md","email":"lee@acme.example"}');
    expect(await screen.findByText(/Caretaker lee@acme.example/)).toBeTruthy();
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Remove the caretaker of Policy" }));
    });
    expect(calls).toContain("DELETE /v1/spaces/ops/caretakers?doc=docs%3AOps%2Fpolicy.md ");
    fireEvent.click(screen.getByText("Terms the documents define"));
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Suggest PO" }));
    });
    expect((screen.getByLabelText("Term") as HTMLInputElement).value).toBe("PO");
    expect((screen.getByLabelText("Meaning") as HTMLInputElement).value).toBe("purchase order");
    expect(calls.some((c) => c.includes("/proposals/term"))).toBe(false); // only filled, nothing proposed
  });
});
