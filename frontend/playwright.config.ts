import { fileURLToPath } from "node:url";

import { defineConfig } from "@playwright/test";

export default defineConfig({
  testDir: "./e2e",
  use: {
    baseURL: "http://localhost:8080",
    browserName: "chromium",
    locale: "en-US",
  },
  webServer: {
    command: "pnpm --dir ../frontend build --emptyOutDir=false && go run ./cmd/handlat",
    cwd: fileURLToPath(new URL("../backend/", import.meta.url)),
    env: {
      PORT: "8080",
      DATABASE_PATH: ":memory:",
      WEBAUTHN_RP_ID: "localhost",
      WEBAUTHN_ORIGIN: "http://localhost:8080",
    },
    url: "http://localhost:8080/api/auth/me",
    reuseExistingServer: false,
    timeout: 120_000,
  },
});
