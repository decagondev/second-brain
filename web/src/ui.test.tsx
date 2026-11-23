import { act, cleanup, fireEvent, render, screen } from "@testing-library/react";
import axe from "axe-core";
import { afterEach, describe, expect, it } from "vitest";
import { AnswerView } from "./Answer";
import { App } from "./App";
import { FeedbackBar } from "./Feedback";
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

  // SB-2205
  it("asks the owners of hidden sources for access and shows only counts", async () => {
    const sent: unknown[] = [];
    const fetcher: Fetch = async (u, init) => {
      sent.push([u, JSON.parse(String(init?.body))]);
      return new Response(JSON.stringify({ sent: 1, already: 0, no_owner: 1 }), { status: 200 });
    };
    render(<AnswerView answer={response([uncited], { notice: "no_access", refused: 2 })} fetcher={fetcher} />);
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Ask the owners for access" }));
    });
    expect(sent).toEqual([["/v1/access-requests", { trace: "tr_1" }]]);
    expect(screen.getByRole("status").textContent).toBe("Done: asked 1 owner, 1 with no owner to ask.");
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
        const cites = Array.from({ length: rnd(3) }, (_, j) => ({
          cite: `docs:d${j}@${1 + rnd(9)}`, span: [0, 1] as [number, number], flags: [],
          ...(rnd(2) ? { spaces: ["finance", "people"].slice(0, 1 + rnd(2)), label: ["internal", "restricted"][rnd(2)] } : {}), // SB-1007
        }));
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
    const fetcher: Fetch = async (url) => {
      if (url === "/v1/spaces") return new Response(JSON.stringify({ spaces: [] }), { status: 200 });
      if (url.startsWith("/v1/changes")) return new Response(JSON.stringify({ items: [] }), { status: 200 }); // SB-1708
      if (url.startsWith("/v1/search")) return new Response(JSON.stringify({ trace: "tr_s", release: "rel_x", results: [], refused: 0 }), { status: 200 }); // SB-2304
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
    const fetcher: Fetch = async (u, init) => {
      if (u === "/v1/spaces") return new Response(JSON.stringify({ spaces: [] }), { status: 200 });
      if (u.startsWith("/v1/changes")) return new Response(JSON.stringify({ items: [] }), { status: 200 }); // SB-1708
      if (u.startsWith("/v1/search")) return new Response(JSON.stringify({ trace: "tr_s", release: "rel_x", results: [], refused: 0 }), { status: 200 }); // SB-2304
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

describe("feedback", () => {
  it("sends the rating, wrong citation and comment against the trace", async () => {
    const sent: unknown[] = [];
    const fetcher: Fetch = async (url, init) => {
      if (url === "/v1/feedback") {
        sent.push(JSON.parse(String(init?.body)));
        return new Response(JSON.stringify({ trace: "tr_1", stored: true }), { status: 200 });
      }
      return new Response(JSON.stringify(response([cited, uncited])), { status: 200 });
    };
    const { container } = render(<App fetcher={fetcher} />);
    fireEvent.change(screen.getByLabelText("Ask a question"), { target: { value: "refund limit?" } });
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Ask" }));
    });
    fireEvent.change(screen.getByLabelText("Wrong citation"), { target: { value: "docs:4f2a9c11@17" } });
    fireEvent.change(screen.getByLabelText("Comment"), { target: { value: " cites the old policy " } });
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Not helpful" }));
    });
    expect(sent).toEqual([{ trace: "tr_1", rating: "down", wrong_citation: "docs:4f2a9c11@17", comment: "cites the old policy" }]);
    expect(screen.getByRole("status").textContent).toMatch(/saved/);
    expect(screen.getByRole("button", { name: "Not helpful" }).getAttribute("aria-pressed")).toBe("true");
    expect(await violations(container)).toEqual([]);
  });

  it("offers only cited claims' citations as wrong", () => {
    render(<FeedbackBar answer={response([cited, { ...uncited, cites: [{ cite: "docs:x@1", span: [0, 1], flags: [] }] }])} />);
    const options = Array.from(screen.getByLabelText("Wrong citation").querySelectorAll("option")).map((o) => o.value);
    expect(options).toEqual(["", "docs:4f2a9c11@17"]);
  });
});


