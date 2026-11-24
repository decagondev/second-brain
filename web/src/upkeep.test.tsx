import { act, cleanup, fireEvent, render, screen } from "@testing-library/react";
import axe from "axe-core";
import { afterEach, describe, expect, it } from "vitest";
import { DocumentsView } from "./Documents";
import { ReviewList } from "./Questions";
import { StewardView } from "./Steward";
import type { Fetch } from "./api";

afterEach(cleanup);

const json = (v: unknown, status = 200) => new Response(JSON.stringify(v), { status });

async function violations(container: HTMLElement) {
  const r = await axe.run(container, { rules: { "color-contrast": { enabled: false } } });
  return r.violations.map((v) => v.id);
}

function cell() {
  const calls: string[] = [];
  const fetcher: Fetch = async (u, init) => {
    const m = init?.method ?? "GET";
    calls.push(`${m} ${u} ${init?.body ?? ""}`);
    if (u.includes("section=due")) {
      return json({
        items: [
          { cite: "docs:e@3", title: "Rota", snippet: "", flags: [], due_in_days: 3 },
          { cite: "docs:a@3", title: "Cutoffs", snippet: "", flags: [], due_in_days: 10 },
        ],
        hidden: [{}],
      });
    }
    if (u.startsWith("/v1/search")) {
      return json({
        trace: "tr_1",
        release: "r",
        refused: 0,
        results: [
          { cite: "docs:a@3", title: "Cutoffs", snippet: "x", flags: [], mine: true },
          { cite: "docs:b@3", title: "Rota", snippet: "y", flags: [] },
        ],
      });
    }
    if (u === "/v1/documents/reviewed") return json({ document: "docs:a", version: 3, spaces: ["ops"] });
    if (u.endsWith("/proposals/term")) return json({ id: "prop_9", hash: "abc", diff: [] });
    if (u.endsWith("/review/marks")) return json({ document: "docs:e", version: 3 });
    return json({ items: [] });
  };
  return { calls, fetcher };
}

// SB-3105
describe("coming due", () => {
  it("lists soonest first and marks one reviewed", async () => {
    const { calls, fetcher } = cell();
    const { container } = render(<ReviewList space="ops" fetcher={fetcher} />);
    await act(async () => {
      fireEvent.click(await screen.findByRole("button", { name: "Show coming due" }));
    });
    const items = screen.getByRole("region", { name: "Coming due" }).querySelectorAll("li");
    expect(Array.from(items).map((li) => li.textContent)).toEqual([expect.stringMatching(/Rota · due in 3 days/), expect.stringMatching(/Cutoffs · due in 10 days/)]);
    expect(screen.getByText("1 more document you can't open falls due")).toBeTruthy();
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Reviewed, still right: Rota" }));
    });
    expect(calls).toContain(`POST /v1/spaces/ops/review/marks {"cite":"docs:e@3"}`);
    expect(screen.queryByRole("button", { name: "Reviewed, still right: Rota" })).toBeNull();
    expect(await violations(container)).toEqual([]);
  });
});

// SB-3106
describe("owners mark their own", () => {
  it("offers the mark only on the person's own documents", async () => {
    const { calls, fetcher } = cell();
    render(<DocumentsView fetcher={fetcher} />);
    fireEvent.change(screen.getByLabelText("Search documents"), { target: { value: "cutoff" } });
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Search" }));
    });
    expect(screen.queryByRole("button", { name: "Reviewed, still right: Rota" })).toBeNull();
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Reviewed, still right: Cutoffs" }));
    });
    expect(calls).toContain(`POST /v1/documents/reviewed {"cite":"docs:a@3"}`);
    expect(screen.getByText("Marked reviewed: Cutoffs")).toBeTruthy();
  });
});

// SB-3107
describe("suggest a term", () => {
  it("sends a proposal with the term, its meaning and its spellings", async () => {
    const { calls, fetcher } = cell();
    render(<StewardView space="ops" fetcher={fetcher} />);
    fireEvent.change(screen.getByLabelText("Term"), { target: { value: "COD" } });
    fireEvent.change(screen.getByLabelText("Meaning"), { target: { value: "cash on delivery" } });
    fireEvent.change(screen.getByLabelText("Other spellings, separated by commas (at most 10)"), { target: { value: "C.O.D., cod " } });
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Suggest a term" }));
    });
    expect(calls).toContain(`POST /v1/spaces/ops/proposals/term {"term":"COD","meaning":"cash on delivery","spellings":["C.O.D.","cod"]}`);
    expect(screen.getByText(/Proposal prop_9 made\. Nothing changes until an operator approves it/)).toBeTruthy();
  });
});
