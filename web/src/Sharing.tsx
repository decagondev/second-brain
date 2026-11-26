import { useState } from "react";
import type { FormEvent } from "react";
import { AskError, audience, confirmShared } from "./api";
import type { Audience, Fetch } from "./api";
import { errorText, t, tn } from "./i18n";

// Who can see this (SB-5404): beside a cite, a note or a shared answer, how many people can
// open it now, asked of the source each time, and only the groups the reader may see.
export function WhoCanSee({ docs, fetcher }: { docs: string[]; fetcher?: Fetch }) {
  const [open, setOpen] = useState(false);
  const [a, setA] = useState<Audience>();
  const [err, setErr] = useState("");
  const show = async () => {
    if (open) {
      setOpen(false);
      return;
    }
    setOpen(true);
    setA(undefined);
    setErr("");
    try {
      setA(await audience(docs, fetcher));
    } catch (e) {
      setErr(e instanceof AskError ? errorText(e.error).message : t("This is unavailable right now."));
    }
  };
  return (
    <span className="audience">
      <button type="button" className="link" aria-expanded={open} onClick={() => void show()}>
        {t("Who can see this")}
      </button>
      {open && (
        <span className="audience-body" role="status">
          {err && <span className="notice">{err}</span>}
          {a && (
            <span>
              {tn(a.people, "{n} person can open this now", "{n} people can open this now")}
              {a.anyone && <> {t("Anyone with the link can open it.")}</>}
              {a.groups.length > 0 && <> {t("Through: {groups}", { groups: a.groups.join(", ") })}</>}
              {a.other_groups > 0 && <> {tn(a.other_groups, "and {n} other group", "and {n} other groups")}</>}
            </span>
          )}
        </span>
      )}
    </span>
  );
}

// SharedOnPurpose is the owner's confirmation, with a reason, that a document shared too
// widely is meant to be (SB-5402). It is audited and lapses when the document or its sharing changes.
export function SharedOnPurpose({ cite, fetcher }: { cite: string; fetcher?: Fetch }) {
  const [reason, setReason] = useState("");
  const [done, setDone] = useState(false);
  const [err, setErr] = useState("");
  const submit = async (e: FormEvent) => {
    e.preventDefault();
    setErr("");
    try {
      await confirmShared(cite, reason.trim(), fetcher);
      setDone(true);
    } catch (x) {
      setErr(x instanceof AskError ? errorText(x.error).message : t("This is unavailable right now."));
    }
  };
  if (done) return <span className="meta">{t("Confirmed as shared on purpose.")}</span>;
  return (
    <form className="inline" onSubmit={(e) => void submit(e)}>
      <input aria-label={t("Why it is shared this widely")} value={reason} onChange={(e) => setReason(e.target.value)} maxLength={200} required />
      <button type="submit">{t("Shared on purpose")}</button>
      {err && <span className="notice">{err}</span>}
    </form>
  );
}
