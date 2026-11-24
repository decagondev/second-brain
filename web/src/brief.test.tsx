import { act, cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import axe from "axe-core";
import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { App } from "./App";
import { NotifySettings } from "./Questions";
import type { Fetch } from "./api";

beforeEach(() => window.localStorage.clear());
afterEach(cleanup);

async function violations(container: HTMLElement) {
  const r = await axe.run(container, { rules: { "color-contrast": { enabled: false } } });
  return r.violations.map((v) => v.id);
}

const json = (x: unknown) => new Response(JSON.stringify(x), { status: 200 });

// SB-3805
describe("daily brief", () => {
  it("chooses the daily digest beside the digest's checkbox, and goes back", async () => {
    const puts: unknown[] = [];
    let digest: string | undefined;
    let off: string[] = [];
    const fetcher: Fetch = async (_u, init) => {
      if (init?.method === "PUT") {
        const b = JSON.parse(String(init.body)) as { off: string[]; digest?: string };
        puts.push(b);
        off = b.off;
        digest = b.digest === "daily" ? "daily" : b.digest === "weekly" ? undefined : digest;
      }
      return json({ off, ...(digest ? { digest } : {}) });
    };
    const { container } = render(<NotifySettings fetcher={fetcher} />);
    await act(async () => {});
    const daily = screen.getByRole("checkbox", { name: "Daily, each working morning" }) as HTMLInputElement;
    expect(daily.checked).toBe(false);
    expect(await violations(container)).toEqual([]);
    await act(async () => {
      fireEvent.click(daily);
    });
    expect((screen.getByRole("checkbox", { name: "Daily, each working morning" }) as HTMLInputElement).checked).toBe(true);
    await act(async () => {
      fireEvent.click(screen.getByRole("checkbox", { name: "Daily, each working morning" }));
    });
    expect(puts).toEqual([
      { off: [], digest: "daily" },
      { off: [], digest: "weekly" },
    ]);
    await act(async () => {
      fireEvent.click(screen.getByRole("checkbox", { name: "What changed in my spaces, weekly" }));
    });
    expect(screen.queryByRole("checkbox", { name: "Daily, each working morning" })).toBeNull();
  });
});

// SB-3806
describe("search from the address bar", () => {
  it("opens the Documents tab and runs the words at once", async () => {
    const urls: string[] = [];
    const fetcher: Fetch = async (u) => {
      urls.push(u);
      if (u === "/v1/notice") return json({});
      if (u === "/v1/spaces") return json({ spaces: [] });
      if (u.startsWith("/v1/changes")) return json({ items: [] });
      if (u.startsWith("/v1/search")) return json({ trace: "tr_s", release: "r", results: [], refused: 0 });
      return json({});
    };
    render(<App fetcher={fetcher} tab="documents" q="refund policy" />);
    await waitFor(() => expect(urls.filter((u) => u.startsWith("/v1/search")).join(" ")).toContain("refund"));
    expect(screen.getByRole("tab", { name: "Documents", selected: true })).toBeTruthy();
  });

  it("ignores q on another tab", async () => {
    const urls: string[] = [];
    const fetcher: Fetch = async (u) => {
      urls.push(u);
      if (u === "/v1/spaces") return json({ spaces: [] });
      return json({ items: [] });
    };
    render(<App fetcher={fetcher} q="refund policy" />);
    await act(async () => {});
    expect(urls.some((u) => u.startsWith("/v1/search"))).toBe(false);
  });
});
