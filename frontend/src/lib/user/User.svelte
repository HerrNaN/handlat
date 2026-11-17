<script lang="ts">
  import { createMutation } from "@tanstack/svelte-query";

  import { p } from "../../router";
  import { post } from "../api";
  import PageHeader from "../components/PageHeader.svelte";
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
  <PageHeader title="Profile">
    {#snippet actions()}
      {#if $session}
        <wa-button
          id="logout-button"
          appearance="plain"
          onclick={() => logout.mutate()}
          disabled={logout.isPending}
        >
          <wa-icon name="arrow-right-from-bracket" label="Log out"></wa-icon>
        </wa-button>
        <wa-tooltip for="logout-button">Log out</wa-tooltip>
      {/if}
    {/snippet}
  </PageHeader>
  <div class="content">
    <div>
      {#if $session}
        <p>You are logged in as: <span class="username">{$session.name}</span>.</p>
      {:else}
        <p>You are not logged in.</p>
        <p><a href={p("/login")}>Login or Create Account</a></p>
      {/if}
    </div>
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
  </div>
</article>

<style>
  article {
    height: 100%;
    display: flex;
    flex-direction: column;
    background-color: var(--wa-color-surface-default);
    overflow-y: hidden;
  }

  .content {
    flex: 1;
    min-height: 0;
    display: flex;
    flex-direction: column;
    gap: var(--wa-space-xl);
    padding: var(--wa-space-l);
    overflow-y: auto;
  }

  h2 {
    margin: 0 0 var(--wa-space-m);
  }

  p {
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
