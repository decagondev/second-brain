import { act, cleanup, fireEvent, render, screen } from "@testing-library/react";
import axe from "axe-core";
import { afterEach, describe, expect, it } from "vitest";
import { AnswerView } from "./Answer";
import { App } from "./App";
import { Localised } from "./Language";
import type { AskResponse, Claim, Fetch } from "./api";
import { catalog } from "./i18n/catalog";
import { errorText, initialLang, lang, setLang } from "./i18n";
import type { Lang } from "./i18n";

// The UI in the company's language (SB-2301): every screen in English, German and French,
// an uncited claim marked in each (LAW-14), and the cell's errors translated by code.
afterEach(() => {
  cleanup();
  setLang("en");
  window.localStorage.clear();
  document.documentElement.lang = "en";
});

const cited: Claim = { text: "Refunds over 500 need a team lead's approval.", cites: [{ cite: "docs:4f2a9c11@17", title: "Refund policy", span: [10, 80], flags: [] }] };
const uncited: Claim = { text: "Ask finance for exceptions.", cites: [], uncited: true };
const answer: AskResponse = { trace: "tr_1", release: "rel_x", answer: { claims: [cited, uncited] }, refused: 2, notice: "most_uncited", conversation: "cv_1" };

async function violations(container: HTMLElement) {
  const r = await axe.run(container, { rules: { "color-contrast": { enabled: false } } });
  return r.violations.map((v) => `${v.id}: ${v.nodes.map((n) => n.html).join(" ")}`);
}

const word = (key: string, l: Lang) => (l === "en" ? key : catalog[key]![l]);

describe.each(["en", "de", "fr"] as Lang[])("the UI in %s", (l) => {
  it("marks an uncited claim with a word in the language, never colour alone (LAW-14)", () => {
    setLang(l);
    const { container } = render(<AnswerView answer={answer} />);
    const marked = container.querySelector("li.claim.uncited");
    expect(marked?.textContent).toContain(word("Uncited", l));
    expect(marked?.getAttribute("title")).toBe(word("No source you can open supports this.", l));
    expect(container.querySelectorAll("li.claim.cited a.chip").length).toBe(1);
    expect(container.textContent).toContain(word("{n} sources were not shown because you can't open them.", l).replace("{n}", "2"));
  });

  it("has no axe violations in the app with an answer and an error", async () => {
    setLang(l);
    let n = 0;
    const fetcher: Fetch = async (url) => {
      if (url === "/v1/notice") return new Response("{}", { status: 200 }); // SB-3207
      if (url === "/v1/spaces") return new Response(JSON.stringify({ spaces: [] }), { status: 200 });
      if (url.startsWith("/v1/changes")) return new Response(JSON.stringify({ items: [] }), { status: 200 });
      if (url.startsWith("/v1/search")) return new Response(JSON.stringify({ trace: "tr_s", release: "rel_x", results: [], refused: 0 }), { status: 200 }); // SB-2304
      n++;
      if (n === 1) return new Response(JSON.stringify(answer), { status: 200 });
      return new Response(JSON.stringify({ error: { code: "EKILLED", message: "Answers are switched off for now.", trace: "tr_9", remedy: "Wait for the operator." } }), { status: 503 });
    };
    const { container } = render(<Localised render={() => <App fetcher={fetcher} />} />);
    const box = screen.getByLabelText(word("Ask a question", l));
    for (const q of ["refund limit?", "and since when?"]) {
      fireEvent.change(box, { target: { value: q } });
      await act(async () => {
        fireEvent.click(screen.getByRole("button", { name: word("Ask", l) }));
      });
    }
    const alert = screen.getByRole("alert").textContent ?? "";
    // In English the cell's own words; in another language the code's.
    expect(alert).toContain(l === "en" ? "Answers are switched off for now." : word("This is switched off for now.", l));
    expect(alert).toContain("tr_9");
    expect(document.documentElement.lang).toBe(l);
    expect(await violations(container)).toEqual([]);
  });
});

describe("choosing the language", () => {
  it("starts in the manifest's language, from the page", () => {
    document.documentElement.lang = "fr";
    expect(initialLang()).toBe("fr");
    document.documentElement.lang = "xx";
    expect(initialLang()).toBe("en");
  });

  it("keeps a person's choice in their browser and switches the screen in place", async () => {
    document.documentElement.lang = "de";
    setLang(initialLang());
    const fetcher: Fetch = async (url) => new Response(JSON.stringify(url === "/v1/spaces" ? { spaces: [] } : { items: [] }), { status: 200 });
    render(<Localised render={() => <App fetcher={fetcher} />} />);
    fireEvent.change(screen.getByLabelText(word("Ask a question", "de")), { target: { value: "kept" } });
    expect(screen.getByRole("tab", { name: "Antwort" })).toBeTruthy();
    fireEvent.change(screen.getByLabelText("Sprache"), { target: { value: "fr" } });
    expect(lang()).toBe("fr");
    expect(screen.getByRole("tab", { name: "Réponse" })).toBeTruthy();
    expect((screen.getByLabelText(word("Ask a question", "fr")) as HTMLTextAreaElement).value).toBe("kept"); // state kept
    expect(window.localStorage.getItem("sb.lang")).toBe("fr");
    document.documentElement.lang = "de";
    expect(initialLang()).toBe("fr"); // the person's choice beats the default
  });

  it("translates every error code, and falls back for one it doesn't know", () => {
    setLang("de");
    expect(errorText({ code: "ESPACE", message: "Not a member of finance." }).message).toBe(catalog["You aren't a member of this space."]!.de);
    expect(errorText({ code: "ERATE", message: "", retry_after: 12 }).remedy).toContain("12");
    expect(errorText({ code: "EWHAT", message: "?" }).message).toBe(catalog["Something went wrong."]!.de);
    setLang("en");
    expect(errorText({ code: "ESPACE", message: "Not a member of finance.", remedy: "Ask its stewards." })).toEqual({ message: "Not a member of finance.", remedy: "Ask its stewards." });
  });
});
