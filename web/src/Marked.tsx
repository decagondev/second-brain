import { t } from "./i18n";
import type { Cite } from "./api";

// Marked as AI (SB-4706, LAW-28): anything an agent wrote says which agent and version
// wrote it wherever it is shown.

export interface Mark {
  agent: string;
  version?: string;
}

export function markLine(agent?: string, version?: string): string {
  if (!agent) return "";
  return version
    ? t("Written by {agent}, version {version}", { agent, version })
    : t("Written by {agent}", { agent });
}

// citeMark is a cited note's mark; "" for a document a person wrote.
export function citeMark(
  c: Pick<Cite, "generated_by" | "approved_by" | "edited">,
): string {
  if (!c.generated_by) return "";
  const at = c.generated_by.indexOf("@");
  const line =
    at < 0
      ? markLine(c.generated_by)
      : markLine(c.generated_by.slice(0, at), c.generated_by.slice(at + 1));
  if (!c.approved_by) return line;
  return c.edited
    ? t("{mark}, edited and approved by {who}", {
        mark: line,
        who: c.approved_by,
      })
    : t("{mark}, approved by {who}", { mark: line, who: c.approved_by });
}

export function MarkLine({
  agent,
  version,
}: {
  agent?: string;
  version?: string;
}) {
  const line = markLine(agent, version);
  return line ? <p className="meta mark">{line}</p> : null;
}
