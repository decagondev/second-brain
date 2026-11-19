import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { App } from "./App";
import { ApproverView } from "./Approver";
import "./styles.css";

const root = document.getElementById("root");
const action = new URLSearchParams(window.location.search).get("action");
if (root) {
  createRoot(root).render(
    <StrictMode>
      {action ? <ApproverView id={action} /> : <App />}
    </StrictMode>,
  );
}
