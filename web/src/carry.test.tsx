import {
  act,
  cleanup,
  fireEvent,
  render,
  screen,
} from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import {
  KeepOnEveryDevice,
  LeftOff,
  MyConversations,
  SharedConversation,
} from "./Conversations";
import type { Fetch } from "./api";

afterEach(() => {
  cleanup();
  window.localStorage.clear();
});

const json = (v: unknown) => new Response(JSON.stringify(v), { status: 200 });

describe("carry the conversation (SB-5802 to SB-5805)", () => {
  it("lists chat conversations only when opened, and continues one checked as you", async () => {
    const asked: string[] = [];
    let continued = "";
    const fetcher: Fetch = async (url) => {
      asked.push(url);
      if (url === "/v1/me/conversations")
        return json({
          sync: false,
          conversations: [
            {
              id: "cv_t1",
              client: "chat",
              first: "Cutoff?",
              turns: 2,
              at: "2026-11-27T10:00:00Z",
            },
          ],
        });
      return json({
        id: "cv_t1",
        turns: [
          {
            question: "Cutoff?",
            trace: "tr_1",
            at: "",
            claims: [
              {
                text: "3pm",
                cites: [
                  { cite: "docs:dock@1", title: "Dock hours", flags: [] },
                ],
              },
            ],
          },
          { question: "Fridays?", trace: "tr_2", at: "", unavailable: true },
        ],
      });
    };
    const { container } = render(
      <MyConversations
        fetcher={fetcher}
        onContinue={(id) => (continued = id)}
      />,
    );
    expect(asked).toEqual([]);
    const d = container.querySelector("details")!;
    await act(async () => {
      d.open = true;
      fireEvent(d, new Event("toggle"));
    });
    expect(screen.getByText(/Cutoff\?/)).toBeTruthy();
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Continue here" }));
    });
    expect(continued).toBe("cv_t1");
    expect(screen.getByText("No longer available to you")).toBeTruthy();
    expect(screen.getByText(/3pm/)).toBeTruthy();
  });

  it("keeps conversations on every device only when turned on", async () => {
    const bodies: string[] = [];
    const fetcher: Fetch = async (_url, init) => {
      bodies.push(String(init?.body));
      return json({ sync: bodies.length === 1, conversations: [] });
    };
    render(<KeepOnEveryDevice fetcher={fetcher} />);
    const box = screen.getByRole("checkbox") as HTMLInputElement;
    expect(box.checked).toBe(false);
    await act(async () => {
      fireEvent.click(box);
    });
    expect(bodies[0]).toBe('{"on":true}');
    expect(box.checked).toBe(true);
  });

  it("shows a shared conversation's answers you can open, and counts the rest", async () => {
    const fetcher: Fetch = async () =>
      json({
        turns: [
          {
            question: "Cutoff?",
            trace: "tr_1",
            at: "",
            claims: [{ text: "3pm", cites: [], uncited: true }],
          },
        ],
        left: 2,
        label: "confidential",
      });
    await act(async () => {
      render(
        <SharedConversation
          fetcher={fetcher}
          id="sh_00000000000000000000000000000000"
        />,
      );
    });
    expect(screen.getByText("2 answers not shared with you.")).toBeTruthy();
    expect(screen.getByText(/Confidential/)).toBeTruthy();
  });

  it("starts a new conversation with where you left off, until Not now", async () => {
    let n = 0;
    const fetcher: Fetch = async () => {
      n++;
      return json({
        items: [
          {
            kind: "question",
            id: "qs_1",
            title: "Who approves refunds?",
            link: "/?tab=questions",
          },
        ],
      });
    };
    await act(async () => {
      render(<LeftOff fetcher={fetcher} start={true} />);
    });
    expect(screen.getByText("Who approves refunds?")).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "Not now" }));
    expect(screen.queryByText("Who approves refunds?")).toBeNull();
    cleanup();
    await act(async () => {
      render(<LeftOff fetcher={fetcher} start={true} />);
    });
    expect(n).toBe(1);
  });
});
