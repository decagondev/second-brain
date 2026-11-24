import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { App } from "./App";
import { ApproverView } from "./Approver";
import { AdminConsole } from "./Admin";
import { StewardView } from "./Steward";
import { Localised } from "./Language";
import { initialLang, setLang } from "./i18n";
import "./styles.css";

const root = document.getElementById("root");
const params = new URLSearchParams(window.location.search);
const action = params.get("action");
const admin = params.has("admin");
const steward = params.get("steward");
// A share link, /q/sh_… (sb-api(3), Links for people): the app opens on the shared question.
const share = /^\/q\/(sh_[0-9a-f]{32})$/.exec(window.location.pathname)?.[1];
setLang(initialLang());
if (root) {
  createRoot(root).render(
    <StrictMode>
      <Localised
        render={() =>
          action ? <ApproverView id={action} /> : admin ? <AdminConsole /> : steward && steward !== "me" ? <StewardView space={steward} /> : <App share={share} tab={steward === "me" ? "questions" : (params.get("tab") ?? undefined)} q={params.get("q") ?? undefined} />
        }
      />
    </StrictMode>,
  );
}
