import { act, cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import { App } from "./App";
import type { AskResponse, Fetch } from "./api";
import { spaceQuery } from "./api";

afterEach(cleanup);

const json = (v: unknown, status = 200) => new Response(JSON.stringify(v), { status });

const answer: AskResponse = { trace: "tr_1", release: "rel_x", refused: 0, notice: null, conversation: "cv_1", answer: { claims: [] } };

// SB-3903
describe("ask across a branch", () => {
  it("offers a department and its teams only when the asker is in a team, and asks with branch=true", async () => {
    const urls: string[] = [];
    const fetcher: Fetch = async (u) => {
      urls.push(u);
      if (u === "/v1/notice") return new Response("{}", { status: 200 });
      if (u === "/v1/spaces")
        return json({ spaces: [{ name: "finance", label: "confidential", teams: true }, { name: "ap", label: "confidential", parent: "finance" }, { name: "ops", label: "internal" }] });
      if (u.startsWith("/v1/search")) return json({ trace: "tr_s", release: "rel_x", refused: 0, results: [] });
      return json(answer);
    };
    render(<App fetcher={fetcher} />);
    await act(async () => undefined);
    const select = screen.getByLabelText("Ask in") as HTMLSelectElement;
    const options = Array.from(select.options).map((o) => o.textContent);
    expect(options).toContain("Finance and its teams");
    expect(options).not.toContain("Ops and its teams");
    fireEvent.change(select, { target: { value: "finance+teams" } });
    fireEvent.change(screen.getByLabelText("Ask a question"), { target: { value: "cutoff" } });
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Ask" }));
    });
    expect(urls).toContain("/v1/ask?space=finance&branch=true");
    expect(urls.some((u) => u.startsWith("/v1/search") && u.endsWith("space=finance&branch=true"))).toBe(true);
  });

  it("builds the space part of a URL", () => {
    expect(spaceQuery("finance")).toBe("space=finance");
    expect(spaceQuery("finance+teams")).toBe("space=finance&branch=true");
  });
});

// SB-3905, SB-3906
describe("leaders and who's who", () => {
  it("shows a department's teams as counts, with too few asks said so", async () => {
    const { TeamsTable } = await import("./Questions");
    render(<TeamsTable teams={[{ space: "ap", to_review: 2, overdue: 1, open_questions: 3, waiting: 4, asks: 9, found_nothing: 2 }, { space: "fpa", to_review: 0, overdue: 0, open_questions: 0, fewer_than_5: true }]} />);
    const rows = screen.getAllByRole("row").map((r) => r.textContent);
    expect(rows[1]).toBe("ap23149, 2 found nothing");
    expect(rows[2]).toBe("fpa0000fewer than 5");
  });

  it("shows a space's stewards, owners and member count", async () => {
    const { WhosWho } = await import("./Start");
    const fetcher: Fetch = async () =>
      json({ space: "ap", stewards: [{ email: "sam@acme-logistics.example" }], owners: [{ email: "ann@acme-logistics.example", cites: 4 }], members: 12 });
    const { container } = render(<WhosWho space="ap" fetcher={fetcher} />);
    await act(async () => undefined);
    expect(container.textContent).toContain("12 members");
    expect(container.textContent).toContain("sam@acme-logistics.example");
    expect(container.textContent).toContain("cited 4 times this month");
  });
});
