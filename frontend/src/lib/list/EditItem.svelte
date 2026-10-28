<script lang="ts">
    import "@awesome.me/webawesome/dist/components/select/select.js";
    import "@awesome.me/webawesome/dist/components/option/option.js";
    import "@awesome.me/webawesome/dist/components/input/input.js";
    import type { Item } from "./list";
    import type { ItemChanges } from "./api";
    import { departments } from "./classifier";
    import { type ChangeEvent } from "../utils/events";

    type Props = {
        item: Item;
        busy: boolean;
        onUpdate: (itemID: string, changes: ItemChanges) => void;
    };

    let { item, busy, onUpdate }: Props = $props();

    const onChangeName = (e: ChangeEvent) =>
        onUpdate(item.id, { text: e.target.value });

    const onChangeDepartement = (e: ChangeEvent) =>
        onUpdate(item.id, { department: e.target.value });
</script>

<section>
    <wa-input value={item.text} label="Name" onchange={onChangeName} disabled={busy}></wa-input>
    <wa-select
        label="Departement"
        value={item.department}
        onchange={onChangeDepartement}
        disabled={busy}
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
