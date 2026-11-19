/// <reference types="vitest" />
import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

// The cell serves dist/ at / (sb-ask(3), The web UI). In development, /v1 is proxied to
// a local cell.
export default defineConfig({
  plugins: [react()],
  build: { outDir: "dist", sourcemap: false, assetsInlineLimit: 0 },
  server: { proxy: { "/v1": "http://localhost:8080" } },
  test: { environment: "jsdom", globals: true },
});
