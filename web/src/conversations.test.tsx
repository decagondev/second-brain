import { act, cleanup, fireEvent, render, screen } from "@testing-library/react";
import axe from "axe-core";
import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { DocumentsView } from "./Documents";
import { StewardQuestions } from "./Questions";
import type { Fetch } from "./api";

beforeEach(() => window.localStorage.clear());
afterEach(cleanup);

const json = (x: unknown) => new Response(JSON.stringify(x), { status: 200 });

async function violations(container: HTMLElement) {
  const r = await axe.run(container, { rules: { "color-contrast": { enabled: false } } });
  return r.violations.map((v) => v.id);
}

// SB-3702, SB-3703
describe("capture conversations", () => {
  it("lists a space's decisions with no query", async () => {
    const calls: string[] = [];
    const fetcher: Fetch = async (u) => {
      calls.push(u);
      return json({ trace: "tr_1", release: "r", results: [{ cite: "docs:carrier@1", title: "Carrier", snippet: "We move", flags: [] }], refused: 0 });
    };
    render(<DocumentsView fetcher={fetcher} space="ops" />);
    fireEvent.change(screen.getByLabelText("File type"), { target: { value: "decision" } });
    await act(async () => {
      fireEvent.submit(screen.getByRole("search"));
    });
    expect(calls).toEqual(["/v1/search?q=&type=decision&space=ops"]);
    expect(screen.getByText("Carrier")).toBeTruthy();
  });

  it("shows a saved thread, starts the reply with it, and discards it", async () => {
    const calls: string[] = [];
    let gone = false;
    const fetcher: Fetch = async (u, init) => {
      calls.push(`${init?.method ?? "GET"} ${u}`);
      if (init?.method === "DELETE") {
        gone = true;
        return json({ discarded: true });
      }
      if (u.endsWith("/questions"))
        return json({ items: gone ? [] : [{ id: "qs_t", question: "ann: When?", at: "2027-03-01T00:00:00Z", state: "open", thread: { by: "sam@acme-logistics.example", link: "https://slack.com/archives/C1/p1", text: "ann: When?\nben: Friday." } }] });
      return json({ items: [] });
    };
    const { container } = render(<StewardQuestions space="ops" fetcher={fetcher} />);
    await act(async () => {});
    const g = screen.getByRole("group", { name: "Saved from chat" });
    expect(g.textContent).toContain("Sent by sam@acme-logistics.example");
    expect(screen.getByRole("link", { name: "Open the thread" }).getAttribute("href")).toBe("https://slack.com/archives/C1/p1");
    expect((screen.getByLabelText("Reply to: ann: When?") as HTMLTextAreaElement).value).toBe("ann: When?\nben: Friday.");
    expect(await violations(container)).toEqual([]);
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Discard" }));
    });
    expect(calls).toContain("DELETE /v1/spaces/ops/questions/qs_t");
    expect(screen.queryByRole("group", { name: "Saved from chat" })).toBeNull();
  });
});
