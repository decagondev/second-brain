import { act, cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import { SharedOnPurpose, WhoCanSee } from "./Sharing";
import type { Fetch } from "./api";

afterEach(cleanup);

describe("know who can see (SB-5402, SB-5404)", () => {
  it("asks for every cite's audience when opened, and names only the groups it is given", async () => {
    const urls: string[] = [];
    const f: Fetch = async (u) => {
      urls.push(u);
      return new Response(JSON.stringify({ documents: 2, people: 41, groups: ["finance"], other_groups: 2 }), { status: 200 });
    };
    render(<WhoCanSee docs={["docs:pay@3", "docs:menu@1"]} fetcher={f} />);
    expect(urls).toHaveLength(0);
    fireEvent.click(screen.getByRole("button", { name: "Who can see this" }));
    await act(async () => {});
    expect(urls[0]).toBe("/v1/audience?doc=docs%3Apay&doc=docs%3Amenu");
    const s = screen.getByRole("status").textContent ?? "";
    expect(s).toContain("41 people can open this now");
    expect(s).toContain("Through: finance");
    expect(s).toContain("and 2 other groups");
  });

  it("confirms shared on purpose with a reason", async () => {
    const bodies: string[] = [];
    const f: Fetch = async (_u, init) => {
      bodies.push(String(init?.body ?? ""));
      return new Response(JSON.stringify({ document: "docs:pay" }), { status: 200 });
    };
    render(<SharedOnPurpose cite="docs:pay@3" fetcher={f} />);
    fireEvent.change(screen.getByLabelText("Why it is shared this widely"), { target: { value: "payroll team" } });
    fireEvent.click(screen.getByRole("button", { name: "Shared on purpose" }));
    await act(async () => {});
    expect(JSON.parse(bodies[0]!)).toEqual({ cite: "docs:pay@3", reason: "payroll team" });
    expect(screen.getByText("Confirmed as shared on purpose.")).toBeTruthy();
  });
});
