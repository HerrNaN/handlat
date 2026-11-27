<script lang="ts">
  import { language, messages } from "../i18n";
  import type { Item } from "./list";
  import { formatQuantity } from "./quantity";
  import SwipeAction from "./SwipeAction.svelte";
  import "@awesome.me/webawesome/dist/components/checkbox/checkbox.js";
  import "@awesome.me/webawesome/dist/components/icon/icon.js";

  type Props = {
    item: Item;
    onEditItem: () => void;
    onCheck: (itemID: string, checked: boolean) => void;
    onDelete: (itemID: string) => Promise<unknown>;
  };
  let { item, onEditItem, onCheck, onDelete }: Props = $props();
  const quantity = $derived(
    formatQuantity(item.amount, item.unit && $messages.units[item.unit], $language),
  );
  const label = $derived(quantity ? `${item.text} ${quantity}` : item.text);
  const checked = $derived(item.checkedAt !== undefined);
  let swipe = $state<SwipeAction>();
  const remove = async () => {
    await onDelete(item.id).catch(() => undefined);
    swipe?.reset();
  };
</script>

<SwipeAction bind:this={swipe} ontrigger={remove}>
  {#snippet underlay()}
    <div class="underlay"><wa-icon name="trash"></wa-icon></div>
  {/snippet}
  <div class="content">
    <button
      type="button"
      role="checkbox"
      aria-label={label}
      aria-checked={checked}
      onclick={() => onCheck(item.id, !checked)}
    >
      <!-- The button owns interaction; the checkbox only displays query state. -->
      <wa-checkbox {checked} defaultChecked={checked} inert aria-hidden="true"></wa-checkbox>
    </button>
    <button
      class="edit"
      type="button"
      aria-label={$messages.items.editNamed(label)}
      onclick={onEditItem}
    >
      <span class="label" class:checked>
        {item.text}{#if quantity}
          <span class="quantity">{` ${quantity}`}</span>{/if}
      </span>
    </button>
  </div>
</SwipeAction>

<style>
  .content {
    position: relative;
    width: 100%;
    min-height: 3rem;
    display: grid;
    grid-template-columns: var(--wa-space-2xs) auto minmax(0, 1fr);
    background-color: var(--handlat-row-background);
    border-radius: var(--wa-border-radius-s);
  }
  button {
    display: flex;
    align-items: center;
    border: 0;
    padding: 0;
    background: none;
    color: inherit;
    font: inherit;
    text-align: start;
    cursor: pointer;
    -webkit-tap-highlight-color: transparent;
  }
  button[role="checkbox"] {
    grid-area: 1 / 2;
    align-self: center;
    z-index: 1;
  }
  .edit {
    grid-area: 1 / 1 / 2 / -1;
    display: grid;
    grid-template-columns: subgrid;
  }
  wa-checkbox {
    pointer-events: none;
    -webkit-tap-highlight-color: transparent;
    padding: 0.675rem;
    border-radius: var(--wa-border-radius-s);
    background-color: var(
      --handlat-checkbox-padding-background,
      var(--wa-color-neutral-fill-quiet)
    );

    &:state(checked) {
      background-color: var(
        --handlat-checked-checkbox-padding-background,
        var(--wa-color-success-fill-quiet)
      );

      &::part(control) {
        color: var(--wa-color-success-on-loud);
        background-color: var(--wa-color-success-fill-loud);
        border-color: var(--wa-color-success-fill-loud);
      }
    }

    &::part(control) {
      margin-inline-end: 0;
    }
  }
  .label {
    grid-column: 3;
    font-weight: var(--wa-font-weight-semibold);
    min-width: 0;
    overflow-wrap: anywhere;
    padding-block: var(--wa-space-xs);
    padding-inline: var(--wa-space-s);
  }
  .quantity {
    white-space: nowrap;
    color: var(--wa-color-text-quiet);
    font-weight: var(--wa-font-weight-normal);
  }
  .checked {
    text-decoration: line-through;
    color: var(--wa-color-text-quiet);
  }
  .underlay {
    position: absolute;
    inset: 0;
    display: flex;
    align-items: center;
    justify-content: end;
    padding: var(--wa-space-s) var(--wa-space-m);
    background-color: var(--wa-color-danger-fill-loud);
    color: var(--wa-color-danger-on-loud);
    border-radius: var(--wa-border-radius-s);
    border-style: var(--wa-border-style);
    border-width: var(--wa-border-width-s);
  }
</style>
