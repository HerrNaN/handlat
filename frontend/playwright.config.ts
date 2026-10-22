import { defineConfig } from "@playwright/test";
import { fileURLToPath } from "node:url";

export default defineConfig({
  testDir: "./e2e",
  use: {
    baseURL: "http://localhost:8080",
    browserName: "chromium",
  },
  webServer: [
    {
      command: "pnpm dev",
      cwd: fileURLToPath(new URL(".", import.meta.url)),
      url: "http://localhost:5173",
      reuseExistingServer: false,
    },
    {
      command: "go run ./cmd/handlat",
      cwd: fileURLToPath(new URL("../backend/", import.meta.url)),
      env: {
        PORT: "8080",
        DATABASE_PATH: ":memory:",
        WEBAUTHN_RP_ID: "localhost",
        WEBAUTHN_ORIGIN: "http://localhost:8080",
        EXTERNAL_WEB_SERVER_ADDRESS: "http://localhost:5173",
      },
      url: "http://localhost:8080/api/auth/me",
      reuseExistingServer: false,
      timeout: 120_000,
    },
  ],
});
