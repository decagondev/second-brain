import { act, cleanup, fireEvent, render, screen } from "@testing-library/react";
import axe from "axe-core";
import { afterEach, describe, expect, it } from "vitest";
import { StewardQuestions } from "./Questions";
import type { Fetch } from "./api";

afterEach(cleanup);

async function violations(container: HTMLElement) {
  const r = await axe.run(container, { rules: { "color-contrast": { enabled: false } } });
  return r.violations.map((v) => `${v.id}: ${v.nodes.map((n) => n.html).join(" ")}`);
}

const json = (v: unknown, status = 200) => new Response(JSON.stringify(v), { status });

// SB-3401 to SB-3404
describe("keep the good replies", () => {
  function cell() {
    const calls: string[] = [];
    const fetcher: Fetch = async (u, init) => {
      calls.push(`${init?.method ?? "GET"} ${u} ${init?.body ?? ""}`);
      if (u.includes("/templates")) return json({ items: [] });
      if (u.endsWith("/note")) return json({ path: "notes/company/qs_3.md", readers: 4, narrower: 0, state: "Ran" });
      if (u.endsWith("/reply")) return json({ id: "qs_1", state: "answered" });
      return json({
        items: [
          { id: "qs_1", question: "vacation days?", at: "2026-11-10T00:00:00Z", state: "open", like: "qs_2", overdue: true },
          { id: "qs_2", question: "how many vacation days?", at: "2026-11-11T00:00:00Z", state: "open", like: "qs_1" },
          { id: "qs_4", question: "payroll date?", at: "2026-11-01T00:00:00Z", state: "open", from_space: "finance", waited: 10 },
          { id: "qs_3", question: "parking?", at: "2026-11-01T00:00:00Z", state: "answered", reply: { by: "sam", text: "Behind B.", cites: [], at: "2026-11-02T00:00:00Z" } },
        ],
      });
    };
    return { calls, fetcher };
  }

  it("shows overdue, repeats and waited questions, and answers both", async () => {
    const { calls, fetcher } = cell();
    const { container } = render(<StewardQuestions space="company" fetcher={fetcher} />);
    await act(async () => {});
    expect(screen.getByText("Overdue")).toBeTruthy();
    expect(screen.getByText("Looks like qs_2")).toBeTruthy();
    expect(screen.getByText("finance hasn't answered this in 10 working days")).toBeTruthy();
    expect(await violations(container)).toEqual([]);
    fireEvent.change(screen.getByLabelText("Reply to: vacation days?"), { target: { value: "25 days." } });
    await act(async () => {
      fireEvent.click(screen.getAllByRole("button", { name: "Answer both" })[0]!);
    });
    expect(calls).toContain('POST /v1/spaces/company/questions/qs_1/reply {"text":"25 days.","cites":[],"both":true}');
  });

  it("keeps the question in the note only when asked", async () => {
    const { calls, fetcher } = cell();
    render(<StewardQuestions space="company" fetcher={fetcher} />);
    await act(async () => {});
    fireEvent.change(screen.getByLabelText("Note title for: parking?"), { target: { value: "Parking" } });
    fireEvent.click(screen.getByLabelText("Keep the question in the note"));
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Publish as note" }));
    });
    expect(calls).toContain('POST /v1/spaces/company/questions/qs_3/note {"title":"Parking","keep_question":true}');
  });
});
