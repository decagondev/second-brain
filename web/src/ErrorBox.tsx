import type { ApiError } from "./api";
import { errorText, t } from "./i18n";

// An error from the cell, in the person's language (sb-errors(7); SB-2301): what went wrong,
// what to do, and the trace to quote.
export function ErrorBox({ error }: { error: ApiError }) {
  const { message, remedy } = errorText(error);
  return (
    <div className="error" role="alert">
      <p>{message}</p>
      {remedy && remedy.toLowerCase() !== message.toLowerCase() && <p>{remedy}</p>}
      {error.trace && <p>{t("Quote trace {trace} to support.", { trace: error.trace })}</p>}
    </div>
  );
}
