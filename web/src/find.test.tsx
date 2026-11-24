import { act, cleanup, fireEvent, render, screen } from "@testing-library/react";
import axe from "axe-core";
import { afterEach, describe, expect, it } from "vitest";
import { AnswerView } from "./Answer";
import { App } from "./App";
import type { AskResponse, Fetch } from "./api";

afterEach(cleanup);

async function violations(container: HTMLElement) {
  const r = await axe.run(container, { rules: { "color-contrast": { enabled: false } } });
  return r.violations.map((v) => `${v.id}: ${v.nodes.map((n) => n.html).join(" ")}`);
}

const json = (v: unknown, status = 200) => new Response(JSON.stringify(v), { status });

const answer: AskResponse = {
  trace: "tr_1",
  release: "rel_x",
  refused: 0,
  notice: null,
  conversation: "cv_1",
  answer: {
    claims: [
      {
        text: "Refunds over 500 need approval.",
        cites: [{ cite: "docs:r@3", title: "Refund policy", span: [0, 1], flags: [], owner: { email: "ann@acme-logistics.example", title: "Support lead" } }],
      },
    ],
  },
  people: [{ email: "ann@acme-logistics.example", title: "Support lead", cites: 1 }],
};

// SB-1702
describe("who to ask", () => {
  it("names each owner with the documents of theirs the answer cites", async () => {
    const { container } = render(<AnswerView answer={answer} />);
    const list = screen.getByRole("list", { name: "Who to ask" });
    expect(list.textContent).toBe("Ask ann@acme-logistics.example (Support lead), owner of Refund policy");
    expect(await violations(container)).toEqual([]);
  });

  it("shows nobody when the answer names nobody", () => {
    render(<AnswerView answer={{ ...answer, people: [] }} />);
    expect(screen.queryByRole("list", { name: "Who to ask" })).toBeNull();
  });
});

// SB-1701
describe("documents", () => {
  it("searches without asking and shows each result with its quote, owner and age", async () => {
    const urls: string[] = [];
    const fetcher: Fetch = async (u) => {
      urls.push(u);
      if (u === "/v1/notice") return new Response("{}", { status: 200 }); // SB-3207
      if (u === "/v1/spaces") return json({ spaces: [] });
      return json({
        trace: "tr_s",
        release: "rel_x",
        refused: 2,
        results: [
          {
            cite: "docs:r@3",
            title: "Refund policy",
            url: "https://drive.example/r?v=3",
            snippet: "Refunds over 500…",
            flags: [],
            age_days: 41,
            owner: { email: "ann@acme-logistics.example" },
            spaces: ["support"],
            label: "internal",
          },
        ],
      });
    };
    const { container } = render(<App fetcher={fetcher} />);
    fireEvent.click(screen.getByRole("tab", { name: "Documents" }));
    fireEvent.change(screen.getByLabelText("Search documents"), { target: { value: "refund" } });
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Search" }));
    });
    expect(urls).toContain("/v1/search?q=refund");
    expect(urls.some((u) => u.startsWith("/v1/ask"))).toBe(false);
    expect(screen.getByRole("link", { name: /Refund policy, source docs:r@3/ }).getAttribute("href")).toBe("https://drive.example/r?v=3");
    expect(container.textContent).toContain("41 days old");
    expect(container.textContent).toContain("Support · internal");
    expect(container.textContent).toContain("2 documents were not shown");
    expect(await violations(container)).toEqual([]);
  });

  it("offers a search when the model can't answer", async () => {
    const urls: string[] = [];
    const fetcher: Fetch = async (u) => {
      urls.push(u);
      if (u === "/v1/notice") return new Response("{}", { status: 200 }); // SB-3207
      if (u === "/v1/spaces") return json({ spaces: [] });
      if (u.startsWith("/v1/search")) return json({ trace: "tr_s", release: "rel_x", refused: 0, results: [] });
      return json({ error: { code: "EBUDGET", message: "The model budget or rate limit was reached.", trace: "tr_9", remedy: "Wait." } }, 429);
    };
    render(<App fetcher={fetcher} />);
    fireEvent.change(screen.getByLabelText("Ask a question"), { target: { value: "refund limit" } });
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Ask" }));
    });
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Search documents instead" }));
    });
    expect(urls).toContain("/v1/search?q=refund%20limit");
    expect(screen.getByRole("tab", { name: "Documents" }).getAttribute("aria-selected")).toBe("true");
  });

  it("moves between tabs with the arrow keys", () => {
    render(<App fetcher={async () => json({ spaces: [] })} />);
    fireEvent.keyDown(screen.getByRole("tab", { name: "Answer" }), { key: "ArrowRight" });
    expect(screen.getByRole("tab", { name: "Documents" }).getAttribute("aria-selected")).toBe("true");
    fireEvent.keyDown(screen.getByRole("tab", { name: "Documents" }), { key: "ArrowLeft" });
    expect(screen.getByRole("tab", { name: "Answer" }).getAttribute("aria-selected")).toBe("true");
  });
});

