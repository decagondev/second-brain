import {
  act,
  cleanup,
  fireEvent,
  render,
  screen,
} from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import { App } from "./App";
import type { Fetch } from "./api";

afterEach(cleanup);

// SB-2901
describe("short conversations", () => {
  it("says when the earliest questions weren't used and starts fresh", async () => {
    const bodies: string[] = [];
    const fetcher: Fetch = async (u, init) => {
      if (u.startsWith("/v1/ask")) {
        bodies.push(String(init?.body ?? ""));
        return new Response(
          JSON.stringify({
            trace: "tr_1",
            release: "r",
            answer: { claims: [{ text: "x", cites: [], uncited: true }] },
            refused: 0,
            notice: null,
            conversation: "cv_1",
            earlier_dropped: 2,
          }),
          { status: 200 },
        );
      }
      return new Response(
        JSON.stringify({ items: [], results: [], spaces: [] }),
        { status: 200 },
      );
    };
    render(<App fetcher={fetcher} />);
    await act(async () => {});
    fireEvent.change(screen.getByLabelText("Ask a question"), {
      target: { value: "tenth question?" },
    });
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Ask" }));
    });
    expect(
      screen.getByText(/the earliest 2 questions weren't used for this answer/),
    ).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "Start fresh" }));
    expect(screen.queryByText(/weren't used/)).toBeNull();
    fireEvent.change(screen.getByLabelText("Ask a question"), {
      target: { value: "new topic?" },
    });
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Ask" }));
    });
    expect(bodies[bodies.length - 1]).not.toContain("cv_1");
  });
});
