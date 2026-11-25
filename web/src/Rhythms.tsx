import { useEffect, useState } from "react";
import type { Fetch } from "./api";
import { call } from "./Steward";
import { t, tn } from "./i18n";

// Ways of working (SB-4801 to SB-4804): a space's rhythms in flight and its projects, on
// the space's home. Every step is a draft in its approver's queue; nothing here publishes.

export interface RhythmStep {
  day: number;
  kind: string;
  ask: string;
  approver: string;
  due?: string;
  state: string;
  why?: string;
  yours?: boolean;
}

export interface Rhythm {
  id: string;
  rhythm: string;
  space: string;
  subject: string;
  subject_kind: string;
  state: string;
  why?: string;
  mine?: boolean;
  steps: RhythmStep[];
}

export interface Project {
  id: string;
  name: string;
  lead: boolean;
  state: string;
  goal?: string;
  decisions: { text: string; cite: string }[];
  hidden: number;
  pending?: string[];
  collection: { id: string; closed?: string };
}

// RhythmsCard lists the rhythms the reader may see, with Approve on a proposed plan and
// Stop on a running rhythm the reader may stop.
export function RhythmsCard({ space, fetcher = fetch }: { space: string; fetcher?: Fetch }) {
  const [items, setItems] = useState<Rhythm[]>([]);
  const [status, setStatus] = useState<string>();
  const base = `/v1/spaces/${encodeURIComponent(space)}/rhythms`;
  async function load() {
    try {
      const d = await call(fetcher, "GET", base);
      setItems(((d.items as Rhythm[]) ?? []).filter((r) => r.state === "running" || r.state === "proposed"));
    } catch {
      setItems([]);
    }
  }
  useEffect(() => {
    void load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [space, fetcher]);
  async function act(id: string, what: "approve" | "stop") {
    try {
      await call(fetcher, "POST", `/v1/rhythms/${encodeURIComponent(id)}/${what}`);
      setStatus(what === "approve" ? t("Approved.") : t("Stopped."));
      await load();
    } catch {
      setStatus(t("That didn't work."));
    }
  }
  if (items.length === 0) return null;
  return (
    <section aria-label={t("Rhythms")} className="start">
      <h2>{t("Rhythms")}</h2>
      <ul>
        {items.map((r) => {
          const done = r.steps.filter((s) => s.state === "done").length;
          const next = r.steps.find((s) => s.state === "waiting");
          return (
            <li key={r.id}>
              <strong>{r.rhythm}</strong>{" "}
              <span className="meta">
                · {r.state === "proposed" ? t("waiting for its plan to be approved") : tn(done, "{n} step done", "{n} steps done")}
                {next && <> · {t("next: {ask}, {due}", { ask: next.ask, due: next.due ?? "" })}</>}
              </span>{" "}
              {r.state === "proposed" && (
                <button type="button" onClick={() => void act(r.id, "approve")}>
                  {t("Approve")}
                </button>
              )}{" "}
              <button type="button" onClick={() => void act(r.id, "stop")}>
                {t("Stop")}
              </button>
            </li>
          );
        })}
      </ul>
      {status && <p role="status">{status}</p>}
    </section>
  );
}

// ProjectsCard lists the reader's projects in the space, with Start a project, and Close
// project for the lead.
export function ProjectsCard({ space, fetcher = fetch }: { space: string; fetcher?: Fetch }) {
  const [items, setItems] = useState<Project[]>([]);
  const [name, setName] = useState("");
  const [members, setMembers] = useState("");
  const [status, setStatus] = useState<string>();
  const base = `/v1/spaces/${encodeURIComponent(space)}/projects`;
  async function load() {
    try {
      const d = await call(fetcher, "GET", base);
      setItems((d.items as Project[]) ?? []);
    } catch {
      setItems([]);
    }
  }
  useEffect(() => {
    void load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [space, fetcher]);
  async function start() {
    try {
      const ms = members.split(/[\s,]+/).filter(Boolean);
      await call(fetcher, "POST", base, { name: name.trim(), members: ms, documents: [] });
      setName("");
      setMembers("");
      setStatus(t("Started. Your kickoff brief is in your queue."));
      await load();
    } catch {
      setStatus(t("The project couldn't be started."));
    }
  }
  async function close(id: string) {
    try {
      await call(fetcher, "POST", `/v1/projects/${encodeURIComponent(id)}/close`, { lessons: [] });
      setStatus(t("Closed. Your retrospective is in your queue."));
      await load();
    } catch {
      setStatus(t("The project couldn't be closed."));
    }
  }
  return (
    <section aria-label={t("Projects")} className="start">
      <h2>{t("Projects")}</h2>
      {items.length > 0 && (
        <ul>
          {items.map((p) => (
            <li key={p.id}>
              <strong>{p.name}</strong>{" "}
              <span className="meta">
                · {p.state === "closed" ? t("closed") : tn(p.decisions.length, "{n} decision", "{n} decisions")}
                {p.hidden > 0 && <> · {tn(p.hidden, "{n} more you can't open", "{n} more you can't open")}</>}
              </span>{" "}
              {p.lead && p.state === "open" && (
                <button type="button" onClick={() => void close(p.id)}>
                  {t("Close project")}
                </button>
              )}
            </li>
          ))}
        </ul>
      )}
      <form
        onSubmit={(e) => {
          e.preventDefault();
          if (name.trim()) void start();
        }}
      >
        <label>
          {t("Project name")} <input value={name} onChange={(e) => setName(e.target.value)} />
        </label>{" "}
        <label>
          {t("Members (emails)")} <input value={members} onChange={(e) => setMembers(e.target.value)} />
        </label>{" "}
        <button type="submit">{t("Start a project")}</button>
      </form>
      {status && <p role="status">{status}</p>}
    </section>
  );
}
