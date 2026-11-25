import { act, cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import { AtHint, atWord, withoutAt } from "./AtHint";
import type { About, Fetch } from "./api";

afterEach(cleanup);

describe("@ in the ask box (SB-5207)", () => {
  it("reads the trailing @word", () => {
    expect(atWord("what about @fin")).toBe("fin");
    expect(atWord("mail me@acme.example")).toBeUndefined();
    expect(withoutAt("what about @fin")).toBe("what about");
  });

  it("offers spaces, people and documents the endpoints return, and picks a document", async () => {
    const calls: string[] = [];
    const f: Fetch = async (u) => {
      calls.push(u);
      if (u.startsWith("/v1/search")) return new Response(JSON.stringify({ results: [{ cite: "docs:fin/close.md@3", title: "Close checklist", snippet: "", flags: [] }] }), { status: 200 });
      if (u.endsWith("/people")) return new Response(JSON.stringify({ space: "finance", stewards: [{ email: "fiona@acme.example" }], owners: [{ email: "zed@acme.example", documents: 1 }], members: 3 }), { status: 200 });
      return new Response("{}", { status: 200 });
    };
    let picked: About | undefined;
    render(
      <AtHint
        question="when is @fi"
        space="finance"
        spaces={[{ name: "finance" }, { name: "ops" }] as never}
        fetcher={f}
        onSpace={() => undefined}
        onDocument={(d) => (picked = d)}
        onPerson={() => undefined}
      />,
    );
    await act(async () => {});
    expect(screen.getByRole("button", { name: "Ask in finance" })).toBeTruthy();
    expect(screen.getByRole("button", { name: "Documents of fiona@acme.example" })).toBeTruthy();
    expect(screen.queryByRole("button", { name: "Documents of zed@acme.example" })).toBeNull();
    fireEvent.click(screen.getByRole("button", { name: "Ask about Close checklist" }));
    expect(picked).toEqual({ key: "docs:fin/close.md", name: "Close checklist" });
    expect(calls.some((c) => c.startsWith("/v1/search?q=fi"))).toBe(true);
  });
});
