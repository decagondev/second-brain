import { act, cleanup, fireEvent, render, screen } from "@testing-library/react";
import axe from "axe-core";
import { afterEach, describe, expect, it } from "vitest";
import { App } from "./App";
import type { AskResponse, Fetch, SearchResponse } from "./api";

afterEach(cleanup);

const answer: AskResponse = {
  trace: "tr_1",
  release: "rel_x",
  refused: 0,
  notice: null,
  conversation: "cv_1",
  answer: { claims: [{ text: "Refunds over 500 need a team lead.", cites: [{ cite: "docs:refund@3", title: "Refund policy", span: [0, 9], flags: [] }] }] },
};
const found: SearchResponse = {
  trace: "tr_s",
  release: "rel_x",
  refused: 0,
  results: [
    { cite: "docs:refund@3", title: "Refund policy", url: "https://drive.example/refund", owner: { email: "bob@acme.example" }, age_days: 12, snippet: "SNIPPET-TEXT over 500", flags: [] },
    { cite: "docs:faq@1", title: "Support FAQ", age_days: 1, snippet: "SNIPPET-TEXT faq", flags: [] },
  ],
};

async function violations(container: HTMLElement) {
  const r = await axe.run(container, { rules: { "color-contrast": { enabled: false } } });
  return r.violations.map((v) => `${v.id}: ${v.nodes.map((n) => n.html).join(" ")}`);
}

// SB-2304: with a slow model, search's allowed sources show first, by title, owner and
// age, never their text and never as claims; the cited answer replaces them.
describe("sources first", () => {
  it("shows the sources while the answer is written, then the answer", async () => {
    let finish: (r: Response) => void = () => undefined;
    const fetcher: Fetch = async (u) => {
      if (u === "/v1/spaces") return new Response(JSON.stringify({ spaces: [] }), { status: 200 });
      if (u.startsWith("/v1/changes")) return new Response(JSON.stringify({ items: [] }), { status: 200 });
      if (u.startsWith("/v1/search")) return new Response(JSON.stringify(found), { status: 200 });
      return new Promise<Response>((resolve) => (finish = resolve)); // the model is slow
    };
    const { container } = render(<App fetcher={fetcher} />);
    fireEvent.change(screen.getByLabelText("Ask a question"), { target: { value: "refund limit?" } });
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Ask" }));
    });
    const list = container.querySelector("ul.sources")!;
    expect(list).toBeTruthy();
    expect(list.textContent).toContain("Refund policy");
    expect(list.textContent).toContain("12 days old");
    expect(list.textContent).toContain("bob@acme.example");
    expect(list.textContent).toContain("Support FAQ");
    expect(container.textContent).not.toContain("SNIPPET-TEXT");
    expect(container.querySelectorAll("li.claim").length).toBe(0);
    expect(screen.getByRole("link", { name: /Refund policy, source docs:refund@3/ }).getAttribute("href")).toBe("https://drive.example/refund");
    expect(await violations(container)).toEqual([]);
    await act(async () => {
      finish(new Response(JSON.stringify(answer), { status: 200 }));
    });
    expect(container.querySelector("ul.sources")).toBeNull();
    expect(container.querySelectorAll("li.claim").length).toBe(1);
    expect(await violations(container)).toEqual([]);
  });

  it("says how long to wait over the per-person limit (SB-2305)", async () => {
    const fetcher: Fetch = async (u) => {
      if (u === "/v1/spaces") return new Response(JSON.stringify({ spaces: [] }), { status: 200 });
      if (u.startsWith("/v1/changes")) return new Response(JSON.stringify({ items: [] }), { status: 200 });
      if (u.startsWith("/v1/search")) return new Response("{}", { status: 503 });
      return new Response(
        JSON.stringify({ error: { code: "ERATE", message: "You've asked a lot in the last minute.", trace: "tr_l", remedy: "Wait 42 seconds, then ask again.", retry_after: 42 } }),
        { status: 429 },
      );
    };
    const { container } = render(<App fetcher={fetcher} />);
    fireEvent.change(screen.getByLabelText("Ask a question"), { target: { value: "again?" } });
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Ask" }));
    });
    expect(screen.getByRole("alert").textContent).toContain("Wait 42 seconds");
    expect(container.querySelector(".sources-first")).toBeNull();
    expect(await violations(container)).toEqual([]);
  });
});
