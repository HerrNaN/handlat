import tsParser from "@typescript-eslint/parser";
import svelte from "eslint-plugin-svelte";
import { defineConfig } from "eslint/config";

import svelteConfig from "./svelte.config.js";

export default defineConfig({
  files: ["**/*.svelte", "**/*.svelte.{js,ts}"],
  extends: [svelte.configs.recommended],
  languageOptions: {
    parserOptions: {
      parser: tsParser,
      svelteConfig,
    },
  },
});
