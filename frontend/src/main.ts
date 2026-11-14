import * as Sentry from "@sentry/svelte";
import { mount } from "svelte";

import "@awesome.me/webawesome/dist/styles/themes/default.css";

import App from "./App.svelte";
import { initializeTheme } from "./lib/theme";

import "./theme.css";

const cleanupTheme = initializeTheme();
import.meta.hot?.dispose(cleanupTheme);

export const observer = Sentry.init({
  dsn: "https://66db7a76007a4546ba6f1c5675959347@app.glitchtip.com/28832",
  integrations: [Sentry.captureConsoleIntegration({ levels: ["error"] })],
  environment: window.location.host,
  dataCollection: {
    cookies: false,
    databaseQueryData: false,
    frameContextLines: 5,
    genAI: { inputs: false, outputs: false },
    graphQL: { document: false, variables: false },
    httpBodies: [],
    httpHeaders: false,
    queues: false,
    stackFrameVariables: false,
    urlQueryParams: false,
    userInfo: false,
  },
});

const app = mount(App, {
  target: document.getElementById("app")!,
});

export default app;
