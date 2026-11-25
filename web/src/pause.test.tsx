import { act, cleanup, render, screen } from "@testing-library/react";
import axe from "axe-core";
import { afterEach, describe, expect, it } from "vitest";
import { App } from "./App";
import type { Fetch } from "./api";

afterEach(cleanup);

async function violations(container: HTMLElement) {
  const r = await axe.run(container, { rules: { "color-contrast": { enabled: false } } });
  return r.violations.map((v) => `${v.id}: ${v.nodes.map((n) => n.html).join(" ")}`);
}

describe("pause from the notice (SB-5107)", () => {
  it("offers Pause this agent here from the notice's link and pauses with a reason", async () => {
    const posts: [string, string][] = [];
    const f: Fetch = async (u, init) => {
      if (init?.method === "POST") {
        posts.push([u, String(init.body)]);
        return new Response(JSON.stringify({ space: "support", agent: "case-brief", paused: true }), { status: 200 });
      }
      return new Response(JSON.stringify({ items: [], results: [], spaces: [] }), { status: 200 });
    };
    render(<App fetcher={f} space="support" pause="case-brief" />);
    await act(async () => {});
    const region = screen.getByRole("region", { name: "Pause this agent here" });
    expect(region.textContent).toContain("Pause case-brief in support?");
    expect(await violations(region)).toEqual([]);
  });
});
