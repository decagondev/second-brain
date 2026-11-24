import { act, cleanup, render, screen } from "@testing-library/react";
import axe from "axe-core";
import { afterEach, describe, expect, it, vi } from "vitest";
import { NoticeBanner, noticeEvery } from "./Notice";
import type { Fetch } from "./api";

afterEach(() => {
  cleanup();
  vi.useRealTimers();
});

const json = (v: unknown, status = 200) => new Response(JSON.stringify(v), { status });
const flush = () => act(async () => {});

describe("the notice to everyone (SB-3207)", () => {
  it("shows the admin's notice as plain text, read again every five minutes, and drops it once it ends", async () => {
    vi.useFakeTimers({ toFake: ["setInterval", "clearInterval", "setTimeout", "clearTimeout", "Date"] });
    vi.setSystemTime(new Date("2027-03-10T09:00:00Z"));
    let body: unknown = { text: "Upgrade at 18:00 <b>tonight</b>", until: "2027-03-10T18:00:00Z" };
    const calls: string[] = [];
    const fetcher: Fetch = async (u) => {
      calls.push(u);
      return json(body);
    };
    const { container } = render(<NoticeBanner fetcher={fetcher} />);
    await flush();
    expect(screen.getByRole("status").textContent).toBe("Notice: Upgrade at 18:00 <b>tonight</b>");
    expect(container.querySelector("b")).toBeNull();

    body = {};
    await act(async () => {
      vi.advanceTimersByTime(noticeEvery);
    });
    await flush();
    expect(calls).toEqual(["/v1/notice", "/v1/notice"]);
    expect(screen.queryByRole("status")).toBeNull();
  });

  it("has no axe violations", async () => {
    const { container } = render(<NoticeBanner fetcher={async () => json({ text: "Upgrade at 18:00.", until: "2999-01-01T00:00:00Z" })} />);
    await flush();
    expect(screen.getByRole("status")).toBeTruthy();
    const r = await axe.run(container, { rules: { "color-contrast": { enabled: false } } });
    expect(r.violations.map((v) => v.id)).toEqual([]);
  });

  it("shows nothing for an ended notice or a failure", async () => {
    for (const res of [json({ text: "old", until: "2000-01-01T00:00:00Z" }), json({}), json({ error: {} }, 503)]) {
      render(<NoticeBanner fetcher={async () => res} />);
      await flush();
      expect(screen.queryByRole("status")).toBeNull();
      cleanup();
    }
    render(<NoticeBanner fetcher={async () => Promise.reject(new Error("down"))} />);
    await flush();
    expect(screen.queryByRole("status")).toBeNull();
  });
});
