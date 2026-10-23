<script lang="ts">
    import "@awesome.me/webawesome/dist/components/select/select.js";
    import "@awesome.me/webawesome/dist/components/option/option.js";
    import "@awesome.me/webawesome/dist/components/input/input.js";
    import type { Item } from "./list";
    import { listStore } from "./store";
    import { departments } from "./classifier";
    import { type ChangeEvent } from "../utils/events";

    type Props = {
        item: Item;
    };

    const { item }: Props = $props();
    const { status } = listStore;

    const onChangeName = (e: ChangeEvent) =>
        listStore.updateItem(item.id, { text: e.target.value });

    const onChangeDepartement = (e: ChangeEvent) =>
        listStore.updateItem(item.id, { department: e.target.value });
</script>

<section>
    <wa-input value={item.text} label="Name" onchange={onChangeName} disabled={$status.saving || $status.loading}></wa-input>
    <wa-select
        label="Departement"
        value={item.department}
        onchange={onChangeDepartement}
        disabled={$status.saving || $status.loading}
    >
        {#each departments as dep (dep.id)}
            <wa-option value={dep.id}>
                {dep.name}
            </wa-option>
        {/each}
    </wa-select>
</section>

<style>
    section {
        display: flex;
        flex-direction: column;
        gap: var(--wa-space-m);
    }
</style>
