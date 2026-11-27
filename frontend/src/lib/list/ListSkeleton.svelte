<script lang="ts">
  import { messages } from "../i18n";
  import "@awesome.me/webawesome/dist/components/skeleton/skeleton.js";

  let { overview = false, label }: { overview?: boolean; label?: string } = $props();
</script>

<div
  role="status"
  aria-label={label ?? (overview ? $messages.lists.loading : $messages.lists.loadingList)}
>
  <div class:overview aria-hidden="true">
    {#if !overview}
      <wa-skeleton class="department" effect="none"></wa-skeleton>
    {/if}
    {#each [65, 45, 75, 55] as width (width)}
      <div class="row">
        <wa-skeleton class="icon" effect="none"></wa-skeleton>
        <wa-skeleton class="text" style:width={`${width}%`} effect="none"></wa-skeleton>
      </div>
    {/each}
    {#if !overview}
      <wa-skeleton class="add" effect="none"></wa-skeleton>
    {/if}
  </div>
</div>

<style>
  div[role="status"] {
    flex: 1;
    min-height: 0;
    overflow: hidden;
    display: flex;
    visibility: hidden;
    animation: show-loading 0s 200ms forwards;
  }

  @keyframes show-loading {
    to {
      visibility: visible;
    }
  }

  div[aria-hidden] {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: var(--wa-space-2xs);
    padding: var(--wa-space-2xs);
  }

  .department {
    width: 6rem;
    height: 0.75rem;
    margin: var(--wa-space-xs) var(--wa-space-m);
  }

  .row {
    display: flex;
    align-items: center;
    gap: var(--wa-space-s);
    flex-shrink: 0;
    height: 3rem;
    padding-inline: var(--wa-space-m);
    background-color: var(--handlat-row-background);
    border-radius: var(--wa-border-radius-s);
    outline: var(--handlat-row-outline);
    outline-offset: -1px;
  }

  .overview .row {
    height: auto;
    padding-block: var(--wa-space-m);
  }

  .icon {
    flex-shrink: 0;
    width: 1.25rem;
    height: 1.25rem;
    border-radius: var(--wa-border-radius-s);
  }

  .text {
    height: 1rem;
  }

  .add {
    flex-shrink: 0;
    width: 3.5rem;
    height: 3.5rem;
    border-radius: var(--wa-border-radius-m);
    margin: auto var(--wa-space-m) var(--wa-space-m);
    align-self: flex-end;
  }
</style>
