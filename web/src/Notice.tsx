import { useEffect, useState } from "react";
import { readNotice } from "./api";
import type { CellNotice, Fetch } from "./api";
import { t } from "./i18n";

// How often the notice is read again (sb-ask(3), The notice to everyone).
export const noticeEvery = 5 * 60 * 1000;

// NoticeBanner is the admin's notice at the top of every screen (SB-3207), as plain text,
// read on load and every five minutes, and gone once its end time passes.
export function NoticeBanner({ fetcher }: { fetcher?: Fetch }) {
  const [notice, setNotice] = useState<CellNotice | null>(null);
  useEffect(() => {
    let live = true;
    const read = () => {
      readNotice(fetcher).then((n) => live && setNotice(n));
    };
    read();
    const id = setInterval(read, noticeEvery);
    return () => {
      live = false;
      clearInterval(id);
    };
  }, [fetcher]);
  useEffect(() => {
    if (!notice) return;
    const left = Date.parse(notice.until) - Date.now();
    const id = setTimeout(() => setNotice(null), Math.max(0, Math.min(left, noticeEvery)));
    return () => clearTimeout(id);
  }, [notice]);
  if (!notice) return null;
  return (
    <div className="cell-notice" role="status" aria-label={t("Notice from your admins")}>
      <strong>{t("Notice:")}</strong> {notice.text}
    </div>
  );
}
