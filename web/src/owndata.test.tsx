import { act, cleanup, fireEvent, render, screen } from "@testing-library/react";
import axe from "axe-core";
import { afterEach, describe, expect, it } from "vitest";
import { MyData, MySessions } from "./Session";
import type { Fetch } from "./api";

afterEach(cleanup);

async function violations(container: HTMLElement) {
  const r = await axe.run(container, { rules: { "color-contrast": { enabled: false } } });
  return r.violations.map((v) => `${v.id}: ${v.nodes.map((n) => n.html).join(" ")}`);
}

const json = (v: unknown, status = 200) => new Response(JSON.stringify(v), { status });

// SB-3501 to SB-3503
describe("your own data", () => {
  it("lists the person's sessions and signs one out", async () => {
    const calls: string[] = [];
    const fetcher: Fetch = async (u, init) => {
      calls.push(`${init?.method ?? "GET"} ${u}`);
      if (init?.method === "DELETE") return json({ signed_out: true });
      return json({
        items: [
          { id: "ses_1", kind: "web", started: "2026-11-24T08:00:00Z", last_used: "2026-11-24", expires: "2026-11-24T20:00:00Z", current: true },
          { id: "ses_2", kind: "token", name: "nightly", started: "2026-11-20T08:00:00Z", last_used: "2026-11-23", expires: "2026-11-27T08:00:00Z" },
        ],
      });
    };
    const { container } = render(<MySessions fetcher={fetcher} />);
    const details = container.querySelector("details")!;
    details.open = true;
    await act(async () => {
      details.dispatchEvent(new Event("toggle"));
    });
    expect(screen.getByText(/This browser/)).toBeTruthy();
    expect(await violations(container)).toEqual([]);
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Sign out: API token nightly" }));
    });
    expect(calls).toContain("DELETE /v1/me/sessions/ses_2");
    expect(screen.queryByText(/API token nightly/)).toBeNull();
  });

  it("forgets only after the phrase is typed, and says what a hold kept", async () => {
    const calls: string[] = [];
    const fetcher: Fetch = async (u, init) => {
      calls.push(`${init?.method ?? "GET"} ${u} ${init?.body ?? ""}`);
      return json({ erased: { traces: 3 }, held: 2 });
    };
    const { container } = render(<MyData fetcher={fetcher} />);
    expect(await violations(container)).toEqual([]);
    const forget = screen.getByRole("button", { name: "Forget my questions" }) as HTMLButtonElement;
    expect(forget.disabled).toBe(true);
    fireEvent.change(screen.getByLabelText(/to erase your questions/), { target: { value: "forget my questions" } });
    await act(async () => {
      fireEvent.click(forget);
    });
    expect(calls).toContain('POST /v1/me/forget {"confirm":"forget my questions"}');
    expect(screen.getByRole("status").textContent).toBe("Forgotten, except 2 records a legal hold keeps.");
  });
});
