import { act, cleanup, fireEvent, render, screen } from "@testing-library/react";
import axe from "axe-core";
import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { App } from "./App";
import { AnswerView } from "./Answer";
import { ReviewList, StewardQuestions } from "./Questions";
import { FeedbackBar } from "./Feedback";
import type { AskResponse, Fetch } from "./api";

beforeEach(() => window.localStorage.clear());
afterEach(cleanup);

async function violations(container: HTMLElement) {
  const r = await axe.run(container, { rules: { "color-contrast": { enabled: false } } });
  return r.violations.map((v) => v.id);
}

const json = (x: unknown) => new Response(JSON.stringify(x), { status: 200 });

const answer: AskResponse = {
  trace: "tr_1",
  release: "r",
  answer: { claims: [{ text: "The cutoff is 3pm.", cites: [{ cite: "docs:c@1", title: "Cutoff", span: [0, 10], flags: [] }] }] },
  refused: 0,
  notice: null,
  conversation: "cv_1",
  pinned: { id: "pin_1", question: "When is the cutoff?", text: "3pm on weekdays.", cites: ["docs:c@1"], verified: "2027-03-01" },
};

// SB-3601, SB-3602
describe("trust each answer", () => {
  it("shows the stewards' verified answer first, then the brain's", async () => {
    const { container } = render(<AnswerView answer={answer} />);
    const pin = screen.getByRole("region", { name: "Verified by the stewards" });
    expect(pin.textContent).toContain("Verified by the stewards on 2027-03-01");
    expect(pin.textContent).toContain("3pm on weekdays.");
    expect(container.querySelector(".answer")!.firstElementChild).toBe(pin);
    expect(await violations(container)).toEqual([]);
  });

  it("asks from verified documents only when the box is ticked, and remembers it", async () => {
    const bodies: string[] = [];
    const fetcher: Fetch = async (u, init) => {
      if (u === "/v1/notice") return json({});
      if (u === "/v1/spaces") return json({ spaces: [] });
      if (u.startsWith("/v1/changes")) return json({ items: [] });
      if (u.startsWith("/v1/search")) return json({ trace: "tr_s", release: "r", results: [], refused: 0 });
      bodies.push(String(init?.body));
      return json(answer);
    };
    render(<App fetcher={fetcher} />);
    fireEvent.click(screen.getAllByRole("checkbox", { name: "Verified only" })[0]!);
    fireEvent.change(screen.getByLabelText("Ask a question"), { target: { value: "cutoff?" } });
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Ask" }));
    });
    expect(JSON.parse(bodies[0]!)).toEqual({ question: "cutoff?", verified: true, language: "en" });
    cleanup();
    render(<App fetcher={fetcher} />);
    expect((screen.getAllByRole("checkbox", { name: "Verified only" })[0] as HTMLInputElement).checked).toBe(true);
  });

  it("lets a steward pin a cited reply with the question askers will see, and remove it", async () => {
    const sent: { u: string; body?: string; method?: string }[] = [];
    let pins: unknown[] = [];
    const fetcher: Fetch = async (u, init) => {
      sent.push({ u, body: init?.body as string | undefined, method: init?.method });
      if (u.endsWith("/pins") && init?.method === "POST") {
        pins = [{ id: "pin_1", question: "When is the cutoff?", text: "3pm.", cites: ["docs:c@1"], at: "2027-03-01T00:00:00Z" }];
        return json(pins[0]);
      }
      if (u.endsWith("/pins/pin_1")) {
        pins = [];
        return json({ removed: true });
      }
      if (u.endsWith("/pins")) return json({ items: pins });
      if (u.endsWith("/templates")) return json({ items: [] });
      return json({ items: [{ id: "qs_1", question: "when is the cutoff, from ann?", at: "2027-03-01T00:00:00Z", state: "answered", reply: { by: "s", text: "3pm.", cites: ["docs:c@1"], at: "2027-03-01T00:00:00Z" } }] });
    };
    const { container } = render(<StewardQuestions space="ops" fetcher={fetcher} />);
    await act(async () => {});
    fireEvent.click(screen.getByRole("button", { name: "Pin as the verified answer" }));
    fireEvent.change(screen.getByLabelText("The question as askers will see it"), { target: { value: "When is the cutoff?" } });
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Pin" }));
    });
    expect(JSON.parse(sent.find((x) => x.method === "POST" && x.u.endsWith("/pins"))!.body!)).toEqual({ question_id: "qs_1", question: "When is the cutoff?" });
    await act(async () => {});
    expect(screen.getByRole("region", { name: "Verified answers" }).textContent).toContain("When is the cutoff?");
    expect(await violations(container)).toEqual([]);
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Remove the verified answer: When is the cutoff?" }));
    });
    expect(screen.queryByRole("region", { name: "Verified answers" })).toBeNull();
  });
});

