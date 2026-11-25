import { act, cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import { AskForChanges, ShareRoutine, SharedRoutines, SkillHint } from "./Shape";
import type { Fetch } from "./api";

afterEach(cleanup);

const json = (v: unknown, status = 200) => new Response(JSON.stringify(v), { status });

// SB-4601 to SB-4603
describe("shape the work", () => {
  it("offers the space's skills while the question starts with a slash", async () => {
    let picked = "";
    const fetcher: Fetch = async () => json({ skills: [{ name: "weekly-update", instruction: "Three headings.", from: "ops" }] });
    render(<SkillHint space="returns" question="/we" fetcher={fetcher} onPick={(q) => (picked = q)} />);
    await act(async () => undefined);
    expect(screen.getByText("Three headings. · from ops")).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "/weekly-update" }));
    expect(picked).toBe("/weekly-update ");
    cleanup();
    render(<SkillHint space="returns" question="What shipped?" fetcher={fetcher} onPick={() => undefined} />);
    await act(async () => undefined);
    expect(screen.queryByRole("list", { name: "Skills" })).toBeNull();
  });

  it("asks for changes with a note, shows the diff, and stops after 3 rounds", async () => {
    const calls: { url: string; body?: string }[] = [];
    let next = "";
    const fetcher: Fetch = async (u, init) => {
      calls.push({ url: u, body: init?.body as string });
      return json({ id: "qs_1", text: "- Dock opens at seven. [Dock]", rounds: 1, versions: ["old"], diff: [] });
    };
    render(<AskForChanges space="ops" id="qs_1" rounds={1} diff={["- Old line", "+ New line"]} fetcher={fetcher} onDone={(x) => (next = x)} />);
    expect(screen.getByText("+ New line").className).toBe("added");
    expect(screen.getByText("1 round of changes")).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "Ask for changes" }));
    fireEvent.change(screen.getByLabelText("What should change?"), { target: { value: "Lead with the decision." } });
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Redraft" }));
    });
    expect(calls[0]!.url).toBe("/v1/spaces/ops/questions/qs_1/changes");
    expect(JSON.parse(calls[0]!.body!)).toEqual({ note: "Lead with the decision." });
    expect(next).toBe("- Dock opens at seven. [Dock]");
    cleanup();
    render(<AskForChanges space="ops" id="qs_1" rounds={3} fetcher={fetcher} onDone={() => undefined} />);
    expect(screen.queryByRole("button", { name: "Ask for changes" })).toBeNull();
  });

  it("lists shared routines: members opt in, the steward sees only the count", async () => {
    const calls: { url: string; method?: string }[] = [];
    let joined = false;
    const fetcher: Fetch = async (u, init) => {
      calls.push({ url: u, method: init?.method });
      if (init?.method === "POST") joined = true;
      return json({
        routines: [
          { id: "sq_1", question: "What changed at the dock?", every: "daily", joined },
          { id: "sq_2", question: "What is at risk?", every: "weekly", day: "monday", joined: false, mine: true, opted_in: 4 },
        ],
      });
    };
    render(<SharedRoutines space="ops" fetcher={fetcher} />);
    await act(async () => undefined);
    expect(screen.getByText("· 4 members opted in")).toBeTruthy();
    expect(screen.queryByRole("button", { name: "Opt in: What is at risk?" })).toBeNull();
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Opt in: What changed at the dock?" }));
    });
    expect(calls.some((c) => c.method === "POST" && c.url === "/v1/spaces/ops/routines/sq_1/join")).toBe(true);
    expect(screen.getByRole("button", { name: "Opt out: What changed at the dock?" })).toBeTruthy();
  });

  it("shares a steward's own routine with its space", async () => {
    const calls: { url: string; method?: string }[] = [];
    const fetcher: Fetch = async (u, init) => {
      calls.push({ url: u, method: init?.method });
      return json({});
    };
    let changed = 0;
    const q = { id: "sq_1", question: "What changed?", space: "spc_ops", at: "", routine: { every: "daily" as const } };
    render(<ShareRoutine q={q} fetcher={fetcher} onChange={() => changed++} />);
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Share with the space: What changed?" }));
    });
    expect(calls[0]).toEqual({ url: "/v1/spaces/ops/routines/sq_1", method: "PUT" });
    expect(changed).toBe(1);
    cleanup();
    render(<ShareRoutine q={{ ...q, from: "sq_9" }} fetcher={fetcher} onChange={() => undefined} />);
    expect(screen.queryByRole("button")).toBeNull();
  });
});
