import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { App } from "./App";
import { ApproverView } from "./Approver";
import { AdminConsole } from "./Admin";
import { StewardView } from "./Steward";
import "./styles.css";

const root = document.getElementById("root");
const params = new URLSearchParams(window.location.search);
const action = params.get("action");
const admin = params.has("admin");
const steward = params.get("steward");
if (root) {
  createRoot(root).render(
    <StrictMode>
      {action ? <ApproverView id={action} /> : admin ? <AdminConsole /> : steward ? <StewardView space={steward} /> : <App />}
    </StrictMode>,
  );
}
