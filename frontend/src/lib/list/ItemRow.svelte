<script lang="ts">
    import { listStore } from "./store";
    import type { Item } from "./list";
    import SwipeAction from "./SwipeAction.svelte";
    import "@awesome.me/webawesome/dist/components/checkbox/checkbox.js";
    import "@awesome.me/webawesome/dist/components/icon/icon.js";
    import "@awesome.me/webawesome/dist/components/button/button.js";

    type Props = { item: Item; onEditItem: () => void };
    let { item, onEditItem }: Props = $props();
    const { status } = listStore;
    const checked = $derived(item.checkedAt !== undefined);
    const busy = $derived($status.saving || $status.loading);
    let swipe = $state<SwipeAction>();
    const remove = async () => {
        await listStore.deleteItem(item.id);
        swipe?.reset();
    };
</script>

<SwipeAction bind:this={swipe} ontrigger={remove} disabled={busy}>
    {#snippet underlay()}
        <div class="underlay"><wa-icon name="trash"></wa-icon></div>
    {/snippet}
    <div class="content">
        <button
            type="button"
            role="checkbox"
            aria-checked={checked}
            disabled={busy}
            onclick={() => listStore.setItemChecked(item.id, !checked)}
        >
            <!-- The button owns interaction; the checkbox only displays saved state. -->
            <wa-checkbox {checked} defaultChecked={checked} inert aria-hidden="true"></wa-checkbox>
            <span class:checked>{item.text}</span>
        </button>
        <wa-button appearance="plain" onclick={onEditItem} disabled={busy} aria-label="Edit {item.text}">
            <wa-icon name="pen"></wa-icon>
        </wa-button>
    </div>
</SwipeAction>

<style>
    .content {
        position: relative;
        width: 100%;
        height: 100%;
        display: flex;
        align-items: center;
        padding-inline: var(--wa-space-2xs);
        background-color: var(--wa-color-surface-default);
        border-radius: var(--wa-border-width-l);
    }
    button {
        flex: 1;
        min-width: 0;
        height: 100%;
        display: flex;
        align-items: center;
        gap: var(--wa-space-s);
        border: 0;
        padding: 0;
        background: none;
        color: inherit;
        font: inherit;
        text-align: start;
        cursor: pointer;
        -webkit-tap-highlight-color: transparent;
    }
    wa-checkbox {
        pointer-events: none;
        -webkit-tap-highlight-color: transparent;
        padding: 0.675rem;
        border-radius: var(--wa-border-width-l);
        background-color: var(--wa-color-neutral-fill-quiet);

        &:state(checked) {
            background-color: var(--wa-color-success-fill-quiet);

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
    .checked {
        text-decoration: line-through;
        color: var(--wa-color-neutral-on-quiet);
    }
    .underlay {
        position: absolute;
        inset: 0;
        display: flex;
        align-items: center;
        justify-content: end;
        padding: var(--wa-space-s) var(--wa-space-m);
        background-color: var(--wa-color-danger-fill-loud);
        color: var(--wa-color-brand-on-loud);
        border-radius: var(--wa-border-width-l);
        border-style: var(--wa-border-style);
        border-width: var(--wa-border-width-s);
    }
</style>
