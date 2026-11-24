import { act, cleanup, fireEvent, render, screen } from "@testing-library/react";
import axe from "axe-core";
import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { App } from "./App";
import { AnswerView } from "./Answer";
import { StewardQuestions } from "./Questions";
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
