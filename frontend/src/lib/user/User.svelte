<script lang="ts">
  import { createMutation } from "@tanstack/svelte-query";

  import { p } from "../../router";
  import { post } from "../api";
  import PageHeader from "../components/PageHeader.svelte";
  import { languagePreference, messages, setLanguage } from "../i18n";
  import { errorMessage } from "../i18n/errors";
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
  <PageHeader title={$messages.profile.title}>
    {#snippet actions()}
      {#if $session}
        <wa-button
          id="logout-button"
          appearance="plain"
          onclick={() => logout.mutate()}
          disabled={logout.isPending}
        >
          <wa-icon name="arrow-right-from-bracket" label={$messages.profile.logout}></wa-icon>
        </wa-button>
        <wa-tooltip for="logout-button">{$messages.profile.logout}</wa-tooltip>
      {/if}
    {/snippet}
  </PageHeader>
  <div class="content">
    <div>
      {#if $session}
        <p>{$messages.profile.signedInAs($session.name)}</p>
      {:else}
        <p>{$messages.profile.signedOut}</p>
        <p><a href={p("/login")}>{$messages.profile.loginOrCreate}</a></p>
      {/if}
    </div>
    {#if logout.isError}
      <p role="alert">{errorMessage(logout.error, $messages, $messages.errors.logout)}</p>
    {/if}
    <section aria-labelledby="appearance-heading">
      <h2 id="appearance-heading">{$messages.profile.appearance}</h2>
      <wa-radio-group
        label={$messages.profile.theme}
        name="theme"
        orientation="horizontal"
        value={$theme}
        onchange={(event: ChangeEvent) => setTheme(event.target.value)}
      >
        <wa-radio appearance="button" value="light">
          <span><wa-icon name="sun" aria-hidden="true"></wa-icon>{$messages.profile.light}</span>
        </wa-radio>
        <wa-radio appearance="button" value="dark">
          <span><wa-icon name="moon" aria-hidden="true"></wa-icon>{$messages.profile.dark}</span>
        </wa-radio>
        <wa-radio appearance="button" value="system">
          <span
            ><wa-icon name="desktop" aria-hidden="true"></wa-icon>{$messages.profile.system}</span
          >
        </wa-radio>
      </wa-radio-group>
      <wa-radio-group
        label={$messages.profile.language}
        name="language"
        orientation="horizontal"
        value={$languagePreference}
        onchange={(event: ChangeEvent) => setLanguage(event.target.value)}
      >
        <wa-radio appearance="button" value="browser">{$messages.profile.browserLanguage}</wa-radio>
        <wa-radio appearance="button" value="en"><span lang="en">English</span></wa-radio>
        <wa-radio appearance="button" value="sv"><span lang="sv">Svenska</span></wa-radio>
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

  .content p {
    overflow-wrap: anywhere;
  }

  section > wa-radio-group + wa-radio-group {
    margin-top: var(--wa-space-m);
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
