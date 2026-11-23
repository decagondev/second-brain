import { act, cleanup, fireEvent, render, screen } from "@testing-library/react";
import axe from "axe-core";
import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { App } from "./App";
import { DocumentsView } from "./Documents";
import type { AskResponse, Fetch } from "./api";
import { recentQuestions, rememberQuestion } from "./recent";

beforeEach(() => window.localStorage.clear());
afterEach(cleanup);

async function violations(container: HTMLElement) {
  const r = await axe.run(container, { rules: { "color-contrast": { enabled: false } } });
  return r.violations.map((v) => `${v.id}: ${v.nodes.map((n) => n.html).join(" ")}`);
}

const json = (v: unknown, status = 200) => new Response(JSON.stringify(v), { status });
const answer = (notice: AskResponse["notice"] = null): AskResponse => ({
  trace: "tr_1",
  release: "rel_x",
  refused: 0,
  notice,
  conversation: "cv_1",
  answer: { claims: [{ text: "Refunds over 500 need approval.", cites: [{ cite: "docs:r@3", title: "Refund policy", span: [0, 1], flags: [] }] }] },
});

// A cell with one space; every request is kept.
function cell(opts: { results?: unknown[]; refused?: number; notice?: AskResponse["notice"]; askStatus?: number } = {}) {
  const calls: string[] = [];
  const fetcher: Fetch = async (u, init) => {
    calls.push(`${init?.method ?? "GET"} ${u} ${init?.body ?? ""}`);
    if (u === "/v1/spaces") return json({ spaces: [{ name: "support", label: "internal" }] });
    if (u.startsWith("/v1/changes") || u.startsWith("/v1/start")) return json({ items: [], hidden: 0 });
    if (u.startsWith("/v1/search")) return json({ trace: "tr_s", release: "rel_x", results: opts.results ?? [], refused: opts.refused ?? 0 });
    if (u === "/v1/sessions/revoke") return json({ revoked: 2 });
    if (u.startsWith("/v1/ask")) {
      if (opts.askStatus) return json({ error: { code: "EAUTH", message: "Signed out.", trace: "", remedy: "" } }, opts.askStatus);
      return json(answer(opts.notice ?? null));
    }
    return json({});
  };
  return { calls, fetcher };
}

async function askIn(q: string) {
  fireEvent.change(screen.getByLabelText("Ask a question"), { target: { value: q } });
  await act(async () => {
    fireEvent.click(screen.getByRole("button", { name: "Ask" }));
  });
}

// SB-2404
describe("search filters", () => {
  it("sends the type and age with the search", async () => {
    const { calls, fetcher } = cell({ results: [{ cite: "docs:a@1", title: "Refund log", snippet: "rows", flags: [] }] });
    const { container } = render(<DocumentsView fetcher={fetcher} />);
    fireEvent.change(screen.getByLabelText("Search documents"), { target: { value: "refund" } });
    fireEvent.change(screen.getByLabelText("File type"), { target: { value: "sheet" } });
    fireEvent.change(screen.getByLabelText("Changed in"), { target: { value: "30" } });
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Search" }));
    });
    expect(calls.some((c) => c.startsWith("GET /v1/search?q=refund&type=sheet&since=30d"))).toBe(true);
    expect(await violations(container)).toEqual([]);
  });
});

// SB-2406
describe("no dead ends", () => {
  async function emptySearch(refused: number) {
    const { calls, fetcher } = cell({ refused });
    let asked = "";
    render(<DocumentsView fetcher={fetcher} space="support" onAsk={(q) => (asked = q)} />);
    fireEvent.change(screen.getByLabelText("Search documents"), { target: { value: "payroll" } });
    fireEvent.change(screen.getByLabelText("File type"), { target: { value: "pdf" } });
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Search" }));
    });
    const steps = [...document.querySelectorAll(".next-steps button")].map((b) => b.textContent);
    return { calls, steps, asked: () => asked };
  }

  it("offers the whole company, clearing filters and asking, the same with or without a hidden document", async () => {
    const none = await emptySearch(0);
    cleanup();
    const hidden = await emptySearch(3);
    expect(hidden.steps).toEqual(none.steps);
    expect(hidden.steps).toEqual(["Search the whole company", "Clear filters", "Ask it as a question"]);
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Search the whole company" }));
    });
    expect(hidden.calls.at(-1)).toMatch(/^GET \/v1\/search\?q=payroll&type=pdf /);
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Clear filters" }));
    });
    expect(hidden.calls.at(-1)).toMatch(/^GET \/v1\/search\?q=payroll /);
    fireEvent.click(screen.getByRole("button", { name: "Ask it as a question" }));
    expect(hidden.asked()).toBe("payroll");
  });

  it("asks the whole company when a space has nothing", async () => {
    const { calls, fetcher } = cell({ notice: "no_sources" });
    render(<App fetcher={fetcher} />);
    await act(async () => {});
    fireEvent.change(screen.getByLabelText("Ask in"), { target: { value: "support" } });
    await askIn("Who runs payroll?");
    expect(calls.some((c) => c.startsWith("POST /v1/ask?space=support"))).toBe(true);
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Ask the whole company" }));
    });
    const last = calls.filter((c) => c.startsWith("POST /v1/ask")).at(-1)!;
    expect(last.startsWith("POST /v1/ask ")).toBe(true);
    expect(last).toContain("Who runs payroll?");
  });
});

// SB-2405
describe("recent questions", () => {
  it("offers the last questions from this browser, and never sends the list", async () => {
    const { calls, fetcher } = cell();
    const { container } = render(<App fetcher={fetcher} />);
    await askIn("First question");
    const before = calls.length;
    await askIn("Second question");
    for (const c of calls.slice(before)) expect(c).not.toContain("First question");
    expect(recentQuestions()).toEqual(["Second question", "First question"]);
    fireEvent.click(screen.getByText("Recent questions"));
    fireEvent.click(screen.getByRole("button", { name: "First question" }));
    expect((screen.getByLabelText("Ask a question") as HTMLTextAreaElement).value).toBe("First question");
    expect(await violations(container)).toEqual([]);
    fireEvent.click(screen.getByRole("button", { name: "Clear the list" }));
    expect(recentQuestions()).toEqual([]);
    expect(screen.queryByText("Recent questions")).toBeNull();
  });

  it("forgets them at sign-out, and when the cell says the person is signed out", async () => {
    const { fetcher } = cell();
    render(<App fetcher={fetcher} />);
    await askIn("Kept until sign-out");
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Sign out everywhere" }));
    });
    expect(recentQuestions()).toEqual([]);
    cleanup();
    const signedOut = cell({ askStatus: 401 });
    window.localStorage.setItem("sb.recent", JSON.stringify(["older"]));
    render(<App fetcher={signedOut.fetcher} />);
    await askIn("After the session ended");
    expect(recentQuestions()).toEqual([]);
  });

  it("keeps at most 20, each once, and survives storage being off", () => {
    for (let i = 0; i < 22; i++) rememberQuestion(`Question ${i}`);
    rememberQuestion("Question 21");
    expect(recentQuestions()).toHaveLength(20);
    expect(recentQuestions()[0]).toBe("Question 21");
    window.localStorage.setItem("sb.recent", "not json");
    expect(recentQuestions()).toEqual([]);
  });
});
