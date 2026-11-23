import { act, cleanup, fireEvent, render, screen } from "@testing-library/react";
import axe from "axe-core";
import { afterEach, describe, expect, it } from "vitest";
import { CitationChip } from "./Answer";
import { ReviewList } from "./Questions";
import type { Fetch } from "./api";

afterEach(cleanup);

async function violations(container: HTMLElement) {
  const r = await axe.run(container, { rules: { "color-contrast": { enabled: false } } });
  return r.violations.map((v) => `${v.id}: ${v.nodes.map((n) => n.html).join(" ")}`);
}

const json = (v: unknown, status = 200) => new Response(JSON.stringify(v), { status });

// SB-2801 to SB-2804
describe("retire and tidy", () => {
  function cell() {
    const calls: string[] = [];
    let retired: { document: string; title?: string; reason: string; replaced_by?: string; by: string; at: string }[] = [];
    const fetcher: Fetch = async (u, init) => {
      const m = init?.method ?? "GET";
      calls.push(`${m} ${u} ${init?.body ?? ""}`);
      if (u.endsWith("/retired/restore")) {
        retired = [];
        return json({ document: "docs:old", restored: true });
      }
      if (u.endsWith("/retired")) {
        if (m === "POST") {
          retired = [{ document: "docs:old", title: "Old rota", reason: "Moved to the new rota.", replaced_by: "docs:new", by: "kim@acme-logistics.example", at: "2026-11-23T10:00:00Z" }];
          return json(retired[0]);
        }
        return json({ items: retired, hidden: 1 });
      }
      if (u.includes("section=uncited")) return json({ items: [{ cite: "docs:dusty@2", title: "Dusty guide", snippet: "", flags: [], age_days: 400 }], hidden: [{ owner: "lee@acme-logistics.example" }] });
      return json({
        items: [
          { cite: "docs:old@3", title: "Old rota", snippet: "", flags: [], cited: 2, flagged: 1 },
          { cite: "notes:n1@1", title: "Payroll note", snippet: "", flags: [], cited: 0, flagged: 0, sources_moved: 2 },
        ],
        hidden: [],
        same_name: [{ title: "Rota", items: [{ cite: "docs:a@1", title: "Rota", snippet: "", flags: [] }, { cite: "docs:b@1", title: "Rota", snippet: "", flags: [] }], hidden: 1 }],
      });
    };
    return { calls, fetcher };
  }

  it("retires with a reason, lists it, and brings it back", async () => {
    const { calls, fetcher } = cell();
    const { container } = render(<ReviewList space="ops" fetcher={fetcher} />);
    await act(async () => {});
    expect(screen.getByText(/2 of its sources changed or are gone/)).toBeTruthy();
    expect(screen.getByRole("region", { name: "Same name, different text" }).textContent).toContain("1 more you can't open");
    fireEvent.click(screen.getByRole("button", { name: "Retire from answers: Old rota" }));
    const retire = screen.getByRole("button", { name: "Retire" }) as HTMLButtonElement;
    expect(retire.disabled).toBe(true);
    fireEvent.change(screen.getByLabelText("Why it should leave this space's answers"), { target: { value: "Moved to the new rota." } });
    fireEvent.change(screen.getByLabelText("Replaced by (KIND:ID, optional)"), { target: { value: "docs:new" } });
    expect(await violations(container)).toEqual([]);
    await act(async () => {
      fireEvent.click(retire);
    });
    expect(calls).toContain('POST /v1/spaces/ops/retired {"cite":"docs:old@3","reason":"Moved to the new rota.","replaced_by":"docs:new"}');
    const list = screen.getByRole("region", { name: "Retired from answers" });
    expect(list.textContent).toContain("replaced by docs:new");
    expect(list.textContent).toContain("1 more retired document you can't open");
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Bring back: Old rota" }));
    });
    expect(calls).toContain('POST /v1/spaces/ops/retired/restore {"document":"docs:old"}');
    expect(screen.queryByRole("button", { name: "Bring back: Old rota" })).toBeNull();
  });

  it("shows never cited only when asked, with the owners of the rest", async () => {
    const { calls, fetcher } = cell();
    const { container } = render(<ReviewList space="ops" fetcher={fetcher} />);
    await act(async () => {});
    expect(calls.some((c) => c.includes("section=uncited"))).toBe(false);
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Show never cited" }));
    });
    const sec = screen.getByRole("region", { name: "Never cited" });
    expect(sec.textContent).toContain("Dusty guide");
    expect(sec.textContent).toContain("lee@acme-logistics.example");
    expect(screen.getByRole("button", { name: "Retire from answers: Dusty guide" })).toBeTruthy();
    expect(await violations(container)).toEqual([]);
  });

  it("says on the chip which document it replaces", () => {
    render(<CitationChip trace="tr_1" cite={{ cite: "docs:new@1", title: "New rota", span: [0, 0], flags: [], replaces: "Old rota" }} />);
    expect(screen.getByRole("link").getAttribute("aria-label")).toContain("Replaces Old rota");
  });
});