// SB-1007: the switcher lists only the asker's spaces, the chosen space goes on the ask,
// and every chip says where its source lives and how sensitive it is.
describe("spaces", () => {
  const finance: Claim = {
    text: "Q3 is ahead of plan.",
    cites: [{ cite: "docs:q3@2", title: "Q3 close", span: [0, 9], flags: [], spaces: ["finance"], label: "restricted" }],
  };

  it("shows each citation's spaces and label in words, in its accessible name", () => {
    render(<AnswerView answer={response([finance])} />);
    const chip = screen.getByRole("link", { name: /docs:q3@2, Q3 close, Finance · restricted/ });
    expect(chip.textContent).toContain("Finance · restricted");
  });

  it("asks in the chosen space and starts a new conversation when it changes", async () => {
    const urls: string[] = [];
    const fetcher: Fetch = async (u) => {
      urls.push(u);
      if (u === "/v1/spaces") return new Response(JSON.stringify({ spaces: [{ name: "finance", label: "restricted" }] }), { status: 200 });
      return new Response(JSON.stringify(response([finance])), { status: 200 });
    };
    const { container } = render(<App fetcher={fetcher} />);
    await act(async () => {});
    const select = screen.getByLabelText("Ask in") as HTMLSelectElement;
    expect(Array.from(select.options).map((o) => o.textContent)).toEqual(["Whole company", "Finance (restricted)"]);
    fireEvent.change(select, { target: { value: "finance" } });
    fireEvent.change(screen.getByLabelText("Ask a question"), { target: { value: "how is Q3?" } });
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Ask" }));
    });
    expect(urls).toContain("/v1/ask?space=finance");
    expect(await violations(container)).toEqual([]);
    fireEvent.change(select, { target: { value: "" } });
    expect(screen.queryByRole("button", { name: "New conversation" })).toBeNull();
    fireEvent.change(screen.getByLabelText("Ask a question"), { target: { value: "and overall?" } });
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Ask" }));
    });
    expect(urls[urls.length - 1]).toBe("/v1/ask");
  });

  it("shows no switcher when the cell has no spaces or can't list them", async () => {
    for (const reply of [{ spaces: [] }, { error: { code: "EUNAVAIL" } }]) {
      const fetcher: Fetch = async () => new Response(JSON.stringify(reply), { status: "error" in reply ? 503 : 200 });
      render(<App fetcher={fetcher} />);
      await act(async () => {});
      expect(screen.queryByLabelText("Ask in")).toBeNull();
      cleanup();
    }
  });
});

describe("signed out", () => {
  it("sends a signed-out user to sign in, and never repeats the same words twice", async () => {
    const fetcher: Fetch = async (u) => {
      if (u === "/v1/spaces") return new Response(JSON.stringify({ spaces: [] }), { status: 401 });
      return new Response(JSON.stringify({ error: { code: "EAUTH", message: "sign in again", remedy: "Sign in again", trace: "" } }), { status: 401 });
    };
    const { container } = render(<App fetcher={fetcher} />);
    fireEvent.change(screen.getByLabelText("Ask a question"), { target: { value: "what documents do we have?" } });
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Ask" }));
    });
    expect(screen.getByRole("link", { name: "Sign in" }).getAttribute("href")).toBe("/login");
    expect(container.textContent).not.toMatch(/sign in again.*sign in again/i);
    expect(await violations(container)).toEqual([]);
  });
});

describe("sign out everywhere", () => {
  it("posts the revoke and says how many sessions ended", async () => {
    const { SignOutEverywhere } = await import("./Session");
    const calls: string[] = [];
    const f: Fetch = async (input, init) => {
      calls.push(`${init?.method ?? "GET"} ${input}`);
      return new Response(JSON.stringify({ revoked: 3, audit: ["aud_1", "aud_2"] }), { status: 200 });
    };
    const { container } = render(<SignOutEverywhere fetcher={f} />);
    expect(await violations(container)).toEqual([]);
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Sign out everywhere" }));
    });
    expect(calls).toEqual(["POST /v1/sessions/revoke"]);
    expect(screen.getByRole("status").textContent).toContain("3 devices");
    expect(screen.getByRole("link", { name: "Sign in again" }).getAttribute("href")).toBe("/login");
  });

  it("says so when the sessions weren't ended", async () => {
    const { SignOutEverywhere } = await import("./Session");
    const f: Fetch = async () => new Response("{}", { status: 503 });
    render(<SignOutEverywhere fetcher={f} />);
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Sign out everywhere" }));
    });
    expect(screen.getByRole("alert").textContent).toContain("weren't ended");
  });
});
