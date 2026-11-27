<script lang="ts">
  import "@awesome.me/webawesome/dist/components/button/button.js";
  import "@awesome.me/webawesome/dist/components/icon/icon.js";
  import { messages } from "../i18n";
  import { dismissInstallBanner, requestInstall, showInstallBanner } from "../installPrompt";
</script>

{#if $showInstallBanner}
  <aside aria-labelledby="install-title">
    <div class="message">
      <strong id="install-title">{$messages.install.title}</strong>
      <p>{$messages.install.description}</p>
    </div>
    <div class="actions">
      <wa-button size="s" variant="brand" onclick={requestInstall}>
        {$messages.install.action}
      </wa-button>
      <wa-button size="s" appearance="plain" onclick={dismissInstallBanner}>
        <wa-icon name="xmark" library="system" label={$messages.install.dismiss}></wa-icon>
      </wa-button>
    </div>
  </aside>
{/if}

<style>
  aside {
    display: flex;
    align-items: center;
    gap: var(--wa-space-m);
    flex-shrink: 0;
    padding: var(--wa-space-s) var(--wa-space-l);
    border-block-end: 1px solid var(--wa-color-surface-border);
    background-color: var(--wa-color-surface-raised);
  }

  .message {
    flex: 1;
    min-inline-size: 0;
    overflow-wrap: anywhere;
  }

  .actions {
    display: flex;
    flex: none;
    gap: var(--wa-space-xs);
  }

  p {
    margin: 0;
  }
</style>
