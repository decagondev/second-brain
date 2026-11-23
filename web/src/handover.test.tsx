import { act, cleanup, fireEvent, render, screen } from "@testing-library/react";
import axe from "axe-core";
import { afterEach, describe, expect, it } from "vitest";
import { App } from "./App";
import type { Fetch } from "./api";

afterEach(cleanup);

async function violations(container: HTMLElement) {
  const r = await axe.run(container, { rules: { "color-contrast": { enabled: false } } });
  return r.violations.map((v) => `${v.id}: ${v.nodes.map((n) => n.html).join(" ")}`);
}

const json = (v: unknown, status = 200) => new Response(JSON.stringify(v), { status });

async function openHandover(fetcher: Fetch) {
  const r = render(<App fetcher={fetcher} />);
  await act(async () => {
    fireEvent.click(screen.getByRole("tab", { name: "Handover" }));
  });
  return r;
}

async function brief(person: string) {
  fireEvent.change(screen.getByLabelText("Person"), { target: { value: person } });
  await act(async () => {
    fireEvent.click(screen.getByRole("button", { name: "Brief" }));
  });
}

// SB-2002
describe("the handover screen", () => {
  it("shows the cited brief grouped by location, and the documents the asker can't open as a count", async () => {
    const posts: unknown[] = [];
    const fetcher: Fetch = async (u, init) => {
      if (u === "/v1/handover") {
        posts.push(JSON.parse(String(init?.body)));
        return json({
          trace: "tr_h1",
          person: "usr_dana",
          dropped: 1,
          hidden: 2,
          locations: { "docs:a@1": "/Support/Refunds", "docs:b@2": "/Support/Refunds", "docs:c@1": "/Ops/Carriers" },
          answer: {
            claims: [
              { text: "Refund policy, updated in May.", cites: [{ cite: "docs:a@1", title: "Refund policy", span: [0, 5], flags: [] }] },
              { text: "Refund exceptions list.", cites: [{ cite: "docs:b@2", title: "Exceptions", span: [0, 5], flags: [] }] },
              { text: "Carrier contacts.", cites: [{ cite: "docs:c@1", title: "Carriers", span: [0, 5], flags: [] }] },
            ],
          },
        });
      }
      return json({ spaces: [] });
    };
    const { container } = await openHandover(fetcher);
    await brief(" usr_dana ");
    expect(posts).toEqual([{ person: "usr_dana" }]);
    const refunds = screen.getByRole("region", { name: "/Support/Refunds" });
    expect(refunds.querySelectorAll("li")).toHaveLength(2);
    expect(screen.getByRole("region", { name: "/Ops/Carriers" }).textContent).toContain("Carrier contacts.");
    expect(screen.getByRole("note").textContent).toContain("2 of their documents are not in this brief");
    expect(screen.getByRole("note").textContent).toContain("move access");
    expect(container.textContent).toContain("1 line was left out");
    expect(await violations(container)).toEqual([]);
  });

  it("shows the policy's reason when the asker may not brief that person", async () => {
    const fetcher: Fetch = async (u) =>
      u === "/v1/handover"
        ? json(
            {
              error: {
                code: "EPERM",
                message: "Only the person, their manager or a steward of a space they are in can have their handover brief, and only of someone in the directory.",
                trace: "",
                remedy: "",
              },
            },
            403,
          )
        : json({ spaces: [] });
    const { container } = await openHandover(fetcher);
    await brief("usr_ann");
    expect(screen.getByRole("alert").textContent).toContain("Only the person, their manager or a steward");
    expect(screen.queryByRole("article")).toBeNull();
    expect(await violations(container)).toEqual([]);
  });
});

// SB-2106
describe("handover by name", () => {
  it("finds the people the asker may brief and briefs the one picked", async () => {
    const posts: unknown[] = [];
    const queries: string[] = [];
    const fetcher: Fetch = async (u, init) => {
      if (u.startsWith("/v1/handover/people")) {
        queries.push(u);
        return json({
          items: [
            { id: "usr_dana", name: "dana@acme.example", email: "dana@acme.example", title: "Support lead" },
            { id: "usr_dan", name: "dan.r", email: "dan.r@acme.example" },
          ],
        });
      }
      if (u === "/v1/handover") {
        posts.push(JSON.parse(String(init?.body)));
        return json({ trace: "tr_h2", person: "usr_dan", dropped: 0, hidden: 0, answer: { claims: [] } });
      }
      return json({ spaces: [] });
    };
    const { container } = await openHandover(fetcher);
    await brief("dan");
    expect(queries).toEqual(["/v1/handover/people?q=dan"]);
    expect(posts).toEqual([]);
    expect(screen.getByRole("list", { name: "People you may brief" }).textContent).toContain("Support lead");
    expect(await violations(container)).toEqual([]);
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Brief dan.r (dan.r@acme.example)" }));
    });
    expect(posts).toEqual([{ person: "usr_dan" }]);
  });

  it("briefs at once when one person matches, and says when no one does", async () => {
    const posts: unknown[] = [];
    let items: unknown[] = [{ id: "usr_dana", name: "dana", email: "dana@acme.example" }];
    const fetcher: Fetch = async (u, init) => {
      if (u.startsWith("/v1/handover/people")) return json({ items });
      if (u === "/v1/handover") {
        posts.push(JSON.parse(String(init?.body)));
        return json({ trace: "tr_h3", person: "usr_dana", dropped: 0, hidden: 0, answer: { claims: [] } });
      }
      return json({ spaces: [] });
    };
    await openHandover(fetcher);
    await brief("dana@acme.example");
    expect(posts).toEqual([{ person: "usr_dana" }]);
    items = [];
    await brief("zed");
    expect(screen.getByText("No one you may brief matches that.")).toBeTruthy();
  });
});
