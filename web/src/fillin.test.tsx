import { act, cleanup, fireEvent, render, screen } from "@testing-library/react";
import axe from "axe-core";
import { afterEach, describe, expect, it } from "vitest";
import { AnswerView } from "./Answer";
import { ReviewList, StewardQuestions } from "./Questions";
import type { AskResponse, Fetch } from "./api";

afterEach(cleanup);

const json = (x: unknown) => new Response(JSON.stringify(x), { status: 200 });

async function violations(container: HTMLElement) {
  const r = await axe.run(container, { rules: { "color-contrast": { enabled: false } } });
  return r.violations.map((v) => v.id);
}

// SB-3705, SB-3706, SB-3707
describe("fill it in once", () => {
  it("drafts a note from a found-nothing term", async () => {
    const posts: { u: string; b: unknown }[] = [];
    const fetcher: Fetch = async (u, init) => {
      if (init?.method === "POST") {
        posts.push({ u, b: JSON.parse(String(init.body)) });
        return json({ id: "qs_d" });
      }
      return json({ items: [], hidden: [], found_nothing: { this_week: 3, last_week: 0, terms: ["PO", "SLA"], counts: { PO: 3 } } });
    };
    const { container } = render(<ReviewList space="ops" fetcher={fetcher} />);
    await act(async () => {});
    expect(screen.queryByRole("button", { name: "Draft a note for SLA" })).toBeNull();
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Draft a note for PO" }));
    });
    expect(posts).toEqual([{ u: "/v1/spaces/ops/drafts", b: { term: "PO", count: 3 } }]);
    expect(screen.getByRole("status").textContent).toContain("Added a draft for PO");
    expect(await violations(container)).toEqual([]);
  });

  it("asks the owner from Who to ask", async () => {
    const posts: unknown[] = [];
    const fetcher: Fetch = async (_u, init) => {
      posts.push(JSON.parse(String(init?.body)));
      return json({ id: "qs_o", to: "ann@acme-logistics.example" });
    };
    const answer: AskResponse = {
      trace: "tr_1",
      release: "r",
      answer: { claims: [{ text: "x", cites: [{ cite: "docs:a@1", title: "Rates", span: [0, 1], flags: [], owner: { email: "ann@acme-logistics.example" } }] }] },
      refused: 0,
      notice: null,
      conversation: "cv_1",
      people: [{ email: "ann@acme-logistics.example", cites: 1 }],
    };
    render(<AnswerView answer={answer} fetcher={fetcher} question="Until when?" space="ops" />);
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Ask ann@acme-logistics.example this question" }));
    });
    expect(posts).toEqual([{ question: "Until when?", trace: "tr_1", to: "ann@acme-logistics.example" }]);
    expect(screen.getByRole("status").textContent).toContain("Sent to ann@acme-logistics.example");
  });

  it("warns before a note repeats a document, and the owner's own queue hides while empty", async () => {
    const fetcher: Fetch = async (u) => {
      if (u.includes("/note")) return json({ path: "notes/ops/qs_1.md", readers: 4, narrower: 0, similar: [{ cite: "docs:rates@2", title: "Rates", snippet: "", flags: [] }] });
      if (u.endsWith("/spaces/me/questions")) return json({ items: [] });
      if (u.endsWith("/questions")) return json({ items: [{ id: "qs_1", question: "zebra", at: "2027-03-01T00:00:00Z", state: "answered", draft: { title: "zebra", comment: "Asked 2 times this week, found nothing" }, reply: { by: "x", text: "Rates are fixed.", cites: [], at: "2027-03-01T00:00:00Z" } }] });
      return json({ items: [] });
    };
    const { container } = render(
      <>
        <StewardQuestions space="me" own fetcher={fetcher} />
        <StewardQuestions space="ops" fetcher={fetcher} />
      </>,
    );
    await act(async () => {});
    expect(screen.queryByRole("region", { name: "Questions to you" })).toBeNull();
    expect(screen.getByText(/Asked 2 times this week, found nothing/)).toBeTruthy();
    expect((screen.getByLabelText("Note title for: zebra") as HTMLInputElement).value).toBe("zebra");
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Publish as note" }));
    });
    expect(screen.getByRole("note", { name: "Already written?" }).textContent).toContain("Rates");
    expect(await violations(container)).toEqual([]);
  });
});
