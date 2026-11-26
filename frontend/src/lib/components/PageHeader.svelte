<script lang="ts">
  import type { Snippet } from "svelte";

  let {
    title,
    leading,
    actions,
    titleContent,
  }: {
    title: string;
    leading?: Snippet;
    actions?: Snippet;
    titleContent?: Snippet;
  } = $props();
</script>

<header>
  {#if leading}
    <div>{@render leading()}</div>
  {/if}
  <h1 title={title || undefined}>
    {#if titleContent}
      {@render titleContent()}
    {:else}
      {title}
    {/if}
  </h1>
  {#if actions}
    <div>{@render actions()}</div>
  {/if}
</header>

<style>
  header {
    box-sizing: border-box;
    min-height: 4rem;
    flex-shrink: 0;
    z-index: 1;
    position: sticky;
    top: 0;
    display: flex;
    align-items: center;
    gap: var(--wa-space-xs);
    padding: var(--wa-space-m) var(--wa-space-l);
    background-color: var(--handlat-header-background);
    border-block-end: 1px solid var(--wa-color-surface-border);
  }

  h1 {
    flex: 1;
    min-width: 0;
    margin: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    font-size: var(--wa-font-size-xl);
    font-weight: var(--wa-font-weight-normal);
    color: var(--wa-color-text-normal);
  }

  div {
    flex-shrink: 0;
    display: flex;
    align-items: center;
  }
</style>