// SB-1703
describe("saved and shared questions", () => {
  it("saves and shares only the question, and asks a saved one again", async () => {
    const posts: [string, unknown][] = [];
    const fetcher: Fetch = async (u, init) => {
      if (u === "/v1/notice") return new Response("{}", { status: 200 }); // SB-3207
      if (u === "/v1/spaces") return json({ spaces: [] });
      if (init?.method === "POST") posts.push([u, JSON.parse(String(init.body))]);
      if (u === "/v1/saved" && init?.method === "POST") return json({ id: "sq_1" });
      if (u === "/v1/saved") return json({ items: [{ id: "sq_1", question: "refund limit?", space: "company", at: "2026-11-22T00:00:00Z" }] });
      if (u === "/v1/shares") return json({ id: "sh_" + "a".repeat(32), url: "/q/sh_" + "a".repeat(32), expires: "2026-12-22T00:00:00Z" });
      return json(answer);
    };
    const { container } = render(<App fetcher={fetcher} />);
    fireEvent.change(screen.getByLabelText("Ask a question"), { target: { value: "refund limit?" } });
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Ask" }));
    });
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Save question" }));
    });
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Share question" }));
    });
    expect(posts.filter(([u]) => u !== "/v1/ask")).toEqual([
      ["/v1/saved", { question: "refund limit?" }],
      ["/v1/shares", { question: "refund limit?" }],
    ]);
    expect((screen.getByLabelText("Share link") as HTMLInputElement).value).toMatch(/\/q\/sh_a{32}$/);
    expect(await violations(container)).toEqual([]);
    await act(async () => {
      fireEvent.click(screen.getByRole("tab", { name: "Saved" }));
    });
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Ask again: refund limit?" }));
    });
    expect(posts.filter(([u]) => u === "/v1/ask").length).toBe(2);
    expect(screen.getByRole("tab", { name: "Answer" }).getAttribute("aria-selected")).toBe("true");
  });

  it("opens a share and asks it as the recipient", async () => {
    const asked: string[] = [];
    const fetcher: Fetch = async (u, init) => {
      if (u === "/v1/notice") return new Response("{}", { status: 200 }); // SB-3207
      if (u === "/v1/spaces") return json({ spaces: [] });
      if (u.startsWith("/v1/shares/")) return json({ id: "sh_1", question: "who approves refunds?", space: "spc_support", expires: "2026-12-22T00:00:00Z" });
      if (u.startsWith("/v1/changes")) return json({ items: [] });
      if (u.startsWith("/v1/search")) return new Response(JSON.stringify({ trace: "tr_s", release: "rel_x", results: [], refused: 0 }), { status: 200 }); // SB-2304
      asked.push(`${u} ${String(init?.body)}`);
      return json(answer);
    };
    const { container } = render(<App fetcher={fetcher} share={"sh_" + "b".repeat(32)} />);
    await act(async () => {});
    expect(screen.getByRole("region", { name: "Shared question" }).textContent).toContain("who approves refunds?");
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Ask this question" }));
    });
    // Not a member of support, so the recipient asks company-wide, under their own access.
    expect(asked).toEqual([`/v1/ask {"question":"who approves refunds?","language":"en"}`]); // the asker's language (SB-3003)
    expect(await violations(container)).toEqual([]);
  });

  it("says when a share can't be opened", async () => {
    const fetcher: Fetch = async (u) =>
      u === "/v1/spaces"
        ? json({ spaces: [] })
        : json({ error: { code: "EPERM", message: "That question isn't yours, has expired or doesn't exist.", trace: "", remedy: "" } }, 403);
    render(<App fetcher={fetcher} share={"sh_" + "c".repeat(32)} />);
    await act(async () => {});
    expect(screen.getByRole("alert").textContent).toMatch(/expired/);
  });
});

// SB-2104
describe("watching a saved question", () => {
  it("watches and stops watching one of the person's saved questions", async () => {
    const puts: [string, unknown][] = [];
    let watch = false;
    const fetcher: Fetch = async (u, init) => {
      if (u === "/v1/notice") return new Response("{}", { status: 200 }); // SB-3207
      if (u === "/v1/spaces") return json({ spaces: [] });
      if (init?.method === "PUT") {
        puts.push([u, JSON.parse(String(init.body))]);
        watch = JSON.parse(String(init.body)).watch;
        return json({ id: "sq_1", watch });
      }
      if (u === "/v1/saved") return json({ items: [{ id: "sq_1", question: "refund limit?", space: "company", at: "2026-11-22T00:00:00Z", watch }] });
      return json({ items: [] });
    };
    const { container } = render(<App fetcher={fetcher} tab="saved" />);
    await act(async () => {});
    const box = screen.getByLabelText("Watch: refund limit?") as HTMLInputElement;
    expect(box.checked).toBe(false);
    expect(await violations(container)).toEqual([]);
    await act(async () => {
      fireEvent.click(box);
    });
    expect(puts).toEqual([["/v1/saved/sq_1/watch", { watch: true }]]);
    expect((screen.getByLabelText("Watch: refund limit?") as HTMLInputElement).checked).toBe(true);
  });
});

// SB-2203
describe("glossary", () => {
  it("shows the space's terms an answer used at its foot", async () => {
    const { container } = render(<AnswerView answer={{ ...answer, terms: [{ term: "PO", means: "purchase order" }] }} />);
    expect(container.querySelector(".terms")?.textContent).toBe("Read with this space's glossary: PO means purchase order.");
    expect(await violations(container)).toEqual([]);
  });

  it("shows no glossary line when no term was used", () => {
    const { container } = render(<AnswerView answer={answer} />);
    expect(container.querySelector(".terms")).toBeNull();
  });
});
