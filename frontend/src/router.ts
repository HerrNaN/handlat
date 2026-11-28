import { createRouter, Navigation } from "sv-router";
import { get } from "svelte/store";

import { hasActiveSession, session } from "./lib/user/session";

// Use sv-router's supported code splitting to load pages without eager
// component/router import cycles.
// sv-router's returned helpers are designed to be destructured and do not use `this`.
// oxlint-disable-next-line typescript/unbound-method
export const { p, navigate, isActive, route } = createRouter({
  "/invitations/:token": () => import("./lib/list/Invitation.svelte"),
  "/login": () => import("./lib/user/Authenticate.svelte"),
  "/connect": () => import("./lib/user/ConnectApp.svelte"),
  "/user": () => import("./lib/user/User.svelte"),
  "/recipes": {
    "/": () => import("./lib/recipe/Recipes.svelte"),
    "/:recipeId": () => import("./lib/recipe/RecipeRoute.svelte"),
  },
  "/lists": {
    "/": () => import("./lib/list/Lists.svelte"),
    "/:id": () => import("./lib/list/List.svelte"),
  },
  "*": () => import("./CatchAll.svelte"),
  layout: () => import("./Layout.svelte"),
  hooks: {
    // Anonymous visitors use one screen for both passkey actions;
    // logged-in visitors skip authentication altogether.
    beforeLoad: async ({ pathname }) => {
      if (pathname === "/connect") {
        return;
      }
      const isLoggedIn = await hasActiveSession();
      const isAuthPage = pathname === "/login";
      if (!isAuthPage && !pathname.startsWith("/invitations/") && !isLoggedIn) {
        throw await navigate("/login", { replace: true });
      }

      if (isAuthPage && isLoggedIn) {
        throw await navigate("/user", { replace: true });
      }

      const isRecipePage = pathname === "/recipes" || pathname.startsWith("/recipes/");
      if (isRecipePage && !get(session)?.recipesEnabled) {
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
