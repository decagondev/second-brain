import { act, cleanup, fireEvent, render, screen } from "@testing-library/react";
import axe from "axe-core";
import { afterEach, describe, expect, it, vi } from "vitest";
import { NotifySettings, ReviewList, StewardQuestions, csvCell, reviewCSV } from "./Questions";
import type { Fetch } from "./api";

afterEach(() => {
  cleanup();
  vi.useRealTimers();
});

async function violations(container: HTMLElement) {
  const r = await axe.run(container, { rules: { "color-contrast": { enabled: false } } });
  return r.violations.map((v) => `${v.id}: ${v.nodes.map((n) => n.html).join(" ")}`);
}

const json = (v: unknown, status = 200) => new Response(JSON.stringify(v), { status });

const review = {
  items: [
    { cite: "docs:returns@4", title: "=HYPERLINK(\"x\")", snippet: "", flags: [], cited: 3, flagged: 1, age_days: 200 },
    { cite: "docs:faq@2", title: "FAQ", snippet: "", flags: [], cited: 1, flagged: 0, owner_gone: true },
  ],
  hidden: [{ owner: "ann@acme-logistics.example" }, { owner: "ann@acme-logistics.example" }],
};

// SB-2504
describe("counts on the steward screen", () => {
  it("says how many wait, in words for screen readers, and reads again every five minutes", async () => {
    vi.useFakeTimers({ toFake: ["setInterval", "clearInterval"] });
    let reads = 0;
    let open = 2;
    const fetcher: Fetch = async (u) => {
      if (u.endsWith("/review")) return json(review);
      if (u.endsWith("/templates") || u.endsWith("/retired") || u.endsWith("/pins")) return json({ items: [] });
      reads++;
      return json({ items: Array.from({ length: open }, (_, i) => ({ id: `qs_${i}`, question: `q${i}?`, at: "2026-11-22T00:00:00Z", state: "open" })) });
    };
    const { container } = render(
      <>
        <StewardQuestions space="support" fetcher={fetcher} />
        <ReviewList space="support" fetcher={fetcher} />
      </>,
    );
    await act(async () => {});
    const [q, r] = [...container.querySelectorAll("h2")];
    expect(q!.textContent).toBe("Questions (2)2 waiting");
    expect(r!.textContent).toBe("Documents to review (4)4 waiting");
    expect(container.querySelectorAll(".saved-list")[1]!.querySelectorAll("li")).toHaveLength(2);
    expect(await violations(container)).toEqual([]);
    open = 1;
    await act(async () => {
      vi.advanceTimersByTime(5 * 60 * 1000);
    });
    await act(async () => {});
    expect(reads).toBe(2);
    expect(q!.textContent).toBe("Questions (1)1 waiting");
  });
});

// SB-2505
describe("the review list as CSV", () => {
  it("has the rows on screen, hidden documents only as a count, and no formula runs", () => {
    const csv = reviewCSV(review).split("\r\n").filter(Boolean);
    expect(csv).toHaveLength(1 + 2 + 1);
    expect(csv[1]).toBe(`"'=HYPERLINK(""x"")","docs:returns@4","3","200","1","0","false",""`);
    expect(csv[2]).toBe(`"FAQ","docs:faq@2","1","","0","0","true",""`);
    expect(csv[3]).toBe(`"2 more documents you can't open","","","","","","","ann@acme-logistics.example"`);
    for (const c of ["+1", "-1", "@x", "\tx"]) expect(csvCell(c).startsWith(`"'`)).toBe(true);
  });

  it("downloads from the list already loaded, with no new request", async () => {
    const calls: string[] = [];
    const fetcher: Fetch = async (u) => {
      calls.push(u);
      return json(review);
    };
    let blob: Blob | undefined;
    URL.createObjectURL = (b: Blob) => ((blob = b), "blob:x");
    URL.revokeObjectURL = () => {};
    render(<ReviewList space="support" fetcher={fetcher} />);
    await act(async () => {});
    fireEvent.click(screen.getByRole("button", { name: "Download CSV" }));
    expect(calls.filter((u) => u.endsWith("/review"))).toHaveLength(1);
    const text = await new Promise<string>((res) => {
      const r = new FileReader();
      r.onload = () => res(String(r.result));
      r.readAsText(blob!);
    });
    expect(text).toBe(reviewCSV(review));
  });
});

// SB-2506
describe("pause notifications", () => {
  it("pauses for the days given and resumes", async () => {
    const puts: unknown[] = [];
    let until: string | undefined;
    let off: string[] = [];
    const fetcher: Fetch = async (_u, init) => {
      if (init?.method === "PUT") {
        const b = JSON.parse(String(init.body)) as { off: string[]; pause_days?: number };
        puts.push(b);
        off = b.off;
        until = b.pause_days ? new Date(Date.now() + b.pause_days * 864e5).toISOString() : b.pause_days === 0 ? undefined : until;
      }
      return json({ off, ...(until ? { paused_until: until } : {}) });
    };
    const { container } = render(<NotifySettings fetcher={fetcher} />);
    await act(async () => {});
    fireEvent.change(screen.getByLabelText("Pause for (days)"), { target: { value: "31" } });
    expect((screen.getByRole("button", { name: "Pause" }) as HTMLButtonElement).disabled).toBe(true);
    fireEvent.change(screen.getByLabelText("Pause for (days)"), { target: { value: "3" } });
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Pause" }));
    });
    expect(screen.getByText(/Paused until/)).toBeTruthy();
    expect(await violations(container)).toEqual([]);
    await act(async () => {
      fireEvent.click(screen.getByRole("checkbox", { name: "Replies to my questions" }));
    });
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Resume" }));
    });
    expect(puts).toEqual([{ off: [], pause_days: 3 }, { off: ["replies"] }, { off: ["replies"], pause_days: 0 }]);
    expect(screen.queryByText(/Paused until/)).toBeNull();
  });
});

// SB-2602
describe("send me a test", () => {
  it("sends the test and says through which channel", async () => {
    const calls: string[] = [];
    const fetcher: Fetch = async (u, init) => {
      calls.push(`${init?.method ?? "GET"} ${u}`);
      return u.endsWith("/test") ? json({ sent: true, via: "mattermost" }) : json({ off: [] });
    };
    render(<NotifySettings fetcher={fetcher} />);
    await act(async () => {});
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Send me a test" }));
    });
    expect(calls).toContain("POST /v1/me/notifications/test");
    expect(screen.getByRole("status").textContent).toBe("Sent by mattermost. If nothing arrives, tell your administrator.");
  });
});
