import ts from "typescript";
import { catalog } from "./catalog";
import { errorCodes, langs, setLang, t, tn } from "./index";

// Every string the UI shows is in the catalog, in English, German and French (SB-2301):
// these tests read the UI's source and fail on a string outside it.
const sources = import.meta.glob(["../*.tsx", "!../*.test.tsx"], { query: "?raw", import: "default", eager: true }) as Record<string, string>;

// The functions whose first string is a catalog key; tn's second and third are too.
const keyed = new Set(["t", "tj", "msg"]);
// Attributes people read or hear.
const shown = new Set(["aria-label", "placeholder", "title", "alt", "label", "aria-description", "aria-valuetext", "aria-roledescription"]);
// Text that reads like words: two words, or one capitalised one.
// Names that are the same in every language: the product, and commands people type.
const names = new Set(["Second Brain", "sb tenant apply"]);
const prose = (s: string) =>
  !names.has(s) && (/[A-Za-z]{2,}[,.!?'’…:;]*\s+[^\n]*[A-Za-z]/.test(s.trim()) || /^[A-Z][a-z]+[.!?…]?$/.test(s.trim()));

interface Found {
  file: string;
  line: number;
  text: string;
}

function scan(): { outside: Found[]; keys: Found[] } {
  const outside: Found[] = [];
  const keys: Found[] = [];
  for (const [file, src] of Object.entries(sources)) {
    const sf = ts.createSourceFile(file, src, ts.ScriptTarget.Latest, true, ts.ScriptKind.TSX);
    const at = (n: ts.Node, text: string) => ({ file, line: sf.getLineAndCharacterOfPosition(n.getStart()).line + 1, text });
    const calleeOf = (lit: ts.Node): string | undefined => {
      // A key may be one branch of a choice: t(on ? "a" : "b").
      let n = lit;
      while (n.parent && ((ts.isConditionalExpression(n.parent) && n.parent.condition !== n) || ts.isParenthesizedExpression(n.parent))) n = n.parent;
      const p = n.parent;
      if (!p || !ts.isCallExpression(p) || !ts.isIdentifier(p.expression)) return undefined;
      const name = p.expression.text;
      const i = p.arguments.indexOf(n as ts.Expression);
      if (keyed.has(name) && i === 0) return name;
      if (name === "tn" && (i === 1 || i === 2)) return name;
      return undefined;
    };
    const exempt = (n: ts.Node): boolean => {
      if (n.parent && (ts.isPropertyAssignment(n.parent) || ts.isPropertySignature(n.parent)) && n.parent.name === n) return true;
      for (let p: ts.Node | undefined = n.parent; p; p = p.parent) {
        if (ts.isImportDeclaration(p) || ts.isExportDeclaration(p) || ts.isTypeNode(p)) return true;
        if (ts.isJsxAttribute(p)) return !shown.has(p.name.getText());
        if (ts.isBinaryExpression(p) && [ts.SyntaxKind.EqualsEqualsEqualsToken, ts.SyntaxKind.ExclamationEqualsEqualsToken].includes(p.operatorToken.kind)) return true;
        if (ts.isCaseClause(p) || ts.isElementAccessExpression(p)) return true;
        if (ts.isCallExpression(p) && ts.isPropertyAccessExpression(p.expression) && p.expression.expression.getText() === "console") return true;
      }
      return false;
    };
    const visit = (n: ts.Node) => {
      if (ts.isJsxText(n) && /[A-Za-z]/.test(n.getText())) outside.push(at(n, n.getText().trim()));
      if (ts.isStringLiteral(n) || ts.isNoSubstitutionTemplateLiteral(n)) {
        if (calleeOf(n)) keys.push(at(n, n.text));
        else if (prose(n.text) && !exempt(n)) outside.push(at(n, n.text));
      }
      if (ts.isTemplateExpression(n) && !exempt(n)) {
        const parts = [n.head.text, ...n.templateSpans.map((s) => s.literal.text)];
        if (parts.some((p) => /(^|\s)[A-Za-z]{2,}(\s|$)/.test(p) && /\s/.test(p))) outside.push(at(n, n.getText()));
      }
      ts.forEachChild(n, visit);
    };
    visit(sf);
  }
  return { outside, keys };
}

const holes = (s: string) => [...s.matchAll(/\{(\w+)\}/g)].map((m) => m[1]).sort().join(",");

describe("the string catalog", () => {
  afterEach(() => setLang("en"));

  it("holds every string the UI shows", () => {
    const { outside } = scan();
    expect(Object.keys(sources).length).toBeGreaterThan(10);
    expect(outside.map((f) => `${f.file}:${f.line} ${f.text}`)).toEqual([]);
  });

  it("knows every key the UI uses, and the error codes'", () => {
    const { keys } = scan();
    const all = [...keys.map((k) => k.text), ...Object.values(errorCodes).flat(), "Something went wrong.", "Try again."];
    expect(all.filter((k) => !(k in catalog))).toEqual([]);
    const used = new Set(all);
    expect(Object.keys(catalog).filter((k) => !used.has(k))).toEqual([]);
  });

  it("has German and French for every entry, with the same placeholders", () => {
    for (const [key, e] of Object.entries(catalog) as [string, { de: string; fr: string }][]) {
      for (const l of ["de", "fr"] as const) {
        expect(e[l].trim(), `${key} in ${l}`).not.toBe("");
        expect(holes(e[l]), `${key} in ${l}`).toBe(holes(key));
      }
    }
  });

  it("speaks each language and counts by its rule", () => {
    expect(langs.map((l) => l.id)).toEqual(["en", "de", "fr"]);
    setLang("de");
    expect(t("Ask")).toBe(catalog.Ask!.de);
    expect(tn(0, "{n} source was not shown because you can't open it.", "{n} sources were not shown because you can't open them.")).toContain("0");
    setLang("fr");
    expect(tn(0, "{n} source was not shown because you can't open it.", "{n} sources were not shown because you can't open them.")).toBe(
      catalog["{n} source was not shown because you can't open it."]!.fr.replace("{n}", "0"),
    );
  });
});
