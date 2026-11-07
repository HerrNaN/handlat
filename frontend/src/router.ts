import { createRouter, Navigation } from "sv-router";

import { hasActiveSession } from "./lib/user/session";

// Use sv-router's supported code splitting to load pages without eager
// component/router import cycles.
export const { p, navigate, isActive, route } = createRouter({
  "/sign-up": () => import("./lib/user/SignUp.svelte"),
  "/sign-in": () => import("./lib/user/SignIn.svelte"),
  "/user": () => import("./lib/user/User.svelte"),
  "/lists": {
    "/": () => import("./lib/list/Lists.svelte"),
    "/:id": () => import("./lib/list/List.svelte"),
  },
  "*": () => import("./CatchAll.svelte"),
  layout: () => import("./Layout.svelte"),
  hooks: {
    // Anonymous visitors can sign up or sign in. Signed-in visitors skip
    // both auth pages. The session is only fetched until we know which it is.
    beforeLoad: async ({ pathname }) => {
      const isLoggedIn = await hasActiveSession();
      const isAuthPage = pathname === "/sign-up" || pathname === "/sign-in";
      if (!isAuthPage && !isLoggedIn) {
        throw await navigate("/sign-in", { replace: true });
      }

      if (isAuthPage && isLoggedIn) {
        throw await navigate("/user", { replace: true });
      }
    },
    // sv-router swallows hook errors otherwise. Redirects are thrown as
    // Navigation values, so only real failures are worth reporting.
    onError: (error) => {
      if (!(error instanceof Navigation)) {
        console.error("Route hook failed", error);
      }
    },
  },
});
