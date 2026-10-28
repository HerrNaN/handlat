<script lang="ts">
    import "@awesome.me/webawesome/dist/components/button/button.js";
    import ItemRow from "./ItemRow.svelte";
    import { departmentOrder, getDepartment } from "./classifier";
    import type { Item } from "./list";

    interface Props {
        items: Item[];
        onClickEditItem: (item: Item) => void;
        pendingItems: Set<string>;
        clearing: boolean;
        onCheck: (itemID: string, checked: boolean) => void;
        onDelete: (itemID: string) => Promise<unknown>;
        onClear: () => void;
    }

    let { items: itemList, onClickEditItem, pendingItems, clearing, onCheck, onDelete, onClear }: Props = $props();

    const checkedItems = $derived(
        itemList
            .filter(
                (item): item is Item & { checkedAt: Date } =>
                    item.checkedAt !== undefined,
            )
            .sort((a, b) => b.checkedAt.getTime() - a.checkedAt.getTime()),
    );
</script>

{#each departmentOrder as deptId (deptId)}
    {@const deptItems = itemList.filter(
        (item) =>
            item.checkedAt === undefined &&
            (item.department ?? "other") === deptId,
    )}
    {#if deptItems.length > 0}
        <h2>{getDepartment(deptId).name}</h2>
        <ul>
            {#each deptItems as item (item.id)}
                <li>
                    <ItemRow {item} busy={pendingItems.has(item.id)} {onCheck} {onDelete} onEditItem={() => onClickEditItem(item)} />
                </li>
            {/each}
        </ul>
    {/if}
{/each}

{#if checkedItems.length > 0}
    <div class="checked-section">
        <h2>
            Handlat
            <wa-button
                size="s"
                variant="brand"
                appearance="plain"
                onclick={onClear}
                disabled={clearing}
            >
                Clear
            </wa-button>
        </h2>
        <ul>
            {#each checkedItems as item (item.id)}
                <li>
                    <ItemRow {item} busy={pendingItems.has(item.id)} {onCheck} {onDelete} onEditItem={() => onClickEditItem(item)} />
                </li>
            {/each}
        </ul>
    </div>
{/if}

<style>
    ul {
        list-style: none;
        margin: 0;
        padding: 0;
        display: flex;
        flex-direction: column;
        gap: var(--wa-space-2xs);
        padding-block: var(--wa-space-2xs);
    }

    li {
        height: 3rem;
        margin-inline: var(--wa-space-2xs);
    }

    .checked-section {
        display: flex;
        flex-direction: column;
        gap: var(--wa-space-2xs);
    }

    h2 {
        display: flex;
        align-items: baseline;
        justify-content: space-between;
        margin-block: var(--wa-space-xs);
        padding: 0 var(--wa-space-m);
        font-size: var(--wa-font-size-xs);
        text-transform: uppercase;
        color: var(--wa-color-neutral-on-normal);
    }

    wa-button {
        -webkit-tap-highlight-color: transparent;
    }
</style>