// SB-3605, SB-3606, SB-3607
describe("see which sources to trust", () => {
  it("says how many sources are verified, and nothing without cites", () => {
    render(<AnswerView answer={{ ...answer, pinned: undefined, verified: { of: 2, marked: 1 } }} />);
    expect(screen.getByText("1 of 2 sources verified")).toBeTruthy();
    cleanup();
    render(<AnswerView answer={{ ...answer, pinned: undefined }} />);
    expect(screen.queryByText(/sources verified/)).toBeNull();
  });

  it("sends a report only after Not helpful and the click, with the note", async () => {
    const posts: { u: string; b: unknown }[] = [];
    const fetcher: Fetch = async (u, init) => {
      posts.push({ u, b: JSON.parse(String(init?.body ?? "{}")) });
      return u.startsWith("/v1/questions") ? json({ id: "qs_1", to: "ops" }) : json({});
    };
    const { container } = render(<FeedbackBar answer={answer} question="When is the cutoff?" space="ops" fetcher={fetcher} />);
    expect(screen.queryByRole("button", { name: "Send to the stewards" })).toBeNull();
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Not helpful" }));
    });
    expect(screen.getByText("The stewards will see your name, the question and this answer.")).toBeTruthy();
    expect(posts.some((p) => p.u.startsWith("/v1/questions"))).toBe(false);
    fireEvent.change(screen.getByLabelText("Note for the stewards (optional)"), { target: { value: "It changed." } });
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Send to the stewards" }));
    });
    expect(posts.at(-1)).toEqual({ u: "/v1/questions?space=ops", b: { question: "When is the cutoff?", trace: "tr_1", report: true, note: "It changed." } });
    expect(screen.getByText("Sent to the stewards of ops.")).toBeTruthy();
    expect(await violations(container)).toEqual([]);
  });

  it("shows a report and the conflicts to the stewards, the rest only counted", async () => {
    const doc = (cite: string, title: string) => ({ cite, title, snippet: "", flags: [], span: [0, 1] as [number, number] });
    const fetcher: Fetch = async (u) => {
      if (u.endsWith("/review"))
        return json({ items: [], hidden: [], conflicts: [{ a: doc("docs:a@1", "Cutoff"), b: doc("docs:b@1", "Rota"), answers: 3 }], conflicts_hidden: 2 });
      if (u.endsWith("/questions"))
        return json({ items: [{ id: "qs_1", question: "When?", at: "2027-03-01T00:00:00Z", state: "open", report: { by: "sam@acme-logistics.example", note: "Wrong day.", claims: ["The cutoff is 3pm."], documents: [doc("docs:a@1", "Cutoff")], hidden: 1 } }] });
      return json({ items: [] });
    };
    const { container } = render(
      <>
        <StewardQuestions space="ops" fetcher={fetcher} />
        <ReviewList space="ops" fetcher={fetcher} />
      </>,
    );
    await act(async () => {});
    const rep = screen.getByRole("group", { name: "Report" });
    expect(rep.textContent).toContain("Sent by sam@acme-logistics.example");
    expect(rep.textContent).toContain("Wrong day.");
    expect(rep.textContent).toContain("1 more you can't open");
    const c = screen.getByRole("region", { name: "Conflicts" });
    expect(c.textContent).toContain("Cutoff and Rota disagreed in 3 answers");
    expect(c.textContent).toContain("2 more pairs you can't open");
    expect(await violations(container)).toEqual([]);
  });
});
