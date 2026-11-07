import { svelte } from "@sveltejs/vite-plugin-svelte";
import { defineConfig } from "vite";

// https://vite.dev/config/
export default defineConfig({
  build: {
    outDir: "../backend/internal/app/web/dist",
    emptyOutDir: true,
  },
  plugins: [svelte()],
});
