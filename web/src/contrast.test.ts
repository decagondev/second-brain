import { describe, expect, it } from "vitest";
import css from "./styles.css?raw";

// Colour contrast (SB-6107, WCAG 2.2 AA 1.4.3 and 1.4.11): jsdom computes no colour, so the
// stylesheet's tokens are checked here, in the light and the dark scheme. Text needs 4.5:1
// against what it sits on; the focus ring, borders that mean something and the flag 3:1.

function tokens(block: string): Record<string, string> {
  const out: Record<string, string> = {};
  for (const m of block.matchAll(/--([a-z-]+):\s*(#[0-9a-fA-F]{6})/g)) out[m[1]!] = m[2]!;
  return out;
}

function lum(hex: string): number {
  const c = [1, 3, 5].map((i) => parseInt(hex.slice(i, i + 2), 16) / 255).map((v) => (v <= 0.03928 ? v / 12.92 : ((v + 0.055) / 1.055) ** 2.4));
  return 0.2126 * c[0]! + 0.7152 * c[1]! + 0.0722 * c[2]!;
}

function ratio(a: string, b: string): number {
  const [x, y] = [lum(a), lum(b)].sort((p, q) => q - p) as [number, number];
  return (x + 0.05) / (y + 0.05);
}

const light = tokens(css.slice(css.indexOf(":root"), css.indexOf("@media (prefers-color-scheme: dark)")));
const dark = { ...light, ...tokens(css.slice(css.indexOf("@media (prefers-color-scheme: dark)"))) };

describe("colour contrast", () => {
  for (const [scheme, t] of [["light", light], ["dark", dark]] as const) {
    it(`meets AA in the ${scheme} scheme`, () => {
      const text: [string, string][] = [["fg", "bg"], ["muted", "bg"], ["accent", "bg"], ["fg", "chip-bg"], ["accent", "chip-bg"], ["fg", "uncited-bg"], ["flag", "bg"]];
      for (const [f, b] of text) {
        expect(ratio(t[f]!, t[b]!), `${f} on ${b}`).toBeGreaterThanOrEqual(4.5);
      }
      for (const [f, b] of [["focus", "bg"], ["uncited-border", "uncited-bg"]] as [string, string][]) {
        expect(ratio(t[f]!, t[b]!), `${f} on ${b}`).toBeGreaterThanOrEqual(3);
      }
    });
  }
});
