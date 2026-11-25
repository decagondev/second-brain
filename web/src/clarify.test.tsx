import { act, cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import { App } from "./App";
import type { Fetch } from "./api";

afterEach(cleanup);

// SB-4901
describe("ask before guessing", () => {
  it("shows one question and asks again with the choice", async () => {
    const bodies: string[] = [];
    const fetcher: Fetch = async (u, init) => {
      if (u.startsWith("/v1/ask")) {
        const body = String(init?.body ?? "");
        bodies.push(body);
        const chose = body.includes('"choice"');
        return new Response(
          JSON.stringify({
            trace: chose ? "tr_1" : "",
            release: "r",
            answer: { claims: chose ? [{ text: "Accounts payable closes on the 3rd.", cites: [], uncited: true }] : [] },
            refused: 0,
            notice: null,
            conversation: chose ? "cv_1" : "",
            clarify: chose ? undefined : { kind: "term", question: 'Which "ap" do you mean?', choices: ["AP as eng uses it: access point", "AP as finance uses it: accounts payable", "just answer"] },
          }),
          { status: 200 },
        );
      }
      return new Response(JSON.stringify({ items: [], results: [], spaces: [] }), { status: 200 });
    };
    render(<App fetcher={fetcher} />);
    await act(async () => {});
    fireEvent.change(screen.getByLabelText("Ask a question"), { target: { value: "When does AP close?" } });
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Ask" }));
    });
    expect(screen.getByText('Which "ap" do you mean?')).toBeTruthy();
    expect(screen.getByRole("button", { name: "Just answer" })).toBeTruthy();
    expect(screen.queryByRole("button", { name: /Helpful/ })).toBeNull();
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "AP as finance uses it: accounts payable" }));
    });
    expect(bodies[0]).toContain('"clarify":true');
    expect(bodies[1]).toContain('"choice":"AP as finance uses it: accounts payable"');
    expect(screen.queryByText('Which "ap" do you mean?')).toBeNull();
    expect(screen.getByText(/Accounts payable closes on the 3rd/)).toBeTruthy();
  });
});
