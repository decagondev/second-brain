import { act, cleanup, fireEvent, render, screen } from "@testing-library/react";
import axe from "axe-core";
import { afterEach, describe, expect, it } from "vitest";
import { AnswerView } from "./Answer";
import { FeedbackBar } from "./Feedback";
import type { AskResponse, Claim, Fetch } from "./api";

afterEach(cleanup);

const claim: Claim = {
  text: "The cutoff is 3pm.",
  cites: [
    { cite: "docs:a@3", title: "Cutoffs", span: [0, 5], flags: [], age_days: 400, stale: true },
    { cite: "docs:b@2", title: "Rota", span: [0, 5], flags: [], age_days: 400, reviewed: "2026-11-20" },
  ],
};
const answer: AskResponse = {
  trace: "tr_1",
  release: "rel_x",
  answer: { claims: [claim] },
  refused: 0,
  notice: null,
  conversation: "cv_1",
  notice_stale: { stale: 2, of: 3, oldest_days: 400 },
  notice_sync: [
    { kind: "docs", last_success: "2027-03-10T14:05:00Z" },
    { kind: "tickets", last_success: null },
  ],
};

// SB-2701, SB-2702, SB-2704: the warnings come first, and a reviewed cite says so.
describe("freshness on an answer", () => {
  it("warns above the answer and shows the review mark on the cite", async () => {
    const { container } = render(<AnswerView answer={answer} />);
    const notes = screen.getAllByRole("note").map((n) => n.textContent ?? "");
    expect(notes[0]).toMatch(/the oldest is 400 days old\)\. Check with their owner/);
    expect(notes[1]).toMatch(/^docs last synced at .+; newer changes may be missing\.$/);
    expect(notes[2]).toBe("tickets: when it last synced is unknown; newer changes may be missing.");
    expect(container.textContent!.indexOf("oldest is 400")).toBeLessThan(container.textContent!.indexOf("The cutoff"));
    expect(screen.getByRole("link", { name: /docs:a@3, Cutoffs, 400 days old/ })).toBeTruthy();
    expect(screen.getByRole("link", { name: /docs:b@2, Rota, Reviewed 2026-11-20/ })).toBeTruthy();
    const r = await axe.run(container, { rules: { "color-contrast": { enabled: false } } });
    expect(r.violations.map((v) => v.id)).toEqual([]);
  });

  it("says nothing when the answer is fresh", () => {
    render(<AnswerView answer={{ ...answer, notice_stale: undefined, notice_sync: undefined }} />);
    expect(screen.queryAllByRole("note")).toHaveLength(0);
  });
});

// SB-2703: a cite can be reported out of date.
describe("out of date", () => {
  it("sends the outdated cite with the rating", async () => {
    const sent: unknown[] = [];
    const fetcher: Fetch = async (_u, init) => {
      sent.push(JSON.parse(String(init?.body)));
      return new Response("{}", { status: 200 });
    };
    render(<FeedbackBar answer={answer} fetcher={fetcher} />);
    fireEvent.change(screen.getByLabelText("Out of date"), { target: { value: "docs:a@3" } });
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Not helpful" }));
    });
    expect(sent).toEqual([{ trace: "tr_1", rating: "down", outdated_citation: "docs:a@3" }]);
  });
});
