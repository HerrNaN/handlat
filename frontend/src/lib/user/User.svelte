<script lang="ts">
  import { createMutation } from "@tanstack/svelte-query";

  import { p } from "../../router";
  import { post } from "../api";
  import { setTheme, theme } from "../theme";
  import type { ChangeEvent } from "../utils/events";
  import { session, clearSession } from "./session";
  import "@awesome.me/webawesome/dist/components/button/button.js";
  import "@awesome.me/webawesome/dist/components/icon/icon.js";
  import "@awesome.me/webawesome/dist/components/tooltip/tooltip.js";
  import "@awesome.me/webawesome/dist/components/radio-group/radio-group.js";
  import "@awesome.me/webawesome/dist/components/radio/radio.js";

  const logout = createMutation(() => ({
    mutationKey: ["auth", "logout"],
    mutationFn: () => post("/api/auth/logout"),
    onSuccess: () => {
      clearSession();
      location.assign("/login");
    },
  }));
</script>

<article>
  <header>
    <div>
      <h1>Profile</h1>
      {#if $session}
        <p>You are logged in as: <span class="username">{$session.name}</span>.</p>
      {:else}
        <p>You are not logged in.</p>
        <p><a href={p("/login")}>Login or Create Account</a></p>
      {/if}
    </div>
    {#if $session}
      <wa-button
        id="logout-button"
        aria-label="Log out"
        appearance="plain"
        size="l"
        onclick={() => logout.mutate()}
        disabled={logout.isPending}
      >
        <wa-icon name="arrow-right-from-bracket" aria-hidden="true"></wa-icon>
      </wa-button>
      <wa-tooltip for="logout-button">Log out</wa-tooltip>
    {/if}
  </header>
  {#if logout.isError}
    <p role="alert">Could not log out. Check your connection and try again.</p>
  {/if}
  <section aria-labelledby="appearance-heading">
    <h2 id="appearance-heading">Appearance</h2>
    <wa-radio-group
      label="Theme"
      name="theme"
      orientation="horizontal"
      value={$theme}
      onchange={(event: ChangeEvent) => setTheme(event.target.value)}
    >
      <wa-radio appearance="button" value="light">
        <span><wa-icon name="sun" aria-hidden="true"></wa-icon>Light</span>
      </wa-radio>
      <wa-radio appearance="button" value="dark">
        <span><wa-icon name="moon" aria-hidden="true"></wa-icon>Dark</span>
      </wa-radio>
      <wa-radio appearance="button" value="system">
        <span><wa-icon name="desktop" aria-hidden="true"></wa-icon>System</span>
      </wa-radio>
    </wa-radio-group>
  </section>
</article>

<style>
  article {
    padding: var(--wa-space-l);
    box-sizing: border-box;
    min-height: 100%;
    display: flex;
    flex-direction: column;
    gap: var(--wa-space-xl);
  }

  header {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: var(--wa-space-m);
  }

  header div {
    min-width: 0;
  }

  header wa-button {
    flex-shrink: 0;
  }

  h1 {
    margin: 0 0 var(--wa-space-s);
  }

  h2 {
    margin: 0 0 var(--wa-space-m);
  }

  header p {
    margin: 0;
  }

  .username {
    font-weight: var(--wa-font-weight-bold);
    overflow-wrap: anywhere;
  }

  wa-radio-group {
    max-width: 24rem;
  }

  wa-radio {
    flex: 1;
    justify-content: center;
  }

  wa-radio span {
    display: inline-flex;
    align-items: center;
    gap: var(--wa-space-xs);
  }
</style>
