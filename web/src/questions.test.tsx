import { act, cleanup, fireEvent, render, screen } from "@testing-library/react";
import axe from "axe-core";
import { afterEach, describe, expect, it } from "vitest";
import { App } from "./App";
import { StewardQuestions } from "./Questions";
import type { AskResponse, Fetch } from "./api";

afterEach(cleanup);

async function violations(container: HTMLElement) {
  const r = await axe.run(container, { rules: { "color-contrast": { enabled: false } } });
  return r.violations.map((v) => `${v.id}: ${v.nodes.map((n) => n.html).join(" ")}`);
}

const json = (v: unknown, status = 200) => new Response(JSON.stringify(v), { status });

const nothing: AskResponse = {
  trace: "tr_7",
  release: "rel_x",
  refused: 0,
  notice: "no_sources",
  conversation: "cv_1",
  answer: { claims: [{ text: "I couldn't find anything you have access to that answers this.", cites: [], uncited: true }] },
};

// SB-1704
describe("asking the stewards", () => {
  it("shows exactly what is sent, and sends it only when the asker confirms", async () => {
    const posts: [string, unknown][] = [];
    const fetcher: Fetch = async (u, init) => {
      if (u === "/v1/spaces") return json({ spaces: [] });
      if (u === "/v1/questions" && init?.method === "POST") {
        posts.push([u, JSON.parse(String(init.body))]);
        return json({ id: "qs_1", to: "company" });
      }
      if (u === "/v1/questions")
        return json({
          items: [
            {
              id: "qs_1",
              question: "zebra cutoff?",
              space: "company",
              at: "2026-11-22T00:00:00Z",
              state: "answered",
              reply: { text: "3pm on weekdays.", at: "2026-11-22T01:00:00Z", cites: [{ cite: "docs:r@1", title: "Shipping", flags: [], snippet: "" }], dropped: 1, note: true },
            },
          ],
        });
      return json(nothing);
    };
    const { container } = render(<App fetcher={fetcher} />);
    fireEvent.change(screen.getByLabelText("Ask a question"), { target: { value: "zebra cutoff?" } });
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Ask" }));
    });
    fireEvent.click(screen.getByRole("button", { name: "Ask the people who answer company-wide questions" }));
    expect(screen.getByRole("group", { name: "What will be sent" }).textContent).toContain("zebra cutoff?");
    expect(posts).toEqual([]);
    expect(await violations(container)).toEqual([]);
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Send question" }));
    });
    expect(posts).toEqual([["/v1/questions", { question: "zebra cutoff?", trace: "tr_7" }]]);
    await act(async () => {
      fireEvent.click(screen.getByRole("tab", { name: "Questions" }));
    });
    const panel = screen.getByRole("region", { name: "Questions to stewards" });
    expect(panel.textContent).toContain("3pm on weekdays.");
    expect(panel.textContent).toContain("1 source was not shown");
    expect(panel.textContent).toContain("Published as a note");
    expect(await violations(container)).toEqual([]);
  });

  it("isn't offered when the answer answered", async () => {
    const fetcher: Fetch = async (u) => (u === "/v1/spaces" ? json({ spaces: [] }) : json({ ...nothing, notice: null }));
    render(<App fetcher={fetcher} />);
    fireEvent.change(screen.getByLabelText("Ask a question"), { target: { value: "x" } });
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Ask" }));
    });
    expect(screen.queryByRole("button", { name: /Ask the/ })).toBeNull();
  });
});

// SB-1705
describe("the steward's queue", () => {
  it("replies, previews a note and publishes it only after confirming the narrower count", async () => {
    const posts: [string, unknown][] = [];
    let replied = false;
    const fetcher: Fetch = async (u, init) => {
      if (init?.method === "POST") {
        posts.push([u, JSON.parse(String(init.body))]);
        if (u.endsWith("/reply")) {
          replied = true;
          return json({ id: "qs_1", state: "answered" });
        }
        const b = JSON.parse(String(init.body)) as { confirm?: number };
        return json(b.confirm === undefined ? { path: "notes/support/qs_1.md", readers: 4, narrower: 1 } : { path: "notes/support/qs_1.md", readers: 4, narrower: 1, state: "Ran" });
      }
      return json({
        items: [{ id: "qs_1", question: "zebra cutoff?", at: "2026-11-22T00:00:00Z", state: replied ? "answered" : "open", ...(replied ? { reply: { by: "u", text: "3pm.", cites: [], at: "x" } } : {}) }],
      });
    };
    const { container } = render(<StewardQuestions space="support" fetcher={fetcher} />);
    await act(async () => {});
    fireEvent.change(screen.getByLabelText("Reply to: zebra cutoff?"), { target: { value: "3pm." } });
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Send reply" }));
    });
    expect(await violations(container)).toEqual([]);
    fireEvent.change(screen.getByLabelText("Note title for: zebra cutoff?"), { target: { value: "Zebra cutoff" } });
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Publish as note" }));
    });
    expect(screen.getByRole("status").textContent).toMatch(/4 readers\. 1 of its cites/);
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Publish, accepting 1 narrower cites" }));
    });
    expect(posts.map(([u, b]) => [u.split("/").pop(), b])).toEqual([
      ["reply", { text: "3pm.", cites: [] }],
      ["note", { title: "Zebra cutoff" }],
      ["note", { title: "Zebra cutoff", confirm: 1 }],
    ]);
    expect(screen.getByRole("status").textContent).toContain("Published as notes/support/qs_1.md");
  });
});

// SB-1706
describe("start here", () => {
  it("shows a space's list to a member, and counts what they can't open", async () => {
    const fetcher: Fetch = async (u) => {
      if (u === "/v1/spaces") return json({ spaces: [{ name: "support", label: "internal" }] });
      if (u === "/v1/start?space=support")
        return json({ items: [{ cite: "docs:welcome@2", title: "Welcome to support", url: "https://drive.example/w", flags: [], snippet: "" }], hidden: 1 });
      return json(nothing);
    };
    const { container } = render(<App fetcher={fetcher} />);
    await act(async () => {});
    expect(screen.queryByRole("region", { name: "Start here" })).toBeNull();
    await act(async () => {
      fireEvent.change(screen.getByLabelText("Ask in"), { target: { value: "support" } });
    });
    const list = screen.getByRole("region", { name: "Start here" });
    expect(list.textContent).toContain("Welcome to support");
    expect(list.textContent).toContain("1 item is not shown");
    expect(await violations(container)).toEqual([]);
  });
});
