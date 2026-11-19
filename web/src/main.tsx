import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { App } from "./App";
import { ApproverView } from "./Approver";
import { AdminConsole } from "./Admin";
import "./styles.css";

const root = document.getElementById("root");
const params = new URLSearchParams(window.location.search);
const action = params.get("action");
const admin = params.has("admin");
if (root) {
  createRoot(root).render(
    <StrictMode>
      {action ? <ApproverView id={action} /> : admin ? <AdminConsole /> : <App />}
    </StrictMode>,
  );
}
