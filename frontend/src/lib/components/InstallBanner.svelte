<script lang="ts">
  import "@awesome.me/webawesome/dist/components/button/button.js";
  import WaDialog from "@awesome.me/webawesome/dist/components/dialog/dialog.js";

  import { messages } from "../i18n";
  import { dismissInstallBanner, installBannerOffer, requestInstall } from "../installPrompt";

  let instructionsDialog = $state<WaDialog>();

  const showInstructions = () => {
    if (instructionsDialog) {
      instructionsDialog.open = true;
    }
  };
</script>

{#if $installBannerOffer}
  <aside aria-labelledby="install-title">
    <div class="message">
      <strong id="install-title">{$messages.install.title}</strong>
      <p>{$messages.install.description}</p>
    </div>
    <div class="actions">
      {#if $installBannerOffer === "native"}
        <wa-button size="s" variant="brand" onclick={requestInstall}>
          {$messages.install.action}
        </wa-button>
      {:else}
        <wa-button size="s" variant="brand" onclick={showInstructions}>
          {$messages.install.instructionsAction}
        </wa-button>
      {/if}
      <wa-button size="s" appearance="plain" onclick={dismissInstallBanner}>
        {$messages.install.dismiss}
      </wa-button>
    </div>
  </aside>

  {#if $installBannerOffer === "ios"}
    <wa-dialog bind:this={instructionsDialog} label={$messages.install.title} light-dismiss>
      <p>{$messages.install.iosInstructions}</p>
      <wa-button slot="footer" data-dialog="close">{$messages.common.done}</wa-button>
    </wa-dialog>
  {/if}
{/if}

<style>
  aside {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: var(--wa-space-xs) var(--wa-space-m);
    flex-shrink: 0;
    padding: var(--wa-space-s) var(--wa-space-l);
    border-block-end: 1px solid var(--wa-color-surface-border);
    background-color: var(--wa-color-surface-raised);
  }

  .message {
    flex: 1 1 14rem;
  }

  .actions {
    display: flex;
    flex-wrap: wrap;
    gap: var(--wa-space-xs);
  }

  p {
    margin: 0;
  }
</style>
