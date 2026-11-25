import { act, cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import { AccountBriefForm, AccountFactsView, HowThisWasMade } from "./Further";
import type { Fetch, Research } from "./api";

afterEach(cleanup);

const json = (v: unknown, status = 200) => new Response(JSON.stringify(v), { status });

const how = {
  trace: "tr_1",
  agent: "case-brief",
  release: "rel_7",
  skill: "weekly-update",
  departments: [{ space: "finance", agent: "close-checklist" }],
  not_consulted: [{ why: "restricted", count: 1 }],
  sources: [{ cite: "crm:companies/c1", title: "Gamma Parts" }],
  sources_hidden: 2,
  refused: 1,
  dropped: 0,
  cost_usd: 0.0123,
};

const open = async () => {
  const d = screen.getByText("How this was made").closest("details")!;
  await act(async () => {
    d.open = true;
    d.dispatchEvent(new Event("toggle"));
  });
};

// SB-4604 to SB-4606
describe("agents that go further", () => {
  it("shows how an answer was made only when opened, the hidden sources counted", async () => {
    const urls: string[] = [];
    const fetcher: Fetch = async (u) => (urls.push(u), json(how));
    render(<HowThisWasMade trace="tr_1" fetcher={fetcher} />);
    expect(urls).toEqual([]);
    await open();
    expect(urls).toEqual(["/v1/traces/tr_1/how"]);
    expect(screen.getByText("Gamma Parts")).toBeTruthy();
    expect(screen.getByText("and 2 you can't open")).toBeTruthy();
    expect(screen.getByText("/weekly-update")).toBeTruthy();
    expect(screen.getByText("finance (close-checklist)")).toBeTruthy();
    expect(screen.getByText("$0.0123")).toBeTruthy();
    expect(screen.queryByText("crm:companies/c1")).toBeNull();
  });

  it("asks a report's how in its space", async () => {
    const urls: string[] = [];
    const fetcher: Fetch = async (u) => (urls.push(u), json({ ...how, report: "rs_1", steps: ["What does the record say?"] }));
    render(<HowThisWasMade report="rs_1" space="support" fetcher={fetcher} />);
    await open();
    expect(urls).toEqual(["/v1/research/rs_1/how?space=support"]);
    expect(screen.getByText("What does the record say?")).toBeTruthy();
  });

  it("briefs an account, and shows the cell's answer when it can't", async () => {
    const bodies: string[] = [];
    let made: Research | undefined;
    let fail = false;
    const fetcher: Fetch = async (u, init) => {
      bodies.push(`${u} ${init?.body as string}`);
      return fail ? json({ error: { message: "No such account you can open in this space." } }, 400) : json({ id: "rs_1", kind: "account", question: "Gamma Parts" });
    };
    render(<AccountBriefForm space="support" fetcher={fetcher} onMade={(r) => (made = r)} />);
    fireEvent.change(screen.getByLabelText("CRM account, such as crm:companies/123"), { target: { value: "companies/c1" } });
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Brief me" }));
    });
    expect(bodies[0]).toBe('/v1/account-brief?space=support {"account":"crm:companies/c1"}');
    expect(made?.kind).toBe("account");
    fail = true;
    fireEvent.change(screen.getByLabelText("CRM account, such as crm:companies/123"), { target: { value: "crm:companies/c2" } });
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Brief me" }));
    });
    expect(screen.getByRole("alert").textContent).toBe("No such account you can open in this space.");
    cleanup();
    render(<AccountBriefForm fetcher={fetcher} onMade={() => undefined} />);
    expect(screen.queryByRole("form", { name: "Account brief" })).toBeNull();
  });

  it("shows an account's counted facts", () => {
    render(<AccountFactsView account={{ name: "Gamma Parts", domain: "gamma-parts.example", renewal: "2027-03-31", tickets: 1, notes: 2, documents: 0, owners: ["sam@acme.example"] }} />);
    expect(screen.getByText(/gamma-parts.example · renews 2027-03-31 · 1 open ticket · 2 recent notes · 0 contracts or proposals · Ask: sam@acme.example/)).toBeTruthy();
  });
});
