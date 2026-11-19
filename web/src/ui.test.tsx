import { act, cleanup, fireEvent, render, screen } from "@testing-library/react";
import axe from "axe-core";
import { afterEach, describe, expect, it } from "vitest";
import { AnswerView } from "./Answer";
import { App } from "./App";
import type { AskResponse, Claim, Fetch } from "./api";

afterEach(cleanup);

const cited: Claim = {
  text: "Refunds over 500 need a team lead's approval.",
  cites: [{ cite: "docs:4f2a9c11@17", title: "Refund policy", span: [10, 80], url: "https://drive.example/refund?version=17", flags: [] }],
};
const traceOnly: Claim = { text: "It changed in March.", cites: [{ cite: "tickets:LIN-7@2", span: [0, 5], flags: ["injection_suspect"] }] };
const uncited: Claim = { text: "Ask finance for exceptions.", cites: [], uncited: true };

function response(claims: Claim[], extra: Partial<AskResponse> = {}): AskResponse {
  return { trace: "tr_1", release: "rel_x", answer: { claims }, refused: 0, notice: null, conversation: "cv_1", ...extra };
}

async function violations(container: HTMLElement) {
  const r = await axe.run(container, { rules: { "color-contrast": { enabled: false } } }); // jsdom can't compute colours
  return r.violations.map((v) => `${v.id}: ${v.nodes.map((n) => n.html).join(" ")}`);
}

describe("citations", () => {
  it("links a chip to the source at the cited version, or to the trace", () => {
    render(<AnswerView answer={response([cited, traceOnly])} />);
    const src = screen.getByRole("link", { name: /docs:4f2a9c11@17/ });
    expect(src.getAttribute("href")).toBe("https://drive.example/refund?version=17");
    expect(src.getAttribute("target")).toBe("_blank");
    expect(src.getAttribute("rel")).toContain("noopener");
    const tr = screen.getByRole("link", { name: /tickets:LIN-7@2.*flagged/ });
    expect(tr.getAttribute("href")).toBe("/v1/traces/tr_1");
    expect(screen.getByText("Flagged")).toBeTruthy();
  });

  it("labels uncited claims with a word, not colour alone", () => {
    const { container } = render(<AnswerView answer={response([uncited])} />);
    const li = container.querySelector("li.claim")!;
    expect(li.classList.contains("uncited")).toBe(true);
    expect(li.textContent).toContain("Uncited");
    expect(li.querySelectorAll("a").length).toBe(0);
  });

  it("shows notices and refused counts without naming sources", () => {
    render(<AnswerView answer={response([uncited], { notice: "no_access", refused: 2 })} />);
    expect(screen.getByRole("note").textContent).toMatch(/don't have access/);
    expect(screen.getByText(/2 sources were not shown/)).toBeTruthy();
  });
});

// LAW-14 in the UI: whatever claims arrive, every rendered claim is either cited (at
// least one chip) or labelled Uncited, never plain text, and never both.
describe("LAW-14 rendering property", () => {
  it("holds for random claims, including inconsistent ones", () => {
    let seed = 7;
    const rnd = (n: number) => {
      seed = (seed * 1103515245 + 12345) % 2147483648;
      return seed % n;
    };
    for (let run = 0; run < 200; run++) {
      const claims: Claim[] = [];
      for (let i = 0; i < 1 + rnd(5); i++) {
        const cites = Array.from({ length: rnd(3) }, (_, j) => ({ cite: `docs:d${j}@${1 + rnd(9)}`, span: [0, 1] as [number, number], flags: [] }));
        const c: Claim = { text: `claim ${i}`, cites };
        const mode = rnd(3);
        if (mode === 1) c.uncited = true; // possibly with cites: must still show as uncited
        if (mode === 2) delete (c as Partial<Claim>).cites; // malformed: no cites field
        claims.push(c);
      }
      const { container } = render(<AnswerView answer={response(claims)} />);
      const items = container.querySelectorAll("li.claim");
      expect(items.length).toBe(claims.length);
      items.forEach((li, k) => {
        const chips = li.querySelectorAll("a.chip").length;
        const labelled = li.classList.contains("uncited") && (li.textContent ?? "").includes("Uncited");
        expect(chips > 0 !== labelled).toBe(true);
        const claim = claims[k]!;
        if (claim.uncited || !claim.cites?.length) expect(labelled).toBe(true); // the server's label wins
      });
      cleanup();
    }
  });
});

describe("accessibility (axe, WCAG 2.2 A and AA rules that jsdom can check)", () => {
  it("has no violations in an answer", async () => {
    const { container } = render(<AnswerView answer={response([cited, traceOnly, uncited], { notice: "most_uncited", refused: 1 })} />);
    expect(await violations(container)).toEqual([]);
  });

  it("has no violations in the app with an answer and an error", async () => {
    let n = 0;
    const fetcher: Fetch = async () => {
      n++;
      if (n === 1) return new Response(JSON.stringify(response([cited, uncited])), { status: 200 });
      return new Response(JSON.stringify({ error: { code: "EKILLED", message: "Answers are switched off for now.", trace: "tr_9", remedy: "Wait for the operator." } }), { status: 503 });
    };
    const { container } = render(<App fetcher={fetcher} />);
    const box = screen.getByLabelText("Ask a question");
    for (const q of ["refund limit?", "and since when?"]) {
      fireEvent.change(box, { target: { value: q } });
      await act(async () => {
        fireEvent.click(screen.getByRole("button", { name: "Ask" }));
      });
    }
    expect(screen.getByRole("alert").textContent).toContain("Quote trace tr_9");
    expect(screen.getByRole("button", { name: "New conversation" })).toBeTruthy();
    expect(await violations(container)).toEqual([]);
  });
});

describe("conversation", () => {
  it("sends the conversation id on later turns", async () => {
    const bodies: string[] = [];
    const fetcher: Fetch = async (_u, init) => {
      bodies.push(String(init?.body));
      return new Response(JSON.stringify(response([cited])), { status: 200 });
    };
    render(<App fetcher={fetcher} />);
    const box = screen.getByLabelText("Ask a question");
    for (const q of ["one", "two"]) {
      fireEvent.change(box, { target: { value: q } });
      await act(async () => {
        fireEvent.click(screen.getByRole("button", { name: "Ask" }));
      });
    }
    expect(JSON.parse(bodies[0]!)).toEqual({ question: "one" });
    expect(JSON.parse(bodies[1]!)).toEqual({ question: "two", conversation: "cv_1" });
  });
});
