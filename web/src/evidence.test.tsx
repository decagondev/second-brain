import { act, cleanup, fireEvent, render, screen } from "@testing-library/react";
import axe from "axe-core";
import { afterEach, describe, expect, it } from "vitest";
import { App } from "./App";
import type { Fetch } from "./api";
import { answerText } from "./Copy";
import { setLang } from "./i18n";

afterEach(() => {
  cleanup();
  setLang("en");
});

async function violations(container: HTMLElement) {
  const r = await axe.run(container, { rules: { "color-contrast": { enabled: false } } });
  return r.violations.map((v) => `${v.id}: ${v.nodes.map((n) => n.html).join(" ")}`);
}

const answer = {
  trace: "tr_1",
  release: "r",
  answer: {
    claims: [
      {
        text: "Carry over up to five days.",
        cites: [{ cite: "docs:hb@3", title: "Staff handbook", section: "Carry-over", span: [120, 480], flags: [] }],
      },
    ],
  },
  refused: 0,
  notice: null,
  conversation: "cv_1",
};

function app(passage: (u: string) => Response, bodies: string[] = []): Fetch {
  return async (u, init) => {
    if (u.startsWith("/v1/ask")) {
      bodies.push(String(init?.body ?? ""));
      return new Response(JSON.stringify(answer), { status: 200 });
    }
    if (u.includes("/passage?")) return passage(u);
    return new Response(JSON.stringify({ items: [], results: [], spaces: [] }), { status: 200 });
  };
}

async function askOne() {
  fireEvent.change(screen.getByLabelText("Ask a question"), { target: { value: "carry over?" } });
  await act(async () => {
    fireEvent.click(screen.getByRole("button", { name: "Ask" }));
  });
}

// SB-3001, SB-3002, SB-3003
describe("show the evidence", () => {
  it("names the section and shows the passage as plain text", async () => {
    const asked: string[] = [];
    const { container } = render(
      <App
        fetcher={app((u) => {
          asked.push(u);
          return new Response(JSON.stringify({ cite: "docs:hb@3", text: "<b>Up to five days</b> carry over.", cut: false }), { status: 200 });
        })}
      />,
    );
    await act(async () => {});
    await askOne();
    expect(screen.getAllByText("Staff handbook › Carry-over").length).toBe(2); // the chip and the printed footnote
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Show passage" }));
    });
    expect(asked).toEqual(["/v1/traces/tr_1/passage?cite=docs%3Ahb%403&start=120"]);
    const region = screen.getByRole("region", { name: "Passage from Staff handbook › Carry-over" });
    expect(region.textContent).toContain("<b>Up to five days</b> carry over.");
    expect(region.querySelector("b")).toBeNull(); // never markup (LAW-09)
    expect(await violations(container)).toEqual([]);
    fireEvent.click(screen.getByRole("button", { name: "Hide passage" }));
    expect(screen.queryByRole("region", { name: /Passage from/ })).toBeNull();
  });

  it("says so when the source can't be opened now, and shows no text", async () => {
    render(
      <App
        fetcher={app(() => new Response(JSON.stringify({ error: { code: "EPERM", message: "You can't open this source now.", trace: "tr_1", remedy: "" } }), { status: 403 }))}
      />,
    );
    await act(async () => {});
    await askOne();
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Show passage" }));
    });
    expect(screen.getByText("You can't open this source now.")).toBeTruthy();
    expect(screen.queryByRole("blockquote")).toBeNull();
  });

  it("says when the version is no longer indexed", async () => {
    render(<App fetcher={app(() => new Response(JSON.stringify({ cite: "docs:hb@3", gone: true }), { status: 200 }))} />);
    await act(async () => {});
    await askOne();
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Show passage" }));
    });
    expect(screen.getByText("This version is no longer indexed.")).toBeTruthy();
  });

  it("asks in the person's language", async () => {
    setLang("de");
    const bodies: string[] = [];
    render(<App fetcher={app(() => new Response("{}"), bodies)} />);
    await act(async () => {});
    fireEvent.change(screen.getByLabelText("Eine Frage stellen"), { target: { value: "Übertrag?" } });
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Fragen" }));
    });
    expect(JSON.parse(bodies[0]!)).toMatchObject({ question: "Übertrag?", language: "de" });
  });

  it("copies the section with the cite", () => {
    expect(answerText("carry over?", answer as never)).toContain("[1] Staff handbook › Carry-over");
  });
});
