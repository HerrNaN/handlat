import tsPlugin from "@typescript-eslint/eslint-plugin";
import tsParser from "@typescript-eslint/parser";
import playwright from "eslint-plugin-playwright";
import svelte from "eslint-plugin-svelte";
import { defineConfig } from "eslint/config";

import svelteConfig from "./svelte.config.js";

export default defineConfig(
  {
    files: ["**/*.svelte", "**/*.svelte.{js,ts}"],
    extends: [svelte.configs.recommended],
    plugins: { "@typescript-eslint": tsPlugin },
    languageOptions: {
      parserOptions: {
        parser: tsParser,
        projectService: true,
        extraFileExtensions: [".svelte"],
        svelteConfig,
      },
    },
    // Oxlint's type-aware engine does not analyze embedded Svelte scripts.
    rules: {
      "@typescript-eslint/no-floating-promises": "error",
      "@typescript-eslint/no-misused-promises": "error",
      "@typescript-eslint/await-thenable": "error",
    },
  },
  {
    files: ["e2e/**/*.ts"],
    languageOptions: {
      parser: tsParser,
    },
    plugins: { playwright },
    rules: {
      "playwright/missing-playwright-await": ["error", { includePageLocatorMethods: true }],
      "playwright/no-focused-test": "error",
      "playwright/no-page-pause": "error",
      "playwright/no-wait-for-timeout": "error",
    },
  },
);
