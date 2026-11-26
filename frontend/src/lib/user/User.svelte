<script lang="ts">
  import WaCheckbox from "@awesome.me/webawesome/dist/components/checkbox/checkbox.js";
  import WaDialog from "@awesome.me/webawesome/dist/components/dialog/dialog.js";
  import { createMutation } from "@tanstack/svelte-query";

  import { p } from "../../router";
  import { post } from "../api";
  import PageHeader from "../components/PageHeader.svelte";
  import { languagePreference, messages, setLanguage } from "../i18n";
  import { errorMessage } from "../i18n/errors";
  import { installOffer, requestInstall } from "../installPrompt";
  import { queryClient } from "../query";
  import { enableRecipes, recipesKey } from "../recipe/api";
  import { setTheme, theme } from "../theme";
  import type { ChangeEvent } from "../utils/events";
  import "@awesome.me/webawesome/dist/components/button/button.js";
  import "@awesome.me/webawesome/dist/components/callout/callout.js";
  import "@awesome.me/webawesome/dist/components/icon/icon.js";
  import "@awesome.me/webawesome/dist/components/tag/tag.js";
  import "@awesome.me/webawesome/dist/components/tooltip/tooltip.js";
  import "@awesome.me/webawesome/dist/components/radio-group/radio-group.js";
  import "@awesome.me/webawesome/dist/components/radio/radio.js";

  import { session, clearSession, markRecipesEnabled } from "./session";

  const appVersion = document.querySelector<HTMLMetaElement>(
    'meta[name="handlat-version"]',
  )?.content;

  const logout = createMutation(() => ({
    mutationKey: ["auth", "logout"],
    mutationFn: () => post("/api/auth/logout"),
    onSuccess: () => {
      clearSession();
      location.assign("/login");
    },
  }));
  let previewDialog = $state<WaDialog>();
  let acknowledged = $state(false);
  const optIn = createMutation(() => ({
    mutationFn: async (account: string) => {
      await enableRecipes();
      markRecipesEnabled(account);
      void queryClient.invalidateQueries({ queryKey: recipesKey(account) });
      if (previewDialog) {
        previewDialog.open = false;
      }
    },
  }));

  const openPreview = () => {
    acknowledged = false;
    optIn.reset();
    if (previewDialog) {
      previewDialog.open = true;
    }
  };
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
        <p>
          {$messages.profile.signedInAs.beforeName}<span class="username">{$session.name}</span
          >{$messages.profile.signedInAs.afterName}
        </p>
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
        <wa-radio appearance="button" value="en"><span lang="en">English</span></wa-radio>
        <wa-radio appearance="button" value="sv"><span lang="sv">Svenska</span></wa-radio>
        <wa-radio appearance="button" value="browser">{$messages.profile.browserLanguage}</wa-radio>
      </wa-radio-group>
    </section>
    {#if $installOffer}
      <section aria-labelledby="install-heading">
        <h2 id="install-heading">{$messages.install.title}</h2>
        {#if $installOffer === "native"}
          <div class="install-details">
            <p>{$messages.install.description}</p>
            <wa-button variant="brand" onclick={requestInstall}>
              {$messages.install.action}
            </wa-button>
          </div>
        {:else}
          <p>{$messages.install.iosInstructions}</p>
        {/if}
      </section>
    {/if}
    {#if $session}
      <section aria-labelledby="experimental-features-heading">
        <h2 id="experimental-features-heading">{$messages.profile.experimentalFeatures}</h2>
        <div class="preview-feature">
          <div class="preview-feature-heading">
            <h3>{$messages.nav.recipes}</h3>
            {#if $session.recipesEnabled}
              <wa-tag variant="success" appearance="filled" size="s" role="status">
                <wa-icon name="check" library="system" aria-hidden="true"></wa-icon>
                {$messages.recipes.enabled}
              </wa-tag>
            {:else}
              <wa-button variant="warning" appearance="filled-outlined" onclick={openPreview}>
                <wa-icon slot="start" name="flask" aria-hidden="true"></wa-icon>
                {$messages.recipes.enable}
              </wa-button>
            {/if}
          </div>
          <p>{$messages.recipes.description}</p>
        </div>
      </section>
      <wa-dialog
        bind:this={previewDialog}
        label={$messages.recipes.enableTitle}
        light-dismiss
        onwa-hide={(event: Event) => {
          if (event.target === previewDialog && optIn.isPending && !$session?.recipesEnabled) {
            event.preventDefault();
          }
        }}
      >
        <div class="preview-opt-in">
          <wa-callout variant="warning">
            {$messages.recipes.disclosure}
          </wa-callout>
          <wa-checkbox
            checked={acknowledged}
            disabled={optIn.isPending}
            onchange={(event: Event) => {
              acknowledged = (event.currentTarget as WaCheckbox).checked;
            }}
          >
            {$messages.recipes.acknowledge}
          </wa-checkbox>
          {#if optIn.isError}
            <p role="alert">{$messages.recipes.enableFailed}</p>
          {/if}
        </div>
        <wa-button
          slot="footer"
          variant="warning"
          disabled={!acknowledged || optIn.isPending}
          loading={optIn.isPending}
          onclick={() => $session && optIn.mutate($session.id)}
        >
          {$messages.recipes.enable}
        </wa-button>
      </wa-dialog>
    {/if}
    {#if appVersion}
      <section aria-labelledby="about-heading">
        <h2 id="about-heading">{$messages.profile.about}</h2>
        <dl>
          <dt>{$messages.profile.version}</dt>
          <dd>{appVersion}</dd>
        </dl>
      </section>
    {/if}
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

  .install-details {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: var(--wa-space-s) var(--wa-space-m);
  }

  .install-details > wa-button {
    margin-inline-start: auto;
  }

  dl {
    display: grid;
    grid-template-columns: max-content minmax(0, 1fr);
    column-gap: var(--wa-space-xs);
    margin: 0;
  }

  dd {
    margin: 0;
    overflow-wrap: anywhere;
  }

  .username {
    font-weight: var(--wa-font-weight-bold);
    overflow-wrap: anywhere;
  }

  section > wa-radio-group + wa-radio-group {
    margin-top: var(--wa-space-m);
  }

  wa-radio-group {
    max-width: 24rem;
  }

  .preview-feature {
    border: 1px solid var(--wa-color-surface-border);
    border-radius: var(--wa-border-radius-m);
    padding: var(--wa-space-m);
    padding-block-end: var(--wa-space-l);
  }

  .preview-feature-heading {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: var(--wa-space-s);
    margin-block-end: var(--wa-space-xs);
  }

  .preview-feature-heading > :is(wa-button, wa-tag) {
    margin-inline-start: auto;
  }

  h3 {
    margin: 0;
  }

  wa-tag::part(content) {
    display: inline-flex;
    align-items: center;
    gap: var(--wa-space-2xs);
  }

  wa-tag wa-icon {
    translate: 0 -0.1em;
  }

  .preview-opt-in {
    display: grid;
    gap: var(--wa-space-m);
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
