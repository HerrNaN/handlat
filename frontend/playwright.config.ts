import { defineConfig } from "@playwright/test";
import { fileURLToPath } from "node:url";

export default defineConfig({
  testDir: "./e2e",
  use: {
    baseURL: "http://localhost:8081",
    browserName: "chromium",
  },
  webServer: [
    {
      command: "pnpm dev --host 127.0.0.1 --port 5183 --strictPort",
      cwd: fileURLToPath(new URL(".", import.meta.url)),
      url: "http://127.0.0.1:5183",
      reuseExistingServer: false,
    },
    {
      command: "go run ./cmd/handlat",
      cwd: fileURLToPath(new URL("../backend/", import.meta.url)),
      env: {
        PORT: "8081",
        DATABASE_PATH: ":memory:",
        WEBAUTHN_RP_ID: "localhost",
        WEBAUTHN_ORIGIN: "http://localhost:8081",
        EXTERNAL_WEB_SERVER_ADDRESS: "http://127.0.0.1:5183",
      },
      url: "http://localhost:8081/api/auth/me",
      reuseExistingServer: false,
      timeout: 120_000,
    },
  ],
});
